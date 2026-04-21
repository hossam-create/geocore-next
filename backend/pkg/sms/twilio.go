package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/geocore-next/backend/pkg/circuit"
)

// TwilioClient handles SMS and WhatsApp messaging via Twilio
type TwilioClient struct {
	accountSID    string
	authToken     string
	fromPhone     string
	fromWhatsApp  string
	verifyService string
	baseURL       string
	httpClient    *http.Client
}

// NewTwilioClient creates a new Twilio client from environment variables
func NewTwilioClient() *TwilioClient {
	return &TwilioClient{
		accountSID:    os.Getenv("TWILIO_ACCOUNT_SID"),
		authToken:     os.Getenv("TWILIO_AUTH_TOKEN"),
		fromPhone:     os.Getenv("TWILIO_PHONE_NUMBER"),
		fromWhatsApp:  os.Getenv("TWILIO_WHATSAPP_NUMBER"),
		verifyService: os.Getenv("TWILIO_VERIFY_SERVICE_SID"),
		baseURL:       "https://api.twilio.com/2010-04-01",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsConfigured checks if Twilio credentials are set
func (t *TwilioClient) IsConfigured() bool {
	return t.accountSID != "" && t.authToken != ""
}

// SendSMS sends an SMS message (protected by circuit breaker)
func (t *TwilioClient) SendSMS(to, message string) error {
	if !t.IsConfigured() {
		return fmt.Errorf("twilio not configured")
	}

	endpoint := fmt.Sprintf("%s/Accounts/%s/Messages.json", t.baseURL, t.accountSID)

	data := url.Values{}
	data.Set("To", to)
	data.Set("From", t.fromPhone)
	data.Set("Body", message)

	err := circuit.SMSBreaker.Execute(func(ctx context.Context) error {
		return t.sendRequestWithContext(ctx, endpoint, data)
	})
	if err != nil {
		slog.Warn("sms circuit breaker blocked or call failed", "to", to, "error", err)
	}
	return err
}

// SendWhatsApp sends a WhatsApp message
func (t *TwilioClient) SendWhatsApp(to, message string) error {
	if !t.IsConfigured() {
		return fmt.Errorf("twilio not configured")
	}

	endpoint := fmt.Sprintf("%s/Accounts/%s/Messages.json", t.baseURL, t.accountSID)

	// WhatsApp numbers need the whatsapp: prefix
	whatsappTo := to
	if !strings.HasPrefix(to, "whatsapp:") {
		whatsappTo = "whatsapp:" + to
	}
	whatsappFrom := t.fromWhatsApp
	if !strings.HasPrefix(whatsappFrom, "whatsapp:") {
		whatsappFrom = "whatsapp:" + whatsappFrom
	}

	data := url.Values{}
	data.Set("To", whatsappTo)
	data.Set("From", whatsappFrom)
	data.Set("Body", message)

	return t.sendRequest(endpoint, data)
}

// SendOTP sends a verification code via Twilio Verify
func (t *TwilioClient) SendOTP(to, channel string) error {
	if !t.IsConfigured() || t.verifyService == "" {
		return fmt.Errorf("twilio verify not configured")
	}

	endpoint := fmt.Sprintf("https://verify.twilio.com/v2/Services/%s/Verifications", t.verifyService)

	data := url.Values{}
	data.Set("To", to)
	data.Set("Channel", channel) // "sms" or "whatsapp"

	return t.sendRequest(endpoint, data)
}

// VerifyOTP verifies a code sent via Twilio Verify
func (t *TwilioClient) VerifyOTP(to, code string) (bool, error) {
	if !t.IsConfigured() || t.verifyService == "" {
		return false, fmt.Errorf("twilio verify not configured")
	}

	endpoint := fmt.Sprintf("https://verify.twilio.com/v2/Services/%s/VerificationCheck", t.verifyService)

	data := url.Values{}
	data.Set("To", to)
	data.Set("Code", code)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return false, err
	}

	req.SetBasicAuth(t.accountSID, t.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		Status string `json:"status"`
		Valid  bool   `json:"valid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result.Status == "approved" || result.Valid, nil
}

// sendRequest sends a POST request to Twilio API (no context — legacy)
func (t *TwilioClient) sendRequest(endpoint string, data url.Values) error {
	return t.sendRequestWithContext(context.Background(), endpoint, data)
}

// sendRequestWithContext sends a POST request to Twilio API with context timeout.
func (t *TwilioClient) sendRequestWithContext(ctx context.Context, endpoint string, data url.Values) error {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(t.accountSID, t.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("twilio error %d: %s", errResp.Code, errResp.Message)
	}

	return nil
}

// ===== Notification Templates =====

// SendAuctionEndingSoon notifies user about auction ending
func (t *TwilioClient) SendAuctionEndingSoon(to, auctionTitle string, minutesLeft int, currentBid float64) error {
	msg := fmt.Sprintf("⏰ المزاد '%s' ينتهي خلال %d دقيقة! السعر الحالي: $%.2f. سارع بالمزايدة!",
		auctionTitle, minutesLeft, currentBid)
	return t.SendSMS(to, msg)
}

// SendOutbidNotification notifies user they were outbid
func (t *TwilioClient) SendOutbidNotification(to, auctionTitle string, newBid float64) error {
	msg := fmt.Sprintf("🔔 تم تجاوز مزايدتك على '%s'! المزايدة الجديدة: $%.2f. زايد الآن!",
		auctionTitle, newBid)
	return t.SendSMS(to, msg)
}

// SendAuctionWon notifies winner
func (t *TwilioClient) SendAuctionWon(to, auctionTitle string, winningBid float64) error {
	msg := fmt.Sprintf("🎉 مبروك! فزت بالمزاد '%s' بمبلغ $%.2f. أكمل الدفع الآن!",
		auctionTitle, winningBid)
	return t.SendSMS(to, msg)
}

// SendOrderConfirmation sends order confirmation
func (t *TwilioClient) SendOrderConfirmation(to, orderID string, total float64) error {
	msg := fmt.Sprintf("✅ تم تأكيد طلبك #%s بمبلغ $%.2f. سنقوم بإعلامك عند الشحن.",
		orderID, total)
	return t.SendSMS(to, msg)
}

// SendDeliveryUpdate sends delivery status update
func (t *TwilioClient) SendDeliveryUpdate(to, orderID, status string) error {
	msg := fmt.Sprintf("📦 تحديث الطلب #%s: %s", orderID, status)
	return t.SendSMS(to, msg)
}

// SendWelcomeMessage sends welcome message to new users
func (t *TwilioClient) SendWelcomeMessage(to, name string) error {
	msg := fmt.Sprintf("مرحباً %s! 👋 شكراً لانضمامك إلى GeoCore. ابدأ التسوق الآن!", name)
	return t.SendSMS(to, msg)
}

// SendPasswordReset sends password reset code
func (t *TwilioClient) SendPasswordReset(to, code string) error {
	msg := fmt.Sprintf("🔐 رمز إعادة تعيين كلمة المرور: %s. صالح لمدة 10 دقائق.", code)
	return t.SendSMS(to, msg)
}

// SendEscrowReleased notifies seller about escrow release
func (t *TwilioClient) SendEscrowReleased(to string, amount float64, orderID string) error {
	msg := fmt.Sprintf("💰 تم تحويل $%.2f إلى محفظتك من الطلب #%s. شكراً لك!",
		amount, orderID)
	return t.SendSMS(to, msg)
}
