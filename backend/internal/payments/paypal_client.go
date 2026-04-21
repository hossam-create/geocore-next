package payments

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/ops"
)

type payPalClient struct {
	clientID     string
	clientSecret string
	baseURL      string
	webhookID    string
	httpClient   *http.Client
}

type payPalOrderResult struct {
	OrderID     string
	ApprovalURL string
}

type payPalWebhookHeaders struct {
	TransmissionID   string
	TransmissionTime string
	CertURL          string
	AuthAlgo         string
	TransmissionSig  string
}

type payPalCaptureResult struct {
	OrderID   string
	CaptureID string
	Status    string
}

var ppClient *payPalClient

func InitPayPal() {
	clientID := strings.TrimSpace(ops.ConfigGet("PAYPAL_CLIENT_ID"))
	clientSecret := strings.TrimSpace(ops.ConfigGet("PAYPAL_CLIENT_SECRET"))
	if clientID == "" || clientSecret == "" {
		slog.Warn("PAYPAL_CLIENT_ID or PAYPAL_CLIENT_SECRET not set - PayPal payments unavailable")
		ppClient = nil
		return
	}

	baseURL := strings.TrimSpace(ops.ConfigGet("PAYPAL_BASE_URL"))
	if baseURL == "" {
		if strings.EqualFold(strings.TrimSpace(ops.ConfigGet("APP_ENV")), "production") {
			baseURL = "https://api-m.paypal.com"
		} else {
			baseURL = "https://api-m.sandbox.paypal.com"
		}
	}

	ppClient = &payPalClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      strings.TrimRight(baseURL, "/"),
		webhookID:    strings.TrimSpace(ops.ConfigGet("PAYPAL_WEBHOOK_ID")),
		httpClient:   &http.Client{Timeout: 20 * time.Second},
	}
	slog.Info("PayPal initialized", "base_url", ppClient.baseURL)
}

func createPayPalOrder(amount float64, currency, description, returnURL, cancelURL string) (*payPalOrderResult, error) {
	if ppClient == nil {
		return nil, fmt.Errorf("paypal not configured")
	}
	token, err := ppClient.accessToken()
	if err != nil {
		return nil, err
	}

	applicationContext := map[string]string{
		"user_action": "PAY_NOW",
	}
	if strings.TrimSpace(returnURL) != "" {
		applicationContext["return_url"] = returnURL
	}
	if strings.TrimSpace(cancelURL) != "" {
		applicationContext["cancel_url"] = cancelURL
	}

	body := map[string]any{
		"intent": "CAPTURE",
		"purchase_units": []map[string]any{
			{
				"description": description,
				"amount": map[string]string{
					"currency_code": strings.ToUpper(currency),
					"value":         fmt.Sprintf("%.2f", amount),
				},
			},
		},
		"application_context": applicationContext,
	}

	buf, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, ppClient.baseURL+"/v2/checkout/orders", bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("paypal create order request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ppClient.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("paypal create order: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("paypal create order failed: status=%d body=%s", resp.StatusCode, string(raw))
	}

	var out struct {
		ID    string `json:"id"`
		Links []struct {
			Rel  string `json:"rel"`
			Href string `json:"href"`
		} `json:"links"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("paypal create order decode: %w", err)
	}

	approvalURL := ""
	for _, l := range out.Links {
		if l.Rel == "approve" {
			approvalURL = l.Href
			break
		}
	}
	if out.ID == "" {
		return nil, fmt.Errorf("paypal create order returned empty id")
	}

	return &payPalOrderResult{OrderID: out.ID, ApprovalURL: approvalURL}, nil
}

func capturePayPalOrder(orderID string) (*payPalCaptureResult, error) {
	if ppClient == nil {
		return nil, fmt.Errorf("paypal not configured")
	}
	token, err := ppClient.accessToken()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, ppClient.baseURL+"/v2/checkout/orders/"+orderID+"/capture", bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, fmt.Errorf("paypal capture request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ppClient.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("paypal capture order: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("paypal capture failed: status=%d body=%s", resp.StatusCode, string(raw))
	}

	var out struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		PurchaseUnits []struct {
			Payments struct {
				Captures []struct {
					ID string `json:"id"`
				} `json:"captures"`
			} `json:"payments"`
		} `json:"purchase_units"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("paypal capture decode: %w", err)
	}

	captureID := ""
	if len(out.PurchaseUnits) > 0 && len(out.PurchaseUnits[0].Payments.Captures) > 0 {
		captureID = out.PurchaseUnits[0].Payments.Captures[0].ID
	}

	return &payPalCaptureResult{OrderID: out.ID, CaptureID: captureID, Status: out.Status}, nil
}

func verifyPayPalWebhook(headers payPalWebhookHeaders, rawBody []byte) error {
	if ppClient == nil {
		return fmt.Errorf("paypal not configured")
	}
	if ppClient.webhookID == "" {
		return fmt.Errorf("PAYPAL_WEBHOOK_ID not configured")
	}
	token, err := ppClient.accessToken()
	if err != nil {
		return err
	}

	body := map[string]any{
		"transmission_id":   headers.TransmissionID,
		"transmission_time": headers.TransmissionTime,
		"cert_url":          headers.CertURL,
		"auth_algo":         headers.AuthAlgo,
		"transmission_sig":  headers.TransmissionSig,
		"webhook_id":        ppClient.webhookID,
		"webhook_event":     json.RawMessage(rawBody),
	}

	buf, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, ppClient.baseURL+"/v1/notifications/verify-webhook-signature", bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("paypal verify webhook request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ppClient.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("paypal verify webhook: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("paypal verify webhook failed: status=%d body=%s", resp.StatusCode, string(raw))
	}

	var out struct {
		VerificationStatus string `json:"verification_status"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("paypal verify webhook decode: %w", err)
	}
	if !strings.EqualFold(out.VerificationStatus, "SUCCESS") {
		return fmt.Errorf("paypal webhook verification failed: status=%s", out.VerificationStatus)
	}

	return nil
}

func (p *payPalClient) accessToken() (string, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequest(http.MethodPost, p.baseURL+"/v1/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("paypal token request: %w", err)
	}
	req.SetBasicAuth(p.clientID, p.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("paypal token: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("paypal token failed: status=%d body=%s", resp.StatusCode, string(raw))
	}

	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("paypal token decode: %w", err)
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("paypal token response missing access_token")
	}
	return out.AccessToken, nil
}
