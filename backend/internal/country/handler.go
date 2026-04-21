package country

import (
	"net/http"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ── Handler ────────────────────────────────────────────────────────────────────

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// ── Public Endpoints ───────────────────────────────────────────────────────────

// GET /api/v1/country/:code — resolve full config for a country (public)
func (h *Handler) GetConfig(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		response.BadRequest(c, "country code is required")
		return
	}

	userID := ""
	if uid, exists := c.Get("user_id"); exists {
		if s, ok := uid.(string); ok {
			userID = s
		}
	}

	resolved, err := h.repo.ResolveConfig(c.Request.Context(), code, userID)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.OK(c, resolved)
}

// GET /api/v1/country — list all active country configs (public)
func (h *Handler) ListConfigs(c *gin.Context) {
	configs, err := h.repo.GetAllConfigs(c.Request.Context())
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, configs)
}

// GET /api/v1/country/:code/overrides — list overrides for a country
func (h *Handler) ListOverrides(c *gin.Context) {
	code := c.Param("code")
	targetType := c.Query("target_type")
	targetID := c.Query("target_id")

	overrides, err := h.repo.GetOverrides(c.Request.Context(), code, targetType, targetID)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, overrides)
}

// ── Admin Endpoints ────────────────────────────────────────────────────────────

// POST /admin/country — create or update a country config
func (h *Handler) UpsertConfig(c *gin.Context) {
	var cfg CountryConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if cfg.Code == "" {
		response.BadRequest(c, "country code is required")
		return
	}

	if err := h.repo.UpsertConfig(c.Request.Context(), &cfg); err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, cfg)
}

// DELETE /admin/country/:code — deactivate a country config
func (h *Handler) DeleteConfig(c *gin.Context) {
	code := c.Param("code")
	if err := h.repo.DeleteConfig(c.Request.Context(), code); err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

// POST /admin/country/:code/overrides — create an override
func (h *Handler) CreateOverride(c *gin.Context) {
	var o CountryOverride
	if err := c.ShouldBindJSON(&o); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	o.CountryCode = c.Param("code")
	if o.CountryCode == "" {
		response.BadRequest(c, "country code is required")
		return
	}

	// Set created_by from authenticated admin
	if adminID, exists := c.Get("user_id"); exists {
		if s, ok := adminID.(string); ok {
			if uid, err := uuid.Parse(s); err == nil {
				o.CreatedBy = &uid
			}
		}
	}

	if err := h.repo.CreateOverride(c.Request.Context(), &o); err != nil {
		response.InternalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, o)
}

// DELETE /admin/country/overrides/:id — delete an override
func (h *Handler) DeleteOverride(c *gin.Context) {
	id := c.Param("id")
	if err := h.repo.DeleteOverride(c.Request.Context(), id); err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

// ── Middleware ──────────────────────────────────────────────────────────────────

// CountryMiddleware resolves the user's country from X-Country-Code header
// or IP geolocation, and injects the resolved config into the Gin context.
func CountryMiddleware(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.GetHeader("X-Country-Code")
		if code == "" {
			// Default to EG (Egypt) as primary market
			code = "EG"
		}

		userID := ""
		if uid, exists := c.Get("user_id"); exists {
			if s, ok := uid.(string); ok {
				userID = s
			}
		}

		resolved, err := repo.ResolveConfig(c.Request.Context(), code, userID)
		if err != nil {
			// Don't block the request — just log and continue without country config
			c.Set("country_code", code)
			c.Next()
			return
		}

		c.Set("country_code", code)
		c.Set("country_config", resolved)
		c.Header("X-Country-Code", code)
		c.Header("X-Currency", resolved.Currency)
		c.Next()
	}
}
