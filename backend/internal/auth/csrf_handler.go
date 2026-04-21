package auth

import (
	"errors"

	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// CSRFToken issues (or reuses) a CSRF cookie token for browser clients.
// GET /api/v1/auth/csrf-token
func (h *Handler) CSRFToken(c *gin.Context) {
	token := middleware.EnsureCSRFCookie(c)
	if token == "" {
		response.InternalError(c, errors.New("failed to generate csrf token"))
		return
	}
	response.OK(c, gin.H{"csrf_token": token})
}
