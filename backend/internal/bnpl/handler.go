package bnpl

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

type Handler struct{ db *gorm.DB }

func NewHandler(db *gorm.DB) *Handler { return &Handler{db: db} }

// ── Shared types ──────────────────────────────────────────────────────────────

type BNPLProvider string

const (
	ProviderTamara BNPLProvider = "tamara"
	ProviderTabby  BNPLProvider = "tabby"
)

// ── Tamara ────────────────────────────────────────────────────────────────────

type tamaraCreateReq struct {
	OrderReferenceID string         `json:"order_reference_id"`
	OrderNumber      string         `json:"order_number"`
	Total            tamaraAmount   `json:"total_amount"`
	Description      string         `json:"description"`
	CountryCode      string         `json:"country_code"`
	PaymentType      string         `json:"payment_type"`
	Instalments      int            `json:"instalments"`
	Items            []tamaraItem   `json:"items"`
	Consumer         tamaraConsumer `json:"consumer"`
	MerchantURL      tamaraMerchant `json:"merchant_url"`
}

type tamaraAmount struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type tamaraItem struct {
	Name        string       `json:"name"`
	ReferenceID string       `json:"reference_id"`
	Type        string       `json:"type"`
	Quantity    int          `json:"quantity"`
	UnitPrice   tamaraAmount `json:"unit_price"`
	TotalAmount tamaraAmount `json:"total_amount"`
}

type tamaraConsumer struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
}

type tamaraMerchant struct {
	Success      string `json:"success"`
	Failure      string `json:"failure"`
	Cancel       string `json:"cancel"`
	Notification string `json:"notification"`
}

// ── Tabby ──────────────────────────────────────────────────────────────────────

type tabbyCreateReq struct {
	Payment tabbyPayment `json:"payment"`
}

type tabbyPayment struct {
	Amount       string     `json:"amount"`
	Currency     string     `json:"currency"`
	Description  string     `json:"description"`
	Buyer        tabbyBuyer `json:"buyer"`
	Order        tabbyOrder `json:"order"`
	MerchantURLs tabbyURLs  `json:"merchant_urls"`
}

type tabbyBuyer struct {
	Phone string `json:"phone"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type tabbyOrder struct {
	TaxAmount      string      `json:"tax_amount"`
	ShippingAmount string      `json:"shipping_amount"`
	Discount       string      `json:"discount_amount"`
	UpdatedAt      string      `json:"updated_at"`
	ReferenceID    string      `json:"reference_id"`
	Items          []tabbyItem `json:"items"`
}

type tabbyItem struct {
	Title       string `json:"title"`
	Quantity    int    `json:"quantity"`
	UnitPrice   string `json:"unit_price"`
	ReferenceID string `json:"reference_id"`
	ImageURL    string `json:"image_url"`
	ProductURL  string `json:"product_url"`
	Category    string `json:"category"`
}

type tabbyURLs struct {
	Success string `json:"success"`
	Cancel  string `json:"cancel"`
	Failure string `json:"failure"`
}

// ── Request/Response for our API ──────────────────────────────────────────────

type CreateBNPLReq struct {
	Provider    BNPLProvider `json:"provider" binding:"required"`
	Amount      float64      `json:"amount" binding:"required,gt=0"`
	Currency    string       `json:"currency" binding:"required"`
	Description string       `json:"description"`
	ListingID   string       `json:"listing_id"`
	AuctionID   string       `json:"auction_id"`
	Instalments int          `json:"instalments"` // 3, 4, 6
	ReturnURL   string       `json:"return_url" binding:"required"`
	CancelURL   string       `json:"cancel_url" binding:"required"`
}

// POST /api/v1/bnpl/create
func (h *Handler) Create(c *gin.Context) {
	var req CreateBNPLReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID, _ := c.Get("userID")
	userIDStr := c.GetString("user_id")
	if userIDStr == "" && userID != nil {
		userIDStr = fmt.Sprintf("%v", userID)
	}
	userEmail, _ := c.Get("userEmail")
	if userEmail == nil {
		userEmail = ""
	}

	refID := uuid.New().String()
	instalments := req.Instalments
	if instalments == 0 {
		instalments = 3
	}

	h.createPendingBNPLPayment(userIDStr, req, refID)

	switch req.Provider {
	case ProviderTamara:
		h.createTamara(c, req, fmt.Sprintf("%v", userID), fmt.Sprintf("%v", userEmail), refID, instalments)
	case ProviderTabby:
		h.createTabby(c, req, fmt.Sprintf("%v", userID), fmt.Sprintf("%v", userEmail), refID)
	default:
		response.BadRequest(c, "provider must be 'tamara' or 'tabby'")
	}
}

// GET /api/v1/bnpl/providers — returns available providers based on env config
func (h *Handler) Providers(c *gin.Context) {
	type providerInfo struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Logo        string  `json:"logo"`
		Tagline     string  `json:"tagline"`
		Available   bool    `json:"available"`
		Instalments []int   `json:"instalments"`
		MinAmount   float64 `json:"min_amount"`
		MaxAmount   float64 `json:"max_amount"`
	}

	providers := []providerInfo{
		{
			ID:          "tamara",
			Name:        "Tamara",
			Logo:        "https://cdn.tamara.co/assets/images/tamara-logo.svg",
			Tagline:     "Buy now, pay in 3 or 4 installments",
			Available:   os.Getenv("TAMARA_API_KEY") != "",
			Instalments: []int{3, 4},
			MinAmount:   50,
			MaxAmount:   5000,
		},
		{
			ID:          "tabby",
			Name:        "Tabby",
			Logo:        "https://cdn.tabby.ai/assets/tabby-logo.png",
			Tagline:     "Pay in 4 interest-free installments",
			Available:   os.Getenv("TABBY_PUBLIC_KEY") != "" && os.Getenv("TABBY_SECRET_KEY") != "",
			Instalments: []int{4},
			MinAmount:   10,
			MaxAmount:   10000,
		},
	}

	response.OK(c, providers)
}

// ── Tamara implementation ─────────────────────────────────────────────────────

func (h *Handler) createTamara(c *gin.Context, req CreateBNPLReq, userID, email, refID string, instalments int) {
	apiKey := os.Getenv("TAMARA_API_KEY")
	apiURL := "https://api-sandbox.tamara.co/checkout" // sandbox by default
	if os.Getenv("APP_ENV") == "production" {
		apiURL = "https://api.tamara.co/checkout"
	}

	if apiKey == "" {
		// Return a mock redirect in non-production so UI can be tested
		response.OK(c, gin.H{
			"provider":     "tamara",
			"checkout_url": req.ReturnURL + "?bnpl=tamara&status=simulated&ref=" + refID,
			"reference_id": refID,
			"simulated":    true,
		})
		return
	}

	itemDesc := req.Description
	if itemDesc == "" {
		itemDesc = "GeoCore Purchase"
	}

	body := tamaraCreateReq{
		OrderReferenceID: refID,
		OrderNumber:      refID[:8],
		Total:            tamaraAmount{Amount: req.Amount, Currency: req.Currency},
		Description:      itemDesc,
		CountryCode:      "SA",
		PaymentType:      "PAY_BY_INSTALMENTS",
		Instalments:      instalments,
		Items: []tamaraItem{
			{
				Name:        itemDesc,
				ReferenceID: req.ListingID,
				Type:        "Physical",
				Quantity:    1,
				UnitPrice:   tamaraAmount{Amount: req.Amount, Currency: req.Currency},
				TotalAmount: tamaraAmount{Amount: req.Amount, Currency: req.Currency},
			},
		},
		Consumer: tamaraConsumer{
			FirstName:   "GeoCore",
			LastName:    "Buyer",
			Email:       email,
			PhoneNumber: "+966500000000",
		},
		MerchantURL: tamaraMerchant{
			Success:      req.ReturnURL + "?bnpl=tamara&status=success&ref=" + refID,
			Failure:      req.CancelURL + "?bnpl=tamara&status=failure",
			Cancel:       req.CancelURL + "?bnpl=tamara&status=cancel",
			Notification: os.Getenv("BACKEND_URL") + "/api/v1/bnpl/tamara/webhook",
		},
	}

	b, _ := json.Marshal(body)
	httpReq, _ := http.NewRequest("POST", apiURL, bytes.NewReader(b))
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil || resp.StatusCode >= 400 {
		response.InternalError(c, fmt.Errorf("tamara API error"))
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	checkoutURL, _ := result["checkout_url"].(string)
	response.OK(c, gin.H{
		"provider":     "tamara",
		"checkout_url": checkoutURL,
		"reference_id": refID,
	})
}

// ── Tabby implementation ──────────────────────────────────────────────────────

func (h *Handler) createTabby(c *gin.Context, req CreateBNPLReq, userID, email, refID string) {
	secretKey := os.Getenv("TABBY_SECRET_KEY")
	apiURL := "https://api.tabby.ai/api/v2/checkout"

	if secretKey == "" {
		response.OK(c, gin.H{
			"provider":     "tabby",
			"checkout_url": req.ReturnURL + "?bnpl=tabby&status=simulated&ref=" + refID,
			"reference_id": refID,
			"simulated":    true,
		})
		return
	}

	itemDesc := req.Description
	if itemDesc == "" {
		itemDesc = "GeoCore Purchase"
	}

	body := tabbyCreateReq{
		Payment: tabbyPayment{
			Amount:      fmt.Sprintf("%.2f", req.Amount),
			Currency:    req.Currency,
			Description: itemDesc,
			Buyer: tabbyBuyer{
				Email: email,
				Name:  "GeoCore Buyer",
				Phone: "+971500000000",
			},
			Order: tabbyOrder{
				TaxAmount:      "0.00",
				ShippingAmount: "0.00",
				Discount:       "0.00",
				UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
				ReferenceID:    refID,
				Items: []tabbyItem{
					{
						Title:       itemDesc,
						Quantity:    1,
						UnitPrice:   fmt.Sprintf("%.2f", req.Amount),
						ReferenceID: req.ListingID,
						Category:    "general",
					},
				},
			},
			MerchantURLs: tabbyURLs{
				Success: req.ReturnURL + "?bnpl=tabby&status=success&ref=" + refID,
				Cancel:  req.CancelURL + "?bnpl=tabby&status=cancel",
				Failure: req.CancelURL + "?bnpl=tabby&status=failure",
			},
		},
	}

	b, _ := json.Marshal(body)
	httpReq, _ := http.NewRequest("POST", apiURL, bytes.NewReader(b))
	httpReq.Header.Set("Authorization", "Bearer "+secretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil || resp.StatusCode >= 400 {
		response.InternalError(c, fmt.Errorf("tabby API error"))
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Tabby returns configuration.available_products.installments[0].web_url
	var checkoutURL string
	if cfg, ok := result["configuration"].(map[string]interface{}); ok {
		if prods, ok := cfg["available_products"].(map[string]interface{}); ok {
			if inst, ok := prods["installments"].([]interface{}); ok && len(inst) > 0 {
				if first, ok := inst[0].(map[string]interface{}); ok {
					checkoutURL, _ = first["web_url"].(string)
				}
			}
		}
	}

	response.OK(c, gin.H{
		"provider":     "tabby",
		"checkout_url": checkoutURL,
		"reference_id": refID,
	})
}

// POST /api/v1/bnpl/tamara/webhook — Tamara notification callback
func (h *Handler) TamaraWebhook(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 65536))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	secret := os.Getenv("TAMARA_WEBHOOK_SECRET")
	if secret != "" {
		sig := c.GetHeader("X-Tamara-Signature")
		if err := verifyWebhookHMAC(body, sig, secret); err != nil {
			slog.Warn("bnpl tamara webhook: signature verification failed", "error", err.Error(), "ip", c.ClientIP())
			c.Status(http.StatusUnauthorized)
			return
		}
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	applyBNPLOrderUpdate(h.db, payload)
	c.Status(http.StatusOK)
}

// POST /api/v1/bnpl/tabby/webhook — Tabby notification callback
func (h *Handler) TabbyWebhook(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 65536))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	secret := os.Getenv("TABBY_WEBHOOK_SECRET")
	if secret == "" {
		secret = os.Getenv("TABBY_SECRET_KEY")
	}
	if secret != "" {
		sig := c.GetHeader("X-Tabby-Signature")
		if err := verifyWebhookHMAC(body, sig, secret); err != nil {
			slog.Warn("bnpl tabby webhook: signature verification failed", "error", err.Error(), "ip", c.ClientIP())
			c.Status(http.StatusUnauthorized)
			return
		}
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	applyBNPLOrderUpdate(h.db, payload)
	c.Status(http.StatusOK)
}

func verifyWebhookHMAC(payload []byte, signature, secret string) error {
	if signature == "" {
		return fmt.Errorf("missing signature header")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(strings.ToLower(expected)), []byte(strings.ToLower(signature))) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

func applyBNPLOrderUpdate(db *gorm.DB, payload map[string]interface{}) {
	referenceID := firstNonEmpty(
		getString(payload, "order_reference_id"),
		getNestedString(payload, "order", "reference_id"),
		getNestedString(payload, "data", "order_reference_id"),
		getNestedString(payload, "metadata", "reference_id"),
		getString(payload, "reference_id"),
	)
	if referenceID == "" {
		return
	}

	rawStatus := strings.ToLower(firstNonEmpty(
		getString(payload, "status"),
		getString(payload, "payment_status"),
		getString(payload, "event_type"),
		getNestedString(payload, "data", "status"),
		getNestedString(payload, "order", "status"),
	))

	newStatus := ""
	switch {
	case strings.Contains(rawStatus, "approved"), strings.Contains(rawStatus, "captured"), strings.Contains(rawStatus, "authorized"), strings.Contains(rawStatus, "success"), strings.Contains(rawStatus, "paid"):
		newStatus = "confirmed"
	case strings.Contains(rawStatus, "cancel"), strings.Contains(rawStatus, "fail"), strings.Contains(rawStatus, "declin"), strings.Contains(rawStatus, "reject"), strings.Contains(rawStatus, "expire"):
		newStatus = "cancelled"
	default:
		return
	}

	updates := map[string]interface{}{
		"status":     newStatus,
		"updated_at": time.Now(),
	}

	paymentStatus := "pending"
	if newStatus == "confirmed" {
		paymentStatus = "succeeded"
	} else if newStatus == "cancelled" {
		paymentStatus = "cancelled"
	}
	db.Table("payments").
		Where("stripe_payment_intent_id = ?", referenceID).
		Updates(map[string]interface{}{"status": paymentStatus, "updated_at": time.Now()})

	result := db.Table("orders").Where("payment_intent_id = ?", referenceID).Updates(updates)
	if result.Error != nil || result.RowsAffected > 0 {
		return
	}

	if oid, err := uuid.Parse(referenceID); err == nil {
		db.Table("orders").Where("id = ?", oid).Updates(updates)
	}
}

func (h *Handler) createPendingBNPLPayment(userIDStr string, req CreateBNPLReq, referenceID string) {
	uid, err := uuid.Parse(strings.TrimSpace(userIDStr))
	if err != nil {
		return
	}

	method := "bnpl"
	if req.Provider != "" {
		method = "bnpl-" + strings.ToLower(string(req.Provider))
	}

	payment := payments.Payment{
		UserID:                uid,
		Kind:                  payments.PaymentKindPurchase,
		StripePaymentIntentID: referenceID,
		Amount:                req.Amount,
		Currency:              strings.ToUpper(req.Currency),
		Status:                payments.PaymentStatusPending,
		PaymentMethod:         method,
		Description:           req.Description,
	}

	if req.ListingID != "" {
		if id, err := uuid.Parse(req.ListingID); err == nil {
			payment.ListingID = &id
		}
	}
	if req.AuctionID != "" {
		if id, err := uuid.Parse(req.AuctionID); err == nil {
			payment.AuctionID = &id
			payment.Kind = payments.PaymentKindAuctionPayment
		}
	}

	if err := h.db.Where("stripe_payment_intent_id = ?", referenceID).FirstOrCreate(&payment).Error; err != nil {
		slog.Warn("bnpl: failed to persist pending payment", "reference_id", referenceID, "error", err.Error())
	}
}

func getString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func getNestedString(m map[string]interface{}, parent, child string) string {
	pv, ok := m[parent]
	if !ok || pv == nil {
		return ""
	}
	obj, ok := pv.(map[string]interface{})
	if !ok {
		return ""
	}
	return getString(obj, child)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
