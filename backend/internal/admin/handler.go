package admin

  import (
        "bytes"
        "encoding/csv"
        "encoding/json"
        "fmt"
        "log/slog"
        "net/http"
        "strings"
        "time"

        "github.com/geocore-next/backend/internal/auctions"
        "github.com/geocore-next/backend/internal/listings"
        "github.com/geocore-next/backend/internal/payments"
        "github.com/geocore-next/backend/internal/users"
        "github.com/geocore-next/backend/pkg/response"
        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "gorm.io/gorm"
  )

  type Handler struct {
        db *gorm.DB
  }

  func NewHandler(db *gorm.DB) *Handler {
        return &Handler{db: db}
  }

  // ════════════════════════════════════════════════════════════════════════════
  // GET /admin/stats
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) GetStats(c *gin.Context) {
        today := time.Now().Truncate(24 * time.Hour)
        weekAgo := today.AddDate(0, 0, -7)
        var stats DashboardStats

        h.db.Model(&users.User{}).Count(&stats.TotalUsers)
        h.db.Model(&users.User{}).Where("created_at >= ?", today).Count(&stats.NewUsersThisWeek)
        h.db.Model(&users.User{}).Where("created_at >= ?", weekAgo).Count(&stats.NewUsersThisWeek)

        h.db.Model(&listings.Listing{}).Count(&stats.TotalListings)
        h.db.Model(&listings.Listing{}).Where("status = ?", "active").Count(&stats.ActiveListings)
        h.db.Model(&listings.Listing{}).Where("status = ?", "pending").Count(&stats.PendingModeration)
        h.db.Model(&listings.Listing{}).Where("created_at >= ?", today).Count(&stats.NewListingsToday)

        h.db.Model(&auctions.Auction{}).Count(&stats.TotalAuctions)
        h.db.Model(&auctions.Auction{}).
                Where("status = ? AND ends_at > NOW()", "active").Count(&stats.LiveAuctions)

        h.db.Model(&payments.Payment{}).
                Where("status = ?", "succeeded").
                Select("COALESCE(SUM(amount), 0)").Scan(&stats.TotalRevenue)
        h.db.Model(&payments.Payment{}).
                Where("status = ? AND created_at >= ?", "succeeded", today).
                Select("COALESCE(SUM(amount), 0)").Scan(&stats.RevenueToday)

        response.OK(c, stats)
  }

  // ════════════════════════════════════════════════════════════════════════════
  // GET /admin/users
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) ListUsers(c *gin.Context) {
        page, perPage := paginationParams(c)

        q := h.db.Model(&users.User{})
        if search := c.Query("q"); search != "" {
                q = q.Where("name ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
        }
        if role := c.Query("role"); role != "" {
                q = q.Where("role = ?", role)
        }
        if banned := c.Query("banned"); banned == "true" {
                q = q.Where("is_banned = true")
        }
        if verified := c.Query("verified"); verified == "true" {
                q = q.Where("email_verified = true")
        }

        var total int64
        q.Count(&total)

        var userList []users.User
        q.Offset((page - 1) * perPage).Limit(perPage).
                Order("created_at DESC").
                Find(&userList)

        response.OKMeta(c, userList, response.Meta{
                Total: total, Page: page, PerPage: perPage,
                Pages: (total + int64(perPage) - 1) / int64(perPage),
        })
  }

  // ════════════════════════════════════════════════════════════════════════════
  // GET /admin/users/:id
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) GetUser(c *gin.Context) {
        var user users.User
        if err := h.db.First(&user, "id = ?", c.Param("id")).Error; err != nil {
                response.NotFound(c, "user")
                return
        }

        var listingCount, soldCount int64
        h.db.Model(&listings.Listing{}).Where("user_id = ?", user.ID).Count(&listingCount)
        h.db.Model(&listings.Listing{}).Where("user_id = ? AND status = ?", user.ID, "sold").Count(&soldCount)

        response.OK(c, gin.H{
                "user":          user,
                "listing_count": listingCount,
                "sold_count":    soldCount,
        })
  }

  // ════════════════════════════════════════════════════════════════════════════
  // PUT /admin/users/:id — update role / verified status
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) UpdateUser(c *gin.Context) {
        var req struct {
                Role          string `json:"role"`
                EmailVerified *bool  `json:"email_verified"`
                IsActive      *bool  `json:"is_active"`
        }
        if err := c.ShouldBindJSON(&req); err != nil {
                response.BadRequest(c, err.Error())
                return
        }

        updates := map[string]any{}
        if req.Role != "" {
                updates["role"] = req.Role
        }
        if req.EmailVerified != nil {
                updates["email_verified"] = *req.EmailVerified
        }
        if req.IsActive != nil {
                updates["is_active"] = *req.IsActive
        }
        if len(updates) == 0 {
                response.BadRequest(c, "no fields to update")
                return
        }

        result := h.db.Model(&users.User{}).Where("id = ?", c.Param("id")).Updates(updates)
        if result.RowsAffected == 0 {
                response.NotFound(c, "user")
                return
        }

        h.logAction(c, "update_user", "user", c.Param("id"), updates)
        response.OK(c, gin.H{"message": "User updated."})
  }

  // ════════════════════════════════════════════════════════════════════════════
  // DELETE /admin/users/:id — soft delete
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) DeleteUser(c *gin.Context) {
        result := h.db.Where("id = ?", c.Param("id")).Delete(&users.User{})
        if result.RowsAffected == 0 {
                response.NotFound(c, "user")
                return
        }
        h.logAction(c, "delete_user", "user", c.Param("id"), nil)
        response.OK(c, gin.H{"message": "User deleted."})
  }

  // ════════════════════════════════════════════════════════════════════════════
  // POST /admin/users/:id/ban
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) BanUser(c *gin.Context) {
        var req struct {
                Reason string `json:"reason" binding:"required"`
        }
        if err := c.ShouldBindJSON(&req); err != nil {
                response.BadRequest(c, err.Error())
                return
        }

        reason := strings.TrimSpace(req.Reason)
        result := h.db.Model(&users.User{}).Where("id = ?", c.Param("id")).
                Updates(map[string]any{"is_banned": true, "ban_reason": reason})
        if result.RowsAffected == 0 {
                response.NotFound(c, "user")
                return
        }
        h.logAction(c, "ban_user", "user", c.Param("id"), map[string]string{"reason": reason})
        response.OK(c, gin.H{"message": "User banned."})
  }

  // ════════════════════════════════════════════════════════════════════════════
  // POST /admin/users/:id/unban
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) UnbanUser(c *gin.Context) {
        result := h.db.Model(&users.User{}).Where("id = ?", c.Param("id")).
                Updates(map[string]any{"is_banned": false, "ban_reason": ""})
        if result.RowsAffected == 0 {
                response.NotFound(c, "user")
                return
        }
        h.logAction(c, "unban_user", "user", c.Param("id"), nil)
        response.OK(c, gin.H{"message": "User unbanned."})
  }

  // ════════════════════════════════════════════════════════════════════════════
  // GET /admin/listings
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) ListListings(c *gin.Context) {
        page, perPage := paginationParams(c)
        q := h.db.Model(&listings.Listing{}).Preload("Category").Preload("Images")

        if status := c.Query("status"); status != "" {
                q = q.Where("status = ?", status)
        }
        if search := c.Query("q"); search != "" {
                q = q.Where("title ILIKE ?", "%"+search+"%")
        }

        var total int64
        q.Count(&total)
        var list []listings.Listing
        q.Offset((page-1)*perPage).Limit(perPage).Order("created_at DESC").Find(&list)

        response.OKMeta(c, list, response.Meta{
                Total: total, Page: page, PerPage: perPage,
                Pages: (total + int64(perPage) - 1) / int64(perPage),
        })
  }

  // ════════════════════════════════════════════════════════════════════════════
  // PUT /admin/listings/:id/approve
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) ApproveListing(c *gin.Context) {
        result := h.db.Model(&listings.Listing{}).
                Where("id = ? AND status = ?", c.Param("id"), "pending").
                Update("status", "active")
        if result.RowsAffected == 0 {
                response.BadRequest(c, "listing not found or not pending")
                return
        }
        h.logAction(c, "approve_listing", "listing", c.Param("id"), nil)
        response.OK(c, gin.H{"message": "Listing approved."})
  }

  // ════════════════════════════════════════════════════════════════════════════
  // PUT /admin/listings/:id/reject
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) RejectListing(c *gin.Context) {
        var req struct {
                Reason string `json:"reason" binding:"required"`
        }
        if err := c.ShouldBindJSON(&req); err != nil {
                response.BadRequest(c, err.Error())
                return
        }
        result := h.db.Model(&listings.Listing{}).Where("id = ?", c.Param("id")).
                Updates(map[string]any{"status": "rejected"})
        if result.RowsAffected == 0 {
                response.NotFound(c, "listing")
                return
        }
        h.logAction(c, "reject_listing", "listing", c.Param("id"), map[string]string{"reason": req.Reason})
        response.OK(c, gin.H{"message": "Listing rejected."})
  }

  // ════════════════════════════════════════════════════════════════════════════
  // DELETE /admin/listings/:id
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) DeleteListing(c *gin.Context) {
        result := h.db.Unscoped().Where("id = ?", c.Param("id")).Delete(&listings.Listing{})
        if result.RowsAffected == 0 {
                response.NotFound(c, "listing")
                return
        }
        h.logAction(c, "delete_listing", "listing", c.Param("id"), nil)
        response.OK(c, gin.H{"message": "Listing permanently deleted."})
  }

  // ════════════════════════════════════════════════════════════════════════════
  // GET /admin/revenue — daily / weekly / monthly breakdown
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) GetRevenue(c *gin.Context) {
        type dailyRevenue struct {
                Date    string  `json:"date"`
                Revenue float64 `json:"revenue"`
                Count   int64   `json:"count"`
        }
        var daily []dailyRevenue
        h.db.Model(&payments.Payment{}).
                Select("TO_CHAR(created_at, 'YYYY-MM-DD') as date, COALESCE(SUM(amount), 0) as revenue, COUNT(*) as count").
                Where("status = ? AND created_at >= NOW() - INTERVAL '30 days'", "succeeded").
                Group("date").
                Order("date DESC").
                Scan(&daily)

        var totalRevenue float64
        h.db.Model(&payments.Payment{}).
                Where("status = ?", "succeeded").
                Select("COALESCE(SUM(amount), 0)").Scan(&totalRevenue)

        response.OK(c, gin.H{
                "total":        totalRevenue,
                "daily_30days": daily,
        })
  }

  // ════════════════════════════════════════════════════════════════════════════
  // GET /admin/transactions — all payments + CSV export
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) GetTransactions(c *gin.Context) {
        page, perPage := paginationParams(c)
        q := h.db.Model(&payments.Payment{}).Preload("Escrow")

        if status := c.Query("status"); status != "" {
                q = q.Where("status = ?", status)
        }

        // CSV export
        if c.Query("export") == "csv" {
                var all []payments.Payment
                q.Order("created_at DESC").Find(&all)

                var buf bytes.Buffer
                w := csv.NewWriter(&buf)
                w.Write([]string{"ID", "User ID", "Amount", "Currency", "Status", "Created At"}) //nolint:errcheck
                for _, p := range all {
                        w.Write([]string{ //nolint:errcheck
                                p.ID.String(),
                                p.UserID.String(),
                                fmt.Sprintf("%.2f", p.Amount),
                                p.Currency,
                                string(p.Status),
                                p.CreatedAt.Format(time.RFC3339),
                        })
                }
                w.Flush()
                c.Header("Content-Disposition", "attachment; filename=transactions.csv")
                c.Data(http.StatusOK, "text/csv", buf.Bytes())
                return
        }

        var total int64
        q.Count(&total)
        var list []payments.Payment
        q.Offset((page-1)*perPage).Limit(perPage).Order("created_at DESC").Find(&list)

        response.OKMeta(c, list, response.Meta{
                Total: total, Page: page, PerPage: perPage,
                Pages: (total + int64(perPage) - 1) / int64(perPage),
        })
  }

  // ════════════════════════════════════════════════════════════════════════════
  // GET /admin/logs — audit trail
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) GetAuditLogs(c *gin.Context) {
        page, perPage := paginationParams(c)
        var total int64
        h.db.Model(&AdminLog{}).Count(&total)

        var logs []AdminLog
        h.db.Offset((page-1)*perPage).Limit(perPage).Order("created_at DESC").Find(&logs)

        response.OKMeta(c, logs, response.Meta{
                Total: total, Page: page, PerPage: perPage,
                Pages: (total + int64(perPage) - 1) / int64(perPage),
        })
  }

  // ════════════════════════════════════════════════════════════════════════════
  // Helpers
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) logAction(c *gin.Context, action, targetType, targetID string, details interface{}) {
        adminIDStr := c.GetString("user_id")
        adminID, err := uuid.Parse(adminIDStr)
        if err != nil {
                return
        }

        detailsJSON := "{}"
        if details != nil {
                if b, e := json.Marshal(details); e == nil {
                        detailsJSON = string(b)
                }
        }

        log := AdminLog{
                AdminID:    adminID,
                Action:     action,
                TargetType: targetType,
                TargetID:   targetID,
                Details:    detailsJSON,
                IPAddress:  c.ClientIP(),
        }
        if err := h.db.Create(&log).Error; err != nil {
                slog.Warn("failed to write admin log", "action", action, "error", err.Error())
        }
  }

  func paginationParams(c *gin.Context) (page, perPage int) {
        page, perPage = 1, 20
        fmt.Sscan(c.DefaultQuery("page", "1"), &page)
        fmt.Sscan(c.DefaultQuery("per_page", "20"), &perPage)
        if page < 1 { page = 1 }
        if perPage < 1 || perPage > 100 { perPage = 20 }
        return
  }

  