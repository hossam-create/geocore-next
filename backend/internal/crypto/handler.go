package crypto

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/payments"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler { return &Handler{db: db} }

type CreateChargeReq struct {
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Currency    string  `json:"currency" binding:"required"`
	Description string  `json:"description"`
	ListingID   string  `json:"listing_id"`
	AuctionID   string  `json:"auction_id"`
	ReturnURL   string  `json:"return_url" binding:"required"`
	CancelURL   string  `json:"cancel_url" binding:"required"`
}

type coinbaseChargeReq struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	PricingType string                 `json:"pricing_type"`
	LocalPrice  coinbaseLocalPrice     `json:"local_price"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	RedirectURL string                 `json:"redirect_url,omitempty"`
	CancelURL   string                 `json:"cancel_url,omitempty"`
}

type coinbaseLocalPrice struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

// GET /api/v1/crypto/providers
func (h *Handler) Providers(c *gin.Context) {
	response.OK(c, []gin.H{
		{
			"id":         "coinbase",
			"name":       "Coinbase Commerce",
			"available":  os.Getenv("COINBASE_COMMERCE_API_KEY") != "",
			"currencies": []string{"BTC", "ETH", "USDC", "USDT"},
		},
	})
}

// POST /api/v1/crypto/create-charge
func (h *Handler) CreateCharge(c *gin.Context) {
	var req CreateChargeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	apiKey := os.Getenv("COINBASE_COMMERCE_API_KEY")
	if apiKey == "" {
		response.OK(c, gin.H{
			"provider":    "coinbase",
			"hosted_url":  req.ReturnURL + "?crypto_status=success&simulated=1",
			"charge_code": "simulated",
			"charge_id":   "simulated",
			"simulated":   true,
		})
		return
	}

	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}
	if req.Description == "" {
		req.Description = "GeoCore Purchase"
	}

	payload := coinbaseChargeReq{
		Name:        "GeoCore Checkout",
		Description: req.Description,
		PricingType: "fixed_price",
		LocalPrice: coinbaseLocalPrice{
			Amount:   fmt.Sprintf("%.2f", req.Amount),
			Currency: currency,
		},
		Metadata: map[string]interface{}{
			"listing_id": req.ListingID,
			"auction_id": req.AuctionID,
			"platform":   "geocore",
		},
		RedirectURL: req.ReturnURL + "?crypto_status=success",
		CancelURL:   req.CancelURL + "?crypto_status=cancelled",
	}

	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequest("POST", "https://api.commerce.coinbase.com/charges", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-CC-Version", "2018-03-22")
	httpReq.Header.Set("X-CC-Api-Key", apiKey)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		response.BadRequest(c, "failed to create coinbase charge")
		return
	}

	var out struct {
		Data struct {
			ID        string `json:"id"`
			Code      string `json:"code"`
			HostedURL string `json:"hosted_url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, gin.H{
		"provider":    "coinbase",
		"hosted_url":  out.Data.HostedURL,
		"charge_code": out.Data.Code,
		"charge_id":   out.Data.ID,
	})

	userIDStr := c.GetString("user_id")
	if userID, err := uuid.Parse(userIDStr); err == nil {
		var listingUUID *uuid.UUID
		if req.ListingID != "" {
			if id, err := uuid.Parse(req.ListingID); err == nil {
				listingUUID = &id
			}
		}

		var auctionUUID *uuid.UUID
		if req.AuctionID != "" {
			if id, err := uuid.Parse(req.AuctionID); err == nil {
				auctionUUID = &id
			}
		}

		kind := payments.PaymentKindPurchase
		if auctionUUID != nil {
			kind = payments.PaymentKindAuctionPayment
		}

		payment := payments.Payment{
			UserID:                userID,
			ListingID:             listingUUID,
			AuctionID:             auctionUUID,
			Kind:                  kind,
			StripePaymentIntentID: out.Data.Code,
			Amount:                req.Amount,
			Currency:              strings.ToUpper(currency),
			Status:                payments.PaymentStatusPending,
			PaymentMethod:         "coinbase",
			Description:           req.Description,
		}

		if err := h.db.Where("stripe_payment_intent_id = ?", out.Data.Code).FirstOrCreate(&payment).Error; err != nil {
			slog.Warn("crypto: failed to persist pending payment for coinbase charge", "charge_code", out.Data.Code, "error", err.Error())
		}
	}
}

// ── Webhook types ────────────────────────────────────────────────────────────

type coinbaseEvent struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type coinbaseWebhookPayload struct {
	Event coinbaseEvent `json:"event"`
}

// verifyCoinbaseSignature checks the X-CC-Webhook-Signature header.
// Coinbase Commerce signs the raw JSON body with HMAC-SHA256 using the
// webhook shared secret, then hex-encodes the digest.
func verifyCoinbaseSignature(payload []byte, sigHeader, secret string) error {
	if sigHeader == "" {
		return fmt.Errorf("missing X-CC-Webhook-Signature header")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(sigHeader)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

// POST /api/v1/crypto/coinbase/webhook
func (h *Handler) CoinbaseWebhook(c *gin.Context) {
	// ── 1. Read raw body (must happen before any body parsing) ──────────────
	payload, err := io.ReadAll(io.LimitReader(c.Request.Body, 65536))
	if err != nil {
		slog.Error("coinbase webhook: failed to read body", "error", err.Error())
		c.Status(http.StatusBadRequest)
		return
	}

	// ── 2. Verify HMAC-SHA256 signature ─────────────────────────────────────
	sigHeader := c.GetHeader("X-CC-Webhook-Signature")
	secret := os.Getenv("COINBASE_COMMERCE_WEBHOOK_SECRET")

	if secret == "" {
		// Warn loudly in logs but allow in dev (no secret configured).
		slog.Warn("coinbase webhook: COINBASE_COMMERCE_WEBHOOK_SECRET not set — skipping signature check",
			"ip", c.ClientIP())
	} else {
		if err := verifyCoinbaseSignature(payload, sigHeader, secret); err != nil {
			slog.Warn("coinbase webhook: signature verification failed",
				"error", err.Error(),
				"ip", c.ClientIP(),
				"sig_header", sigHeader,
			)
			c.Status(http.StatusUnauthorized)
			return
		}
	}

	// ── 3. Parse the event ──────────────────────────────────────────────────
	var whPayload coinbaseWebhookPayload
	if err := json.Unmarshal(payload, &whPayload); err != nil {
		slog.Error("coinbase webhook: failed to parse payload", "error", err.Error())
		c.Status(http.StatusBadRequest)
		return
	}

	event := whPayload.Event
	slog.Info("coinbase webhook received",
		"event_id", event.ID,
		"event_type", event.Type,
		"ip", c.ClientIP(),
	)

	// ── 4. Route by event type ───────────────────────────────────────────────
	switch event.Type {
	case "charge:confirmed":
		h.handleChargeConfirmed(event)
	case "charge:failed":
		h.handleChargeFailed(event)
	case "charge:pending":
		slog.Info("coinbase webhook: charge pending", "event_id", event.ID)
	case "charge:delayed":
		slog.Warn("coinbase webhook: charge delayed", "event_id", event.ID)
	case "charge:resolved":
		slog.Info("coinbase webhook: charge resolved (overpaid/underpaid)", "event_id", event.ID)
	default:
		slog.Debug("coinbase webhook: unhandled event type", "type", event.Type)
	}

	// Always 200 — Coinbase retries with exponential back-off on non-2xx.
	c.Status(http.StatusOK)
}

// handleChargeConfirmed reconciles a confirmed crypto charge.
func (h *Handler) handleChargeConfirmed(event coinbaseEvent) {
	type chargeData struct {
		Code     string `json:"code"`
		Metadata struct {
			ListingID string `json:"listing_id"`
			AuctionID string `json:"auction_id"`
		} `json:"metadata"`
	}
	var charge chargeData
	if err := json.Unmarshal(event.Data, &charge); err != nil {
		slog.Error("coinbase webhook: failed to parse charge:confirmed data",
			"event_id", event.ID, "error", err.Error())
		return
	}
	slog.Info("coinbase webhook: charge confirmed",
		"event_id", event.ID,
		"charge_code", charge.Code,
		"listing_id", charge.Metadata.ListingID,
		"auction_id", charge.Metadata.AuctionID,
	)
	h.updatePaymentByChargeCode(charge.Code, payments.PaymentStatusSucceeded, "")
}

// handleChargeFailed logs a failed crypto charge.
func (h *Handler) handleChargeFailed(event coinbaseEvent) {
	type chargeData struct {
		Code string `json:"code"`
	}
	var charge chargeData
	if err := json.Unmarshal(event.Data, &charge); err != nil {
		slog.Error("coinbase webhook: failed to parse charge:failed data",
			"event_id", event.ID, "error", err.Error())
		return
	}
	slog.Warn("coinbase webhook: charge failed",
		"event_id", event.ID,
		"charge_code", charge.Code,
	)
	h.updatePaymentByChargeCode(charge.Code, payments.PaymentStatusFailed, "Coinbase charge failed")
}

func (h *Handler) updatePaymentByChargeCode(chargeCode string, status payments.PaymentStatus, failureReason string) {
	if strings.TrimSpace(chargeCode) == "" {
		return
	}

	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if status == payments.PaymentStatusFailed && failureReason != "" {
		updates["failure_reason"] = failureReason
	}

	result := h.db.Model(&payments.Payment{}).
		Where("stripe_payment_intent_id = ?", chargeCode).
		Updates(updates)

	if result.Error != nil {
		slog.Error("coinbase webhook: failed updating payment by charge code",
			"charge_code", chargeCode,
			"status", status,
			"error", result.Error.Error(),
		)
		return
	}

	if result.RowsAffected == 0 {
		slog.Warn("coinbase webhook: no local payment found for charge code", "charge_code", chargeCode)
	}
}
