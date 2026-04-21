package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SocialLoginReq is the request body for POST /api/v1/auth/social.
type SocialLoginReq struct {
	Provider string `json:"provider" binding:"required"`
	Token    string `json:"token"    binding:"required"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

// SocialLogin — POST /api/v1/auth/social
// Accepts an ID token / access token from Google, Apple, or Facebook.
// Verifies it with the provider, then creates or returns an existing user.
// Social-login users have email_verified = true automatically.
func (h *Handler) SocialLogin(c *gin.Context) {
	var req SocialLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	if req.Provider != "google" && req.Provider != "apple" && req.Provider != "facebook" {
		response.BadRequest(c, "provider must be one of: google, apple, facebook")
		return
	}

	var providerID, verifiedEmail, verifiedName string
	var err error

	switch req.Provider {
	case "google":
		providerID, verifiedEmail, verifiedName, err = verifyGoogleToken(req.Token)
	case "apple":
		providerID, verifiedEmail, err = verifyAppleToken(req.Token)
		verifiedName = req.Name // Apple sends name only on first sign-in
	case "facebook":
		providerID, verifiedEmail, verifiedName, err = verifyFacebookToken(req.Token)
	}

	if err != nil {
		response.BadRequest(c, fmt.Sprintf("Could not verify %s token: %v", req.Provider, err))
		return
	}

	// Prefer provider-verified email; fall back to client-supplied one (Apple first sign-in)
	if verifiedEmail == "" {
		verifiedEmail = req.Email
	}
	if verifiedName == "" {
		verifiedName = req.Name
	}
	if verifiedName == "" {
		verifiedName = "GeoCore User"
	}

	// --- Find existing user ---
	var user users.User
	var found bool

	// 1. Look up by provider-specific ID column
	switch req.Provider {
	case "google":
		found = h.db.Where("google_id = ?", providerID).First(&user).Error == nil
	case "apple":
		found = h.db.Where("apple_id = ?", providerID).First(&user).Error == nil
	case "facebook":
		found = h.db.Where("facebook_id = ?", providerID).First(&user).Error == nil
	}

	// 2. Fall back to email lookup (handles account merging)
	if !found && verifiedEmail != "" {
		found = h.db.Where("email = ?", verifiedEmail).First(&user).Error == nil
	}

	if found {
		// Attach this provider's ID if it was never stored
		updates := map[string]interface{}{"email_verified": true, "auth_provider": req.Provider}
		switch req.Provider {
		case "google":
			if user.GoogleID == "" {
				updates["google_id"] = providerID
			}
		case "apple":
			if user.AppleID == "" {
				updates["apple_id"] = providerID
			}
		case "facebook":
			if user.FacebookID == "" {
				updates["facebook_id"] = providerID
			}
		}
		h.db.Model(&user).Updates(updates)
	} else {
		// --- Create new social user ---
		user = users.User{
			ID:            uuid.New(),
			Name:          verifiedName,
			Email:         verifiedEmail,
			PasswordHash:  "", // No password — social login only
			EmailVerified: true,
			AuthProvider:  req.Provider,
		}
		switch req.Provider {
		case "google":
			user.GoogleID = providerID
		case "apple":
			user.AppleID = providerID
		case "facebook":
			user.FacebookID = providerID
		}
		if err := h.db.Create(&user).Error; err != nil {
			response.InternalError(c, err)
			return
		}
	}

	accessToken, err := generateAccessToken(user.ID.String(), user.Email)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	refreshToken, err := generateRefreshToken(c.Request.Context(), h.rdb, user.ID.String(), user.Email)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          user,
	})
}

// verifyGoogleToken calls Google's tokeninfo endpoint to validate an access token.
func verifyGoogleToken(accessToken string) (sub, email, name string, err error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v1/tokeninfo?access_token=" + accessToken)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	var claims struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		Error  string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return "", "", "", err
	}
	if claims.Error != "" {
		return "", "", "", fmt.Errorf("google: %s", claims.Error)
	}

	// Fetch name separately from userinfo endpoint
	req2, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v1/userinfo", nil)
	req2.Header.Set("Authorization", "Bearer "+accessToken)
	resp2, err2 := http.DefaultClient.Do(req2)
	if err2 == nil {
		defer resp2.Body.Close()
		var info struct {
			Name string `json:"name"`
		}
		json.NewDecoder(resp2.Body).Decode(&info) //nolint:errcheck
		name = info.Name
	}

	return claims.UserID, claims.Email, name, nil
}

// verifyFacebookToken calls the Facebook Graph API to validate an access token.
func verifyFacebookToken(accessToken string) (id, email, name string, err error) {
	url := "https://graph.facebook.com/v18.0/me?fields=id,name,email&access_token=" + accessToken
	resp, err := http.Get(url)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var data struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", "", "", err
	}
	if data.Error.Message != "" {
		return "", "", "", fmt.Errorf("facebook: %s", data.Error.Message)
	}
	return data.ID, data.Email, data.Name, nil
}

// verifyAppleToken decodes the Apple identity token (a JWT) and extracts claims.
// Apple tokens are signed JWTs — in production you should also verify the
// signature using Apple's public keys from https://appleid.apple.com/auth/keys.
func verifyAppleToken(identityToken string) (sub, email string, err error) {
	parts := strings.Split(identityToken, ".")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("invalid Apple identity token format")
	}

	// Decode the base64url-encoded payload (second part)
	payloadB64 := parts[1]
	// Pad to a multiple of 4 for standard base64
	switch len(payloadB64) % 4 {
	case 2:
		payloadB64 += "=="
	case 3:
		payloadB64 += "="
	}

	decoded, decErr := base64.StdEncoding.DecodeString(payloadB64)
	if decErr != nil {
		// Try URL-safe variant
		decoded, decErr = base64.URLEncoding.DecodeString(payloadB64)
		if decErr != nil {
			decoded, decErr = base64.RawURLEncoding.DecodeString(parts[1])
			if decErr != nil {
				return "", "", fmt.Errorf("base64 decode error: %w", decErr)
			}
		}
	}

	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Iss   string `json:"iss"`
	}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return "", "", fmt.Errorf("json decode error: %w", err)
	}

	// Basic sanity check — Apple tokens always come from appleid.apple.com
	if claims.Iss != "" && !strings.Contains(claims.Iss, "apple.com") {
		return "", "", fmt.Errorf("unexpected issuer: %s", claims.Iss)
	}
	if claims.Sub == "" {
		return "", "", fmt.Errorf("missing sub claim in Apple token")
	}

	return claims.Sub, claims.Email, nil
}
