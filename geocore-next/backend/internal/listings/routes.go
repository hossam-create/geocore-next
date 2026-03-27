package listings

import (
        "time"

        "github.com/geocore-next/backend/pkg/middleware"
        "github.com/gin-gonic/gin"
        "github.com/redis/go-redis/v9"
        "gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client, rl *middleware.RateLimiter) {
        h := NewHandler(db, rdb)

        r.GET("/categories", h.GetCategories)

        // ── Search endpoints (public) ──────────────────────────────────────────────
        r.GET("/listings/search",      h.Search)
        r.GET("/listings/suggestions", h.Suggestions)

        listings := r.Group("/listings")
        {
                // Public — anyone can browse listings
                listings.GET("", h.List)

                // Auth required — must be registered BEFORE /:id so Gin matches "me" as a literal
                authed := listings.Group("")
                authed.Use(middleware.Auth())
                {
                        authed.GET("/me",              h.GetMyListings)
                        authed.GET("/recent-searches", h.RecentSearches)
                        authed.PUT("/:id",             h.Update)
                        authed.DELETE("/:id",          h.Delete)
                        authed.POST("/:id/favorite",   h.ToggleFavorite)
                }

                // Public wildcard — registered after literal /me so it does not shadow it
                listings.GET("/:id", h.Get)

                // Auth + email verified — posting requires verified email
                // Per-user rate limit: 10 new listings per hour to prevent spam.
                verified := listings.Group("")
                verified.Use(
                        middleware.Auth(),
                        middleware.EmailVerified(db),
                        rl.LimitByUser(10, time.Hour, "listings:create"),
                )
                {
                        verified.POST("", h.Create)
                }
        }
}
