package admin

  import (
        "bytes"
        "encoding/csv"
        "encoding/json"
        "fmt"
        "log/slog"
        "net/http"
        "os"
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

  // requireAdmin is a defense-in-depth role check called at the top of every
  // admin handler. It performs a fresh DB lookup independently of the middleware
  // stack — so it remains effective even if Auth()+AdminWithDB() are accidentally
  // mis-ordered or removed in a future refactor.
  // Returns true if the request should continue; false (with 403 abort) if not.
  func (h *Handler) requireAdmin(c *gin.Context) bool {
        userID := c.GetString("user_id")
        if userID == "" {
                c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                        "error":   "unauthorized",
                        "message": "authentication required",
                })
                return false
        }

        var row struct{ Role string }
        if err := h.db.Table("users").Select("role").Where("id = ?", userID).Scan(&row).Error; err != nil {
                slog.Error("admin: DB role verification failed",
                        "user_id", userID,
                        "path",    c.FullPath(),
                        "error",   err.Error(),
                )
                c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
                        "error":   "server_error",
                        "message": "could not verify admin role",
                })
                return false
        }

        if row.Role != "admin" && row.Role != "super_admin" {
                slog.Warn("admin: unauthorized handler access",
                        "user_id",     userID,
                        "db_role",     row.Role,
                        "ctx_role",    c.GetString("user_role"),
                        "path",        c.FullPath(),
                        "ip",          c.ClientIP(),
                )
                c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                        "error":   "forbidden",
                        "message": "admin role required",
                })
                return false
        }
        return true
  }

  // ════════════════════════════════════════════════════════════════════════════
  // GET /admin/stats
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) GetStats(c *gin.Context) {
        if !h.requireAdmin(c) { return }
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
        if !h.requireAdmin(c) { return }
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
        if !h.requireAdmin(c) { return }
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
        if !h.requireAdmin(c) { return }
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
        if !h.requireAdmin(c) { return }
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
        if !h.requireAdmin(c) { return }
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
        if !h.requireAdmin(c) { return }
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
        if !h.requireAdmin(c) { return }
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
        if !h.requireAdmin(c) { return }
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
        if !h.requireAdmin(c) { return }
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
        if !h.requireAdmin(c) { return }
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
        if !h.requireAdmin(c) { return }
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

        // ── Monetization breakdown ────────────────────────────────────────────
        var totalCommissions float64
        h.db.Table("platform_commissions").
                Select("COALESCE(SUM(commission_amount), 0)").
                Scan(&totalCommissions)

        var boostRevenue float64
        h.db.Model(&payments.Payment{}).
                Where("status = ? AND kind = ?", "succeeded", "boost").
                Select("COALESCE(SUM(amount), 0)").Scan(&boostRevenue)

        var subscriptionRevenue float64
        h.db.Model(&payments.Payment{}).
                Where("status = ? AND kind = ?", "succeeded", "subscription").
                Select("COALESCE(SUM(amount), 0)").Scan(&subscriptionRevenue)

        var activeSubscriptions int64
        h.db.Table("users").
                Where("subscription_tier != ? AND (subscription_expires_at IS NULL OR subscription_expires_at > NOW()) AND deleted_at IS NULL",
                        "basic").
                Count(&activeSubscriptions)

        response.OK(c, gin.H{
                "total":                 totalRevenue,
                "daily_30days":          daily,
                "total_commissions":     totalCommissions,
                "boost_revenue":         boostRevenue,
                "subscription_revenue":  subscriptionRevenue,
                "active_subscriptions":  activeSubscriptions,
        })
  }

  // ════════════════════════════════════════════════════════════════════════════
  // GET /admin/transactions — all payments + CSV export
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) GetTransactions(c *gin.Context) {
        if !h.requireAdmin(c) { return }
        page, perPage := paginationParams(c)
        q := h.db.Model(&payments.Payment{}).Preload("Escrow")

        if status := c.Query("status"); status != "" {
                q = q.Where("status = ?", status)
        }

        // CSV export (capped at 10,000 rows to prevent OOM)
        if c.Query("export") == "csv" {
                var all []payments.Payment
                q.Order("created_at DESC").Limit(10000).Find(&all)

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
        if !h.requireAdmin(c) { return }
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

  // knownIntegrationKeys lists every integration key the platform can use.
  // Env vars always override DB values.
  var knownIntegrationKeys = []string{
        // Stripe
        "STRIPE_SECRET_KEY", "STRIPE_WEBHOOK_SECRET", "STRIPE_PUBLISHABLE_KEY",
        // PayPal
        "PAYPAL_CLIENT_ID", "PAYPAL_CLIENT_SECRET", "PAYPAL_MODE",
        // Resend (email)
        "RESEND_API_KEY", "RESEND_FROM_EMAIL", "RESEND_FROM_NAME",
        // Firebase push
        "FIREBASE_SERVICE_ACCOUNT_JSON",
        // Cloudflare R2 storage
        "R2_ACCOUNT_ID", "R2_ACCESS_KEY_ID", "R2_SECRET_ACCESS_KEY",
        "R2_BUCKET_NAME", "R2_PUBLIC_URL",
        // Google OAuth
        "GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET",
        // Google Analytics
        "GA_MEASUREMENT_ID",
        // Apple Sign In
        "APPLE_TEAM_ID", "APPLE_KEY_ID", "APPLE_PRIVATE_KEY",
        // Twilio SMS
        "TWILIO_ACCOUNT_SID", "TWILIO_AUTH_TOKEN", "TWILIO_FROM_NUMBER",
        // WhatsApp Business
        "WHATSAPP_API_KEY", "WHATSAPP_PHONE_NUMBER_ID",
        // Redis (for managed Redis in production)
        "REDIS_URL",
  }

  // maskValue returns a masked version of a secret for display (e.g., sk_live_****abc).
  func maskValue(v string) string {
        if len(v) == 0 {
                return ""
        }
        if len(v) <= 8 {
                return "****"
        }
        return v[:4] + "****" + v[len(v)-4:]
  }

  // GetIntegrations returns the status of every integration key.
  // Env vars are checked first; DB values serve as fallback.
  // Raw values are NEVER returned — only masked versions.
  func (h *Handler) GetIntegrations(c *gin.Context) {
        // Load all DB-stored configs
        var rows []IntegrationConfig
        h.db.Find(&rows)
        dbMap := make(map[string]IntegrationConfig, len(rows))
        for _, r := range rows {
                dbMap[r.Key] = r
        }

        out := make([]IntegrationStatus, 0, len(knownIntegrationKeys))
        for _, key := range knownIntegrationKeys {
                status := IntegrationStatus{Key: key, Source: "unset"}
                envVal := os.Getenv(key)
                if envVal != "" {
                        status.Configured = true
                        status.Source = "env"
                        status.Masked = maskValue(envVal)
                } else if row, ok := dbMap[key]; ok && row.Value != "" {
                        status.Configured = true
                        status.Source = "db"
                        status.Masked = maskValue(row.Value)
                        status.UpdatedAt = row.UpdatedAt
                }
                out = append(out, status)
        }
        response.OK(c, out)
  }

  // SaveIntegrations persists integration key-value pairs to the DB.
  // Accepts: { "STRIPE_SECRET_KEY": "sk_live_...", ... }
  // Empty string values delete the DB entry.
  func (h *Handler) SaveIntegrations(c *gin.Context) {
        var body map[string]string
        if err := c.ShouldBindJSON(&body); err != nil {
                response.BadRequest(c, "invalid body")
                return
        }

        // Validate keys are all known
        allowed := make(map[string]bool, len(knownIntegrationKeys))
        for _, k := range knownIntegrationKeys {
                allowed[k] = true
        }

        for key, val := range body {
                if !allowed[key] {
                        response.BadRequest(c, "unknown integration key: "+key)
                        return
                }
                if val == "" {
                        h.db.Delete(&IntegrationConfig{}, "key = ?", key)
                        continue
                }
                h.db.Save(&IntegrationConfig{Key: key, Value: val})
        }

        response.OK(c, gin.H{"saved": len(body)})
  }

  