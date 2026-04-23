package push

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// FirebaseClient sends push notifications via Firebase Cloud Messaging HTTP v1 API.
// Uses a service account JWT for authentication — no third-party SDK required.
type FirebaseClient struct {
	projectID   string
	clientEmail string
	privateKey  *rsa.PrivateKey
	httpClient  *http.Client
	cachedToken string
	tokenExpiry time.Time
}

type serviceAccountJSON struct {
	ProjectID   string `json:"project_id"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
}

// NewFirebaseClientFromEnv reads FIREBASE_SERVICE_ACCOUNT_JSON and initialises the client.
// Returns nil if the env var is not set (push notifications disabled).
func NewFirebaseClientFromEnv() *FirebaseClient {
	raw := os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON")
	if raw == "" {
		slog.Warn("push: FIREBASE_SERVICE_ACCOUNT_JSON not set — push notifications disabled")
		return nil
	}

	var sa serviceAccountJSON
	if err := json.Unmarshal([]byte(raw), &sa); err != nil {
		slog.Error("push: failed to parse service account JSON", "error", err.Error())
		return nil
	}

	block, _ := pem.Decode([]byte(sa.PrivateKey))
	if block == nil {
		slog.Error("push: invalid private key PEM")
		return nil
	}

	var pk *rsa.PrivateKey
	var err error
	if key, e := x509.ParsePKCS8PrivateKey(block.Bytes); e == nil {
		var ok bool
		if pk, ok = key.(*rsa.PrivateKey); !ok {
			slog.Error("push: service account key is not RSA")
			return nil
		}
	} else {
		pk, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			slog.Error("push: failed to parse RSA private key", "error", err.Error())
			return nil
		}
	}

	slog.Info("push: Firebase client initialised", "project_id", sa.ProjectID)
	return &FirebaseClient{
		projectID:   sa.ProjectID,
		clientEmail: sa.ClientEmail,
		privateKey:  pk,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// Send delivers a push notification to a single FCM token.
// Returns FCMResult with the message ID or token-level error.
func (f *FirebaseClient) Send(ctx context.Context, token, title, body string, data map[string]string, priority string) (*FCMResult, error) {
	bearer, err := f.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("fcm: get access token: %w", err)
	}

	// Build FCM v1 message payload
	message := map[string]any{
		"token": token,
	}

	// Add notification payload unless silent push
	if title != "" || body != "" {
		message["notification"] = map[string]string{
			"title": title,
			"body":  body,
		}
	}

	// Add data payload
	if data != nil {
		message["data"] = data
	}

	// Android-specific: priority mapping
	android := map[string]any{}
	if priority == PriorityHigh {
		android["priority"] = "high"
		message["android"] = android
	}

	// APNS-specific: priority mapping
	if priority == PriorityHigh {
		apns := map[string]any{
			"payload": map[string]any{
				"aps": map[string]any{
					"content-available": 1,
					"sound":             "default",
				},
			},
		}
		message["apns"] = apns
	}

	payload := map[string]any{"message": message}
	bodyBytes, _ := json.Marshal(payload)

	endpoint := fmt.Sprintf(
		"https://fcm.googleapis.com/v1/projects/%s/messages:send",
		f.projectID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearer)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fcm: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Parse FCM error for token-level diagnostics
		errMsg := string(respBody)
		return &FCMResult{Error: errMsg}, fmt.Errorf("fcm: HTTP %d: %s", resp.StatusCode, errMsg)
	}

	// Parse success response for message ID
	var successResp struct {
		Name string `json:"name"` // projects/.../messages/0:1234567890
	}
	_ = json.Unmarshal(respBody, &successResp)

	return &FCMResult{MessageID: successResp.Name}, nil
}

// ════════════════════════════════════════════════════════════════════════════
// JWT / OAuth2 helpers (stdlib only — no SDK dependency)
// ════════════════════════════════════════════════════════════════════════════

const googleTokenURL = "https://oauth2.googleapis.com/token"
const fcmScope = "https://www.googleapis.com/auth/firebase.messaging"

func (f *FirebaseClient) getAccessToken() (string, error) {
	if f.cachedToken != "" && time.Now().Before(f.tokenExpiry.Add(-30*time.Second)) {
		return f.cachedToken, nil
	}

	now := time.Now().Unix()
	header := b64url(`{"alg":"RS256","typ":"JWT"}`)
	claims := b64url(fmt.Sprintf(
		`{"iss":"%s","scope":"%s","aud":"%s","iat":%d,"exp":%d}`,
		f.clientEmail, fcmScope, googleTokenURL, now, now+3600,
	))
	unsigned := header + "." + claims

	sig, err := rsa.SignPKCS1v15(rand.Reader, f.privateKey, 0, sha256Hash([]byte(unsigned)))
	if err != nil {
		return "", fmt.Errorf("jwt sign: %w", err)
	}
	assertionJWT := unsigned + "." + base64.RawURLEncoding.EncodeToString(sig)

	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {assertionJWT},
	}
	resp, err := f.httpClient.PostForm(googleTokenURL, form)
	if err != nil {
		return "", fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()

	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", err
	}

	f.cachedToken = tok.AccessToken
	f.tokenExpiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	return f.cachedToken, nil
}

func b64url(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

func sha256Hash(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

// Ensure FirebaseClient satisfies FirebaseSender
var _ FirebaseSender = (*FirebaseClient)(nil)

// Ensure unused import guard is satisfied
var _ = strings.Contains
