package admin

import (
	"time"

	"github.com/geocore-next/backend/internal/fees"
	"github.com/geocore-next/backend/internal/reports"
	"github.com/geocore-next/backend/internal/settlement"
	"github.com/geocore-next/backend/pkg/jobs"
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterRoutes mounts all /admin endpoints.
// All routes require: Auth() + AdminOnly() (role must be "admin" or "super_admin")
// Rate limited: 60 requests/minute per user (stricter than public 120/min).
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client, jobQueue *jobs.JobQueue) {
	h := NewHandler(db)
	dlqH := NewDLQHandler(jobQueue)

	rl := middleware.NewRateLimiter(rdb)

	adm := r.Group("/admin")
	adm.Use(middleware.Auth(), middleware.AdminWithDB(db), rl.LimitByUser(60, time.Minute, "admin:global"))
	{
		// Dashboard
		adm.GET("/stats", middleware.RequireAnyPermission(middleware.PermAdminDashboardRead), h.GetStats)
		adm.GET("/dashboard", middleware.RequireAnyPermission(middleware.PermAdminDashboardRead), h.GetDashboardFull)

		// Users
		adm.GET("/users", middleware.RequireAnyPermission(middleware.PermUsersRead), h.ListUsers)
		adm.GET("/users/:id", middleware.RequireAnyPermission(middleware.PermUsersRead), h.GetUser)
		adm.PUT("/users/:id", middleware.RequireAnyPermission(middleware.PermUsersWrite), h.UpdateUser)
		adm.DELETE("/users/:id", middleware.RequireAnyPermission(middleware.PermUsersDelete), h.DeleteUser)
		adm.POST("/users/:id/ban", middleware.RequireAnyPermission(middleware.PermUsersBan), h.BanUser)
		adm.POST("/users/:id/unban", middleware.RequireAnyPermission(middleware.PermUsersBan), h.UnbanUser)
		adm.PUT("/users/:id/suspend", middleware.RequireAnyPermission(middleware.PermUsersBan), h.SuspendUser)
		adm.PUT("/users/:id/verify", middleware.RequireAnyPermission(middleware.PermUsersWrite), h.VerifyUser)
		adm.PUT("/users/:id/role", middleware.RequireAnyPermission(middleware.PermUsersWrite), h.ChangeUserRole)
		adm.PUT("/users/:id/group", middleware.RequireAnyPermission(middleware.PermUsersWrite), h.ChangeUserGroup)
		adm.GET("/users/:id/listings", middleware.RequireAnyPermission(middleware.PermUsersRead), h.GetUserListings)
		adm.GET("/users/:id/orders", middleware.RequireAnyPermission(middleware.PermUsersRead), h.GetUserOrders)
		adm.POST("/users/:id/impersonate", middleware.RequireAnyPermission(middleware.PermUsersWrite), h.ImpersonateUser)

		// User groups
		adm.GET("/user-groups", middleware.RequireAnyPermission(middleware.PermUsersRead), h.ListUserGroups)
		adm.POST("/user-groups", middleware.RequireAnyPermission(middleware.PermUsersWrite), h.CreateUserGroup)
		adm.PUT("/user-groups/:id", middleware.RequireAnyPermission(middleware.PermUsersWrite), h.UpdateUserGroup)
		adm.DELETE("/user-groups/:id", middleware.RequireAnyPermission(middleware.PermUsersWrite), h.DeleteUserGroup)

		// User custom fields
		adm.GET("/user-fields", middleware.RequireAnyPermission(middleware.PermUsersRead), h.ListUserCustomFields)
		adm.POST("/user-fields", middleware.RequireAnyPermission(middleware.PermUsersWrite), h.CreateUserCustomField)
		adm.PUT("/user-fields/:id", middleware.RequireAnyPermission(middleware.PermUsersWrite), h.UpdateUserCustomField)
		adm.DELETE("/user-fields/:id", middleware.RequireAnyPermission(middleware.PermUsersWrite), h.DeleteUserCustomField)

		// Listings moderation
		adm.GET("/listings", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.ListListings)
		adm.GET("/listings/pending", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.ListPendingListings)
		adm.GET("/listings/:id", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.GetListing)
		adm.PUT("/listings/:id", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.UpdateListing)
		adm.PUT("/listings/:id/approve", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.ApproveListing)
		adm.PUT("/listings/:id/reject", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.RejectListing)
		adm.PUT("/listings/:id/feature", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.FeatureListing)
		adm.PUT("/listings/:id/extend", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.ExtendListing)
		adm.POST("/listings/:id/extras", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.AddListingExtra)
		adm.DELETE("/listings/:id/extras/:extraId", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.RemoveListingExtra)
		adm.DELETE("/listings/:id", middleware.RequireAnyPermission(middleware.PermListingsDelete), h.DeleteListing)
		adm.POST("/listings/bulk-approve", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.BulkApproveListings)
		adm.POST("/listings/bulk-reject", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.BulkRejectListings)
		adm.POST("/listings/bulk-delete", middleware.RequireAnyPermission(middleware.PermListingsDelete), h.BulkDeleteListings)

		// Listing extras management
		adm.GET("/listing-extras", middleware.RequireAnyPermission(middleware.PermPlansManage), h.ListListingExtras)
		adm.POST("/listing-extras", middleware.RequireAnyPermission(middleware.PermPlansManage), h.CreateListingExtra)
		adm.PUT("/listing-extras/:id", middleware.RequireAnyPermission(middleware.PermPlansManage), h.UpdateListingExtra)
		adm.DELETE("/listing-extras/:id", middleware.RequireAnyPermission(middleware.PermPlansManage), h.DeleteListingExtra)

		// Auctions management
		adm.GET("/auctions", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.ListAuctions)
		adm.GET("/auctions/pending", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.ListPendingAuctions)
		adm.GET("/auctions/:id", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.GetAuction)
		adm.PUT("/auctions/:id/approve", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.ApproveAuction)
		adm.PUT("/auctions/:id/reject", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.RejectAuction)
		adm.PUT("/auctions/:id/cancel", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.CancelAuction)
		adm.PUT("/auctions/:id/extend", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.ExtendAuction)
		adm.GET("/auctions/:id/bids", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.GetAuctionBids)
		adm.DELETE("/auctions/:id/bids/:bidId", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.DeleteAuctionBid)

		// Revenue & transactions
		adm.GET("/revenue", middleware.RequireAnyPermission(middleware.PermFinanceRead), h.GetRevenue)
		adm.GET("/transactions", middleware.RequireAnyPermission(middleware.PermFinanceRead), h.GetTransactions)    // ?export=csv for CSV download
		adm.GET("/finance/report", middleware.RequireAnyPermission(middleware.PermFinanceRead), h.GetFinanceReport) // ?format=csv|pdf&from=&to=

		// Audit logs
		adm.GET("/logs", middleware.RequireAnyPermission(middleware.PermAuditLogsRead), h.GetAuditLogs)

		// Category management
		adm.GET("/categories", middleware.RequireAnyPermission(middleware.PermCatalogManage), h.ListCategories)
		adm.POST("/categories", middleware.RequireAnyPermission(middleware.PermCatalogManage), h.CreateCategory)
		adm.PUT("/categories/:id", middleware.RequireAnyPermission(middleware.PermCatalogManage), h.UpdateCategory)
		adm.DELETE("/categories/:id", middleware.RequireAnyPermission(middleware.PermCatalogManage), h.DeleteCategory)
		adm.PUT("/categories/:id/reorder", middleware.RequireAnyPermission(middleware.PermCatalogManage), h.ReorderCategory)
		adm.GET("/categories/:id/fields", middleware.RequireAnyPermission(middleware.PermCatalogManage), h.ListCategoryFields)
		adm.POST("/categories/:id/fields", middleware.RequireAnyPermission(middleware.PermCatalogManage), h.CreateCategoryField)
		adm.PUT("/categories/:id/fields/:fieldId", middleware.RequireAnyPermission(middleware.PermCatalogManage), h.UpdateCategoryField)
		adm.DELETE("/categories/:id/fields/:fieldId", middleware.RequireAnyPermission(middleware.PermCatalogManage), h.DeleteCategoryField)

		// Plans management
		adm.GET("/plans", middleware.RequireAnyPermission(middleware.PermPlansManage), h.ListPlans)
		adm.POST("/plans", middleware.RequireAnyPermission(middleware.PermPlansManage), h.CreatePlan)
		adm.PUT("/plans/:id", middleware.RequireAnyPermission(middleware.PermPlansManage), h.UpdatePlan)
		adm.DELETE("/plans/:id", middleware.RequireAnyPermission(middleware.PermPlansManage), h.DeletePlan)

		// Payment gateways
		adm.GET("/payment-gateways", middleware.RequireAnyPermission(middleware.PermFinanceRead), h.ListPaymentGateways)
		adm.PUT("/payment-gateways/:id", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.UpdatePaymentGateway)

		// Invoices
		adm.GET("/invoices", middleware.RequireAnyPermission(middleware.PermFinanceRead), h.ListInvoices)
		adm.GET("/invoices/:id", middleware.RequireAnyPermission(middleware.PermFinanceRead), h.GetInvoice)

		// Discount codes
		adm.GET("/discount-codes", middleware.RequireAnyPermission(middleware.PermPlansManage), h.ListDiscountCodes)
		adm.POST("/discount-codes", middleware.RequireAnyPermission(middleware.PermPlansManage), h.CreateDiscountCode)
		adm.PUT("/discount-codes/:id", middleware.RequireAnyPermission(middleware.PermPlansManage), h.UpdateDiscountCode)
		adm.DELETE("/discount-codes/:id", middleware.RequireAnyPermission(middleware.PermPlansManage), h.DeleteDiscountCode)

		// Email templates
		adm.GET("/email-templates", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.ListEmailTemplates)
		adm.GET("/email-templates/:slug", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.GetEmailTemplate)
		adm.PUT("/email-templates/:slug", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.UpdateEmailTemplate)
		adm.POST("/email-templates/:slug/preview", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.PreviewEmailTemplate)
		adm.POST("/email-templates/:slug/test", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.TestEmailTemplate)

		// Static pages
		adm.GET("/pages", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.ListStaticPages)
		adm.GET("/pages/:id", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.GetStaticPage)
		adm.POST("/pages", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.CreateStaticPage)
		adm.PUT("/pages/:id", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.UpdateStaticPage)
		adm.DELETE("/pages/:id", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.DeleteStaticPage)

		// Announcements
		adm.GET("/announcements", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.ListAnnouncements)
		adm.POST("/announcements", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.CreateAnnouncement)
		adm.PUT("/announcements/:id", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.UpdateAnnouncement)
		adm.DELETE("/announcements/:id", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.DeleteAnnouncement)

		// Geography
		adm.GET("/geography", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.ListGeoRegions)
		adm.GET("/geography/:id/children", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.GetGeoChildren)
		adm.POST("/geography", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.CreateGeoRegion)
		adm.PUT("/geography/:id", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.UpdateGeoRegion)
		adm.DELETE("/geography/:id", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.DeleteGeoRegion)

		// Storefronts admin
		adm.GET("/storefronts", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.ListStorefronts)
		adm.GET("/storefronts/:id", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.GetStorefront)
		adm.PUT("/storefronts/:id/approve", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.ApproveStorefront)
		adm.PUT("/storefronts/:id/suspend", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.SuspendStorefront)
		adm.PUT("/storefronts/:id/feature", middleware.RequireAnyPermission(middleware.PermListingsModerate), h.FeatureStorefront)
		adm.DELETE("/storefronts/:id", middleware.RequireAnyPermission(middleware.PermListingsDelete), h.DeleteStorefront)

		// Reports queue
		reports.RegisterAdminRoutes(adm, db)

		// Dead Letter Queue (DLQ) admin
		adm.GET("/dlq", middleware.RequireAnyPermission(middleware.PermAuditLogsRead), dlqH.ListFailedJobs)
		adm.POST("/dlq/retry", middleware.RequireAnyPermission(middleware.PermAuditLogsRead), dlqH.RetryAllFailed)
		adm.POST("/dlq/:id/retry", middleware.RequireAnyPermission(middleware.PermAuditLogsRead), dlqH.RetryOneFailed)
		adm.DELETE("/dlq", middleware.RequireAnyPermission(middleware.PermAuditLogsRead), dlqH.PurgeDLQ)

		// Payout admin (settlement)
		settlement.RegisterAdminRoutes(adm, db)

		// Fee Engine admin
		feeH := fees.NewHandler(db)
		adm.GET("/fees", middleware.RequireAnyPermission(middleware.PermSettingsRead), feeH.ListFees)
		adm.POST("/fees", middleware.RequireAnyPermission(middleware.PermSettingsWrite), feeH.CreateFee)
		adm.PUT("/fees/:id", middleware.RequireAnyPermission(middleware.PermSettingsWrite), feeH.UpdateFee)
		adm.DELETE("/fees/:id", middleware.RequireAnyPermission(middleware.PermSettingsWrite), feeH.DeleteFee)
		adm.GET("/fees/calculate", middleware.RequireAnyPermission(middleware.PermSettingsRead), feeH.CalculateFee)

		// Sprint 8.5: Production Safety Controls
		adm.POST("/users/:id/freeze", middleware.RequireAnyPermission(middleware.PermUsersBan), h.FreezeUserHandler)
		adm.POST("/users/:id/unfreeze", middleware.RequireAnyPermission(middleware.PermUsersBan), h.UnfreezeUserHandler)
		adm.POST("/wallet/adjust", middleware.RequireAnyPermission(middleware.PermFinanceRead), h.AdjustWalletHandler)
		adm.POST("/override/transaction", middleware.RequireAnyPermission(middleware.PermFinanceRead), h.OverrideTransactionHandler)
		adm.GET("/audit-log", middleware.RequireAnyPermission(middleware.PermAuditLogsRead), h.GetAuditLogHandler)
	}
}
