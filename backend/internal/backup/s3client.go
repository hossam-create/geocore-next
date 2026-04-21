// Package backup provides a pure-stdlib S3-compatible client using AWS SigV4.
// Works with Amazon S3, Cloudflare R2, and MinIO out of the box.
package backup

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// S3Client is a minimal S3-compatible object storage client.
type S3Client struct {
	Endpoint  string // e.g. "https://s3.amazonaws.com" or R2 endpoint
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	hc        *http.Client
}

func NewS3Client(endpoint, region, bucket, accessKey, secretKey string) *S3Client {
	return &S3Client{
		Endpoint:  strings.TrimRight(endpoint, "/"),
		Region:    region,
		Bucket:    bucket,
		AccessKey: accessKey,
		SecretKey: secretKey,
		hc:        &http.Client{Timeout: 10 * time.Minute},
	}
}

// PutObject uploads data to key.
func (c *S3Client) PutObject(key string, body []byte, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	bodyHash := hexSHA256(body)
	reqURL := c.objectURL(key)
	req, err := http.NewRequest(http.MethodPut, reqURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-amz-content-sha256", bodyHash)
	c.sign(req, bodyHash, time.Now().UTC())
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3 PutObject %s: %s %s", key, resp.Status, b)
	}
	return nil
}

// GetObject downloads the object at key and returns its bytes.
func (c *S3Client) GetObject(key string) ([]byte, error) {
	reqURL := c.objectURL(key)
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-amz-content-sha256", hexSHA256(nil))
	c.sign(req, hexSHA256(nil), time.Now().UTC())
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("s3 GetObject %s: %s %s", key, resp.Status, b)
	}
	return io.ReadAll(resp.Body)
}

// DeleteObject removes the object at key.
func (c *S3Client) DeleteObject(key string) error {
	reqURL := c.objectURL(key)
	req, err := http.NewRequest(http.MethodDelete, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-amz-content-sha256", hexSHA256(nil))
	c.sign(req, hexSHA256(nil), time.Now().UTC())
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != 204 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3 DeleteObject %s: %s %s", key, resp.Status, b)
	}
	return nil
}

// ListObjectsResult holds a page of S3 object keys.
type ListObjectsResult struct {
	Keys []string
}

// ListObjects returns all keys under the given prefix.
func (c *S3Client) ListObjects(prefix string) (*ListObjectsResult, error) {
	reqURL := fmt.Sprintf("%s/%s?list-type=2&prefix=%s",
		c.Endpoint, c.Bucket, url.QueryEscape(prefix))
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-amz-content-sha256", hexSHA256(nil))
	c.sign(req, hexSHA256(nil), time.Now().UTC())
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("s3 ListObjects: %s %s", resp.Status, b)
	}
	var parsed struct {
		Contents []struct {
			Key string `xml:"Key"`
		} `xml:"Contents"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	out := &ListObjectsResult{}
	for _, c := range parsed.Contents {
		out.Keys = append(out.Keys, c.Key)
	}
	return out, nil
}

// ─── AWS SigV4 signing ───────────────────────────────────────────────────────

func (c *S3Client) sign(req *http.Request, payloadHash string, now time.Time) {
	dateISO := now.Format("20060102T150405Z")
	dateShort := now.Format("20060102")
	service := "s3"

	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("x-amz-date", dateISO)

	// Canonical headers (sorted).
	headerNames := []string{}
	for k := range req.Header {
		headerNames = append(headerNames, strings.ToLower(k))
	}
	sort.Strings(headerNames)

	var canonHeaders, signedHeaders strings.Builder
	for i, k := range headerNames {
		canonHeaders.WriteString(k + ":" + strings.TrimSpace(req.Header.Get(k)) + "\n")
		if i > 0 {
			signedHeaders.WriteString(";")
		}
		signedHeaders.WriteString(k)
	}
	signedHeadersStr := signedHeaders.String()

	// Canonical URI.
	canonURI := req.URL.EscapedPath()
	if canonURI == "" {
		canonURI = "/"
	}

	// Canonical query string.
	queryParts := []string{}
	for k, vs := range req.URL.Query() {
		for _, v := range vs {
			queryParts = append(queryParts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	sort.Strings(queryParts)
	canonQuery := strings.Join(queryParts, "&")

	canonRequest := strings.Join([]string{
		req.Method,
		canonURI,
		canonQuery,
		canonHeaders.String(),
		signedHeadersStr,
		payloadHash,
	}, "\n")

	credScope := dateShort + "/" + c.Region + "/" + service + "/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + dateISO + "\n" + credScope + "\n" + hexSHA256str(canonRequest)

	signingKey := hmacSHA256(
		hmacSHA256(
			hmacSHA256(
				hmacSHA256([]byte("AWS4"+c.SecretKey), []byte(dateShort)),
				[]byte(c.Region)),
			[]byte(service)),
		[]byte("aws4_request"))
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s,SignedHeaders=%s,Signature=%s",
		c.AccessKey, credScope, signedHeadersStr, signature,
	))
}

func (c *S3Client) objectURL(key string) string {
	return fmt.Sprintf("%s/%s/%s", c.Endpoint, c.Bucket, key)
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func hexSHA256(b []byte) string {
	h := sha256.New()
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

func hexSHA256str(s string) string { return hexSHA256([]byte(s)) }
