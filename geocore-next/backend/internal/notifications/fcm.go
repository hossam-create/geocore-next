package notifications

  import (
  	"bytes"
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
  	"math/big"
  	"net/http"
  	"net/url"
  	"os"
  	"strings"
  	"time"
  )

  // FCMClient sends push notifications via Firebase Cloud Messaging HTTP v1 API.
  // Authentication uses a service account JWT — no third-party SDK required.
  type FCMClient struct {
  	projectID    string
  	clientEmail  string
  	privateKey   *rsa.PrivateKey
  	httpClient   *http.Client
  	cachedToken  string
  	tokenExpiry  time.Time
  }

  type serviceAccountJSON struct {
  	ProjectID   string `json:"project_id"`
  	ClientEmail string `json:"client_email"`
  	PrivateKey  string `json:"private_key"`
  }

  // NewFCMClientFromEnv reads FIREBASE_SERVICE_ACCOUNT_JSON and initialises the client.
  // Returns nil if the env var is not set (push notifications disabled).
  func NewFCMClientFromEnv() *FCMClient {
  	raw := os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON")
  	if raw == "" {
  		slog.Warn("FIREBASE_SERVICE_ACCOUNT_JSON not set — push notifications disabled")
  		return nil
  	}

  	var sa serviceAccountJSON
  	if err := json.Unmarshal([]byte(raw), &sa); err != nil {
  		slog.Error("FCM: failed to parse service account JSON", "error", err.Error())
  		return nil
  	}

  	// Parse RSA private key from PEM
  	block, _ := pem.Decode([]byte(sa.PrivateKey))
  	if block == nil {
  		slog.Error("FCM: invalid private key PEM")
  		return nil
  	}
  	var pk *rsa.PrivateKey
  	var err error
  	// Try PKCS8 first, fall back to PKCS1
  	if key, e := x509.ParsePKCS8PrivateKey(block.Bytes); e == nil {
  		var ok bool
  		if pk, ok = key.(*rsa.PrivateKey); !ok {
  			slog.Error("FCM: service account key is not an RSA key")
  			return nil
  		}
  	} else {
  		pk, err = x509.ParsePKCS1PrivateKey(block.Bytes)
  		if err != nil {
  			slog.Error("FCM: failed to parse RSA private key", "error", err.Error())
  			return nil
  		}
  	}

  	slog.Info("✅ FCM client initialised", "project_id", sa.ProjectID)
  	return &FCMClient{
  		projectID:   sa.ProjectID,
  		clientEmail: sa.ClientEmail,
  		privateKey:  pk,
  		httpClient:  &http.Client{Timeout: 10 * time.Second},
  	}
  }

  // Send sends a push notification to a single FCM registration token.
  func (f *FCMClient) Send(token, title, body string, data map[string]string) error {
  	bearer, err := f.getAccessToken()
  	if err != nil {
  		return fmt.Errorf("fcm: get access token: %w", err)
  	}

  	payload := map[string]interface{}{
  		"message": map[string]interface{}{
  			"token": token,
  			"notification": map[string]string{
  				"title": title,
  				"body":  body,
  			},
  			"data": data,
  		},
  	}
  	body2, _ := json.Marshal(payload)

  	endpoint := fmt.Sprintf(
  		"https://fcm.googleapis.com/v1/projects/%s/messages:send",
  		f.projectID,
  	)
  	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body2))
  	if err != nil {
  		return err
  	}
  	req.Header.Set("Content-Type", "application/json")
  	req.Header.Set("Authorization", "Bearer "+bearer)

  	resp, err := f.httpClient.Do(req)
  	if err != nil {
  		return fmt.Errorf("fcm: HTTP request failed: %w", err)
  	}
  	defer resp.Body.Close()

  	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
  		respBody, _ := io.ReadAll(resp.Body)
  		return fmt.Errorf("fcm: HTTP %d: %s", resp.StatusCode, respBody)
  	}
  	return nil
  }

  // SendMulticast sends a notification to multiple tokens (sequentially — FCM v1 has no batch endpoint).
  func (f *FCMClient) SendMulticast(tokens []string, title, body string, data map[string]string) {
  	for _, t := range tokens {
  		if err := f.Send(t, title, body, data); err != nil {
  			slog.Warn("FCM multicast: send failed", "token_prefix", safePrefix(t), "error", err.Error())
  		}
  	}
  }

  // ════════════════════════════════════════════════════════════════════════════
  // JWT / OAuth2 helpers (stdlib only)
  // ════════════════════════════════════════════════════════════════════════════

  const googleTokenURL = "https://oauth2.googleapis.com/token"
  const fcmScope       = "https://www.googleapis.com/auth/firebase.messaging"

  // getAccessToken returns a cached bearer token, refreshing it when it expires.
  func (f *FCMClient) getAccessToken() (string, error) {
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

  	sig, err := signRS256(f.privateKey, []byte(unsigned))
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

  	f.cachedToken  = tok.AccessToken
  	f.tokenExpiry  = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
  	return f.cachedToken, nil
  }

  func b64url(s string) string {
  	return base64.RawURLEncoding.EncodeToString([]byte(s))
  }

  func signRS256(key *rsa.PrivateKey, data []byte) ([]byte, error) {
  	h := sha256.New()
  	h.Write(data)
  	digest := h.Sum(nil)
  	return rsa.SignPKCS1v15(rand.Reader, key, 0, digest)
  }

  // bigZero is used internally for compatibility.
  var _ = big.NewInt(0)

  func safePrefix(token string) string {
  	if len(token) > 8 {
  		return token[:8] + "..."
  	}
  	return strings.Repeat("*", len(token))
  }
  