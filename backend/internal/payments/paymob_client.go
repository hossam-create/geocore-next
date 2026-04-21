package payments

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/geocore-next/backend/pkg/circuit"
	"github.com/geocore-next/backend/pkg/retry"
)

// ════════════════════════════════════════════════════════════════════════════
// PayMob API client — MENA payment gateway
// ════════════════════════════════════════════════════════════════════════════

const paymobBaseURL = "https://accept.paymob.com/api"

// PayMobClient wraps HTTP calls to the PayMob API.
type PayMobClient struct {
	apiKey        string
	hmacSecret    string
	integrationID int64
	iframeID      int64
	httpClient    *http.Client
}

// NewPayMobClient creates a client from env vars.
func NewPayMobClient() *PayMobClient {
	intID, _ := strconv.ParseInt(os.Getenv("PAYMOB_INTEGRATION_ID"), 10, 64)
	iframeID, _ := strconv.ParseInt(os.Getenv("PAYMOB_IFRAME_ID"), 10, 64)
	return &PayMobClient{
		apiKey:        os.Getenv("PAYMOB_API_KEY"),
		hmacSecret:    os.Getenv("PAYMOB_HMAC_SECRET"),
		integrationID: intID,
		iframeID:      iframeID,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

// IsConfigured returns true when PayMob credentials are set.
func (c *PayMobClient) IsConfigured() bool {
	return c.apiKey != "" && c.integrationID > 0
}

// authTokenResponse is the response from PayMob auth/token request.
type authTokenResponse struct {
	Token string `json:"token"`
}

// GetAuthToken obtains an authentication token from PayMob.
func (c *PayMobClient) GetAuthToken(ctx context.Context) (string, error) {
	body := map[string]string{"api_key": c.apiKey}
	var resp authTokenResponse
	if err := c.doRequest(ctx, "POST", "/auth/tokens", body, &resp); err != nil {
		return "", fmt.Errorf("paymob auth: %w", err)
	}
	return resp.Token, nil
}

// registerOrderResponse is the response from PayMob order registration.
type registerOrderResponse struct {
	ID int64 `json:"id"`
}

// RegisterOrder creates an order on PayMob and returns the order ID.
func (c *PayMobClient) RegisterOrder(ctx context.Context, authToken string, amountCents int64, currency string, merchantOrderID string) (int64, error) {
	body := map[string]interface{}{
		"auth_token":        authToken,
		"delivery_needed":   false,
		"amount_cents":      amountCents,
		"currency":          currency,
		"merchant_order_id": merchantOrderID,
		"items":             []interface{}{},
	}
	var resp registerOrderResponse
	if err := c.doRequest(ctx, "POST", "/ecommerce/orders", body, &resp); err != nil {
		return 0, fmt.Errorf("paymob register order: %w", err)
	}
	return resp.ID, nil
}

// paymentKeyResponse is the response from PayMob payment key request.
type paymentKeyResponse struct {
	Token string `json:"token"`
}

// GetPaymentKey generates a payment key for the given order.
func (c *PayMobClient) GetPaymentKey(ctx context.Context, authToken string, orderID int64, amountCents int64, currency string, billingData map[string]string) (string, error) {
	if billingData == nil {
		billingData = map[string]string{
			"first_name":   "N/A",
			"last_name":    "N/A",
			"email":        "na@geocore.app",
			"phone_number": "+201000000000",
			"apartment":    "N/A",
			"floor":        "N/A",
			"building":     "N/A",
			"street":       "N/A",
			"city":         "Cairo",
			"country":      "EG",
		}
	}
	body := map[string]interface{}{
		"auth_token":     authToken,
		"amount_cents":   amountCents,
		"expiration":     3600,
		"order_id":       orderID,
		"billing_data":   billingData,
		"currency":       currency,
		"integration_id": c.integrationID,
	}
	var resp paymentKeyResponse
	if err := c.doRequest(ctx, "POST", "/acceptance/payment_keys", body, &resp); err != nil {
		return "", fmt.Errorf("paymob payment key: %w", err)
	}
	return resp.Token, nil
}

// IframeURL returns the PayMob iframe URL for the given payment key.
func (c *PayMobClient) IframeURL(paymentKey string) string {
	if c.iframeID == 0 {
		return fmt.Sprintf("%s/acceptance/iframes/?paymentToken=%s", paymobBaseURL, paymentKey)
	}
	return fmt.Sprintf("%s/acceptance/iframes/%d?paymentToken=%s", paymobBaseURL, c.iframeID, paymentKey)
}

// VerifyHMAC validates the HMAC signature on PayMob webhook payloads.
// This is critical for security — prevents forged webhook calls.
func (c *PayMobClient) VerifyHMAC(payload []byte, receivedHMAC string) bool {
	if c.hmacSecret == "" || receivedHMAC == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(c.hmacSecret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(receivedHMAC), []byte(expectedMAC))
}

// doRequest makes an HTTP request to PayMob API through the circuit breaker
// with retry+backoff for transient failures.
func (c *PayMobClient) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	err := retry.DoWithContextAndJitter(ctx, 3, 2*time.Second, func() error {
		return circuit.PaymentsBreaker.Execute(func(callCtx context.Context) error {
			// Re-create reader for each retry attempt
			var reader io.Reader = bodyReader
			if body != nil {
				b, _ := json.Marshal(body)
				reader = bytes.NewReader(b)
			}

			req, err := http.NewRequestWithContext(callCtx, method, paymobBaseURL+path, reader)
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := c.httpClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				respBody, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("paymob %s: status %d: %s", path, resp.StatusCode, string(respBody))
			}

			if result != nil {
				return json.NewDecoder(resp.Body).Decode(result)
			}
			return nil
		})
	})

	return err
}
