package images

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// PresignPutURL generates an AWS Signature V4 presigned PUT URL for direct
// client-to-storage uploads. expireSecs is how long the URL is valid.
func (c *R2Client) PresignPutURL(key, contentType string, expireSecs int) (string, error) {
	now := time.Now().UTC()
	datetime := now.Format("20060102T150405Z")
	date := now.Format("20060102")

	objectURL := fmt.Sprintf("%s/%s/%s", c.endpointURL, c.Bucket, key)
	u, err := url.Parse(objectURL)
	if err != nil {
		return "", err
	}

	credential := fmt.Sprintf("%s/%s/%s/%s/aws4_request", c.AccessKey, date, r2Region, r2Service)

	q := u.Query()
	q.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	q.Set("X-Amz-Credential", credential)
	q.Set("X-Amz-Date", datetime)
	q.Set("X-Amz-Expires", fmt.Sprintf("%d", expireSecs))
	q.Set("X-Amz-SignedHeaders", "content-type;host")
	u.RawQuery = q.Encode()

	canonicalURI := u.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	canonicalQueryString := u.RawQuery

	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\n", contentType, u.Host)
	signedHeaders := "content-type;host"

	canonicalRequest := strings.Join([]string{
		"PUT",
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		"UNSIGNED-PAYLOAD",
	}, "\n")

	credentialScope := date + "/" + r2Region + "/" + r2Service + "/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		datetime,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	kSigning := signingKey(c.SecretKey, date, r2Region, r2Service)
	signature := hmacSHA256(kSigning, stringToSign)

	q.Set("X-Amz-Signature", fmt.Sprintf("%x", signature))
	u.RawQuery = q.Encode()

	return u.String(), nil
}
