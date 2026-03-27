package search

  import (
      "github.com/gin-gonic/gin"
      "gorm.io/gorm"
  )

  // RegisterRoutes mounts search endpoints under the given router group.
  func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
      h := NewHandler(db)

      // Semantic AI search
      rg.POST("/search", h.Search)

      // Autocomplete suggestions
      rg.GET("/search/suggest", h.Suggest)

      // Trending queries
      rg.GET("/search/trending", h.Trending)

      // On-demand embedding for a specific listing (admin/indexer use)
      rg.POST("/listings/:id/embed", h.EmbedListing)
  }
  