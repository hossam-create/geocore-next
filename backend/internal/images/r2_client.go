package images

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// ════════════════════════════════════════════════════════════════════════════
// R2Client — S3-compatible client for Cloudflare R2
// ════════════════════════════════════════════════════════════════════════════

// R2Client holds the configuration for Cloudflare R2 object storage.
// All communication uses AWS Signature Version 4 (standard library only).
type R2Client struct {
	AccountID   string
	AccessKey   string
	SecretKey   string
	Bucket      string
	PublicURL   string // e.g. https://images.geocore.com
	endpointURL string // https://{account}.r2.cloudflarestorage.com
	httpClient  *http.Client
}

// NewR2ClientFromEnv creates an R2Client from environment variables.
//
//	R2_ACCOUNT_ID        — Cloudflare account ID
//	R2_ACCESS_KEY_ID     — R2 access key
//	R2_SECRET_ACCESS_KEY — R2 secret key
//	R2_BUCKET_NAME       — bucket name (default: geocore-images)
//	R2_PUBLIC_URL        — public CDN base URL
func NewR2ClientFromEnv() *R2Client {
	acct := os.Getenv("R2_ACCOUNT_ID")
	if acct == "" {
		slog.Warn("R2_ACCOUNT_ID not set — image uploads disabled")
		return nil
	}
	bucket := os.Getenv("R2_BUCKET_NAME")
	if bucket == "" {
		bucket = "geocore-images"
	}
	return &R2Client{
		AccountID:   acct,
		AccessKey:   os.Getenv("R2_ACCESS_KEY_ID"),
		SecretKey:   os.Getenv("R2_SECRET_ACCESS_KEY"),
		Bucket:      bucket,
		PublicURL:   os.Getenv("R2_PUBLIC_URL"),
		endpointURL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", acct),
		httpClient:  &http.Client{Timeout: 60 * time.Second},
	}
}

// Put uploads data to R2 at the given key and returns the public URL.
func (c *R2Client) Put(key string, data []byte, contentType string) (string, error) {
	url := fmt.Sprintf("%s/%s/%s", c.endpointURL, c.Bucket, key)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("r2: build request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = int64(len(data))

	c.signV4(req, data)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("r2: PUT %s: %w", key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("r2: PUT %s: HTTP %d: %s", key, resp.StatusCode, body)
	}

	publicURL := fmt.Sprintf("%s/%s", strings.TrimRight(c.PublicURL, "/"), key)
	slog.Debug("r2: uploaded", "key", key, "bytes", len(data), "url", publicURL)
	return publicURL, nil
}

// Get fetches an object from R2 by key and returns its data.
func (c *R2Client) Get(key string) ([]byte, error) {
	url := fmt.Sprintf("%s/%s/%s", c.endpointURL, c.Bucket, key)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("r2: build request: %w", err)
	}

	c.signV4(req, nil)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("r2: GET %s: %w", key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("r2: GET %s: HTTP %d: %s", key, resp.StatusCode, body)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("r2: GET %s: read body: %w", key, err)
	}
	return data, nil
}

// Delete removes an object from R2 by key.
func (c *R2Client) Delete(key string) error {
	url := fmt.Sprintf("%s/%s/%s", c.endpointURL, c.Bucket, key)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("r2: build request: %w", err)
	}

	c.signV4(req, nil)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("r2: DELETE %s: %w", key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("r2: DELETE %s: HTTP %d: %s", key, resp.StatusCode, body)
	}
	return nil
}

// ════════════════════════════════════════════════════════════════════════════
// AWS Signature Version 4 signing  (stdlib-only)
// Reference: https://docs.aws.amazon.com/general/latest/gr/sigv4_signing.html
// ════════════════════════════════════════════════════════════════════════════

const (
	r2Region  = "auto" // Cloudflare R2 region constant
	r2Service = "s3"
)

// signV4 mutates req to add the Authorization header signed with AWS SigV4.
func (c *R2Client) signV4(req *http.Request, body []byte) {
	now := time.Now().UTC()
	datetime := now.Format("20060102T150405Z")
	date := now.Format("20060102")

	// ── Payload hash ─────────────────────────────────────────────────────────
	payloadHash := sha256Hex(body)

	// ── Required headers ─────────────────────────────────────────────────────
	req.Header.Set("x-amz-date", datetime)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	req.Header.Set("Host", req.URL.Host)

	// ── Canonical headers (sorted, lowercase) ────────────────────────────────
	signedHeaders, canonicalHeaders := buildCanonicalHeaders(req)

	// ── Canonical request ────────────────────────────────────────────────────
	canonicalURI := req.URL.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	canonicalQueryString := req.URL.Query().Encode()

	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	// ── String to sign ────────────────────────────────────────────────────────
	credentialScope := date + "/" + r2Region + "/" + r2Service + "/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		datetime,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	// ── Signing key: HMAC chain ────────────────────────────────────────────────
	kSigning := signingKey(c.SecretKey, date, r2Region, r2Service)

	// ── Signature ────────────────────────────────────────────────────────────
	signature := hex.EncodeToString(hmacSHA256(kSigning, stringToSign))

	// ── Authorization header ─────────────────────────────────────────────────
	authorization := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.AccessKey, credentialScope, signedHeaders, signature,
	)
	req.Header.Set("Authorization", authorization)
}

// buildCanonicalHeaders returns (signedHeaders, canonicalHeaders) for SigV4.
func buildCanonicalHeaders(req *http.Request) (signed, canonical string) {
	// Collect headers that should be signed
	type kv struct{ k, v string }
	var headers []kv
	for k, vs := range req.Header {
		lower := strings.ToLower(k)
		// Sign Host + all x-amz-* headers + Content-Type
		if lower == "host" || lower == "content-type" || strings.HasPrefix(lower, "x-amz-") {
			headers = append(headers, kv{lower, strings.Join(vs, ",")})
		}
	}
	sort.Slice(headers, func(i, j int) bool { return headers[i].k < headers[j].k })

	var sb strings.Builder
	keys := make([]string, 0, len(headers))
	for _, h := range headers {
		sb.WriteString(h.k)
		sb.WriteByte(':')
		sb.WriteString(strings.TrimSpace(h.v))
		sb.WriteByte('\n')
		keys = append(keys, h.k)
	}
	return strings.Join(keys, ";"), sb.String()
}

// signingKey derives the SigV4 signing key via nested HMAC chain.
func signingKey(secretKey, date, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), date)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
