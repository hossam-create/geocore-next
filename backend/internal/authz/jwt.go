package authz

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ControlPlaneClaims are the JWT claims embedded in control-plane tokens.
type ControlPlaneClaims struct {
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// IssueToken creates a signed control-plane JWT for a tenant.
func IssueToken(tenantID, role string, ttl time.Duration) (string, error) {
	secret := cpSecret()
	claims := ControlPlaneClaims{
		TenantID: tenantID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   tenantID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ValidateToken parses and validates a control-plane JWT.
func ValidateToken(tokenStr string) (*ControlPlaneClaims, error) {
	claims := &ControlPlaneClaims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(cpSecret()), nil
	})
	if err != nil || !tok.Valid {
		return nil, fmt.Errorf("invalid or expired control plane token")
	}
	return claims, nil
}

func cpSecret() string {
	if s := os.Getenv("CONTROL_PLANE_SECRET"); s != "" {
		return s
	}
	return "dev-control-plane-secret-change-in-prod"
}
