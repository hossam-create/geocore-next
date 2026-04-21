package subscriptions

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	stripe "github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/subscription"
	"gorm.io/gorm"
)

// Handler provides HTTP handlers for subscription operations
type Handler struct {
	db *gorm.DB
}

// NewHandler creates a new subscriptions handler
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// ListPlans — GET /api/v1/plans
// Public endpoint. Returns all active plans sorted by price.
func (h *Handler) ListPlans(c *gin.Context) {
	var plans []Plan
	h.db.Where("is_active = true").Order("sort_order ASC").Find(&plans)
	response.OK(c, plans)
}

// GetMySubscription — GET /api/v1/subscriptions/me
// Returns the authenticated user's current subscription + plan.
func (h *Handler) GetMySubscription(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	var sub Subscription
	err := h.db.Preload("Plan").Where("user_id = ?", userID).First(&sub).Error
	if err != nil {
		// No subscription → user is on the Free plan
		var freePlan Plan
		h.db.Where("name = ?", "free").First(&freePlan)
		c.JSON(http.StatusOK, gin.H{
			"subscription": nil,
			"plan":         freePlan,
			"on_free_plan": true,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscription": sub,
		"plan":         sub.Plan,
		"on_free_plan": false,
	})
}

// CreateSubscriptionReq defines the body for subscribing to a plan
type CreateSubscriptionReq struct {
	PlanID string `json:"plan_id" binding:"required"`
}

// CreateSubscription — POST /api/v1/subscriptions
// Creates (or updates) a Stripe subscription and returns the checkout URL.
func (h *Handler) CreateSubscription(c *gin.Context) {
	var req CreateSubscriptionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	planUUID, err := uuid.Parse(req.PlanID)
	if err != nil {
		response.BadRequest(c, "invalid plan_id")
		return
	}

	var plan Plan
	if err := h.db.First(&plan, "id = ? AND is_active = true", planUUID).Error; err != nil {
		response.NotFound(c, "plan")
		return
	}

	if plan.Name == "free" {
		response.BadRequest(c, "cannot subscribe to the Free plan")
		return
	}

	if plan.StripePriceID == "" || os.Getenv("STRIPE_SECRET_KEY") == "" {
		// Non-Stripe env: create a local subscription record directly
		userID, _ := uuid.Parse(c.GetString("user_id"))
		h.upsertLocalSubscription(userID, plan)
		response.Created(c, gin.H{
			"message":      "Subscription activated (non-Stripe environment)",
			"plan":         plan.Name,
			"checkout_url": "",
		})
		return
	}

	// Stripe path
	userID, _ := uuid.Parse(c.GetString("user_id"))
	var userRow struct {
		Email            string
		StripeCustomerID string
	}
	h.db.Table("users").Select("email, stripe_customer_id").Where("id = ?", userID).First(&userRow)

	stripeCustomerID := userRow.StripeCustomerID
	if stripeCustomerID == "" {
		response.BadRequest(c, "no Stripe customer on record — initiate a payment first")
		return
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	params := &stripe.SubscriptionParams{
		Customer: stripe.String(stripeCustomerID),
		Items: []*stripe.SubscriptionItemsParams{
			{Price: stripe.String(plan.StripePriceID)},
		},
		PaymentBehavior: stripe.String("default_incomplete"),
	}
	params.AddExpand("latest_invoice.payment_intent")

	sub, err := subscription.New(params)
	if err != nil {
		slog.Error("subscriptions: stripe subscription failed",
			"user_id", userID.String(), "plan", plan.Name, "error", err.Error())
		response.BadRequest(c, fmt.Sprintf("Stripe error: %v", err))
		return
	}

	// Persist subscription record
	now := time.Now()
	s := Subscription{
		UserID:               userID,
		PlanID:               plan.ID,
		Status:               StatusIncomplete,
		StripeSubscriptionID: sub.ID,
		StripeCustomerID:     stripeCustomerID,
		CurrentPeriodStart:   &now,
	}
	h.db.Where("user_id = ?", userID).Assign(s).FirstOrCreate(&s)

	clientSecret := ""
	if sub.LatestInvoice != nil && sub.LatestInvoice.PaymentIntent != nil {
		clientSecret = sub.LatestInvoice.PaymentIntent.ClientSecret
	}

	response.Created(c, gin.H{
		"subscription_id": sub.ID,
		"client_secret":   clientSecret,
		"status":          string(sub.Status),
	})
}

// CancelSubscription — DELETE /api/v1/subscriptions/me
// Cancels at period end (graceful cancellation).
func (h *Handler) CancelSubscription(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	var sub Subscription
	if err := h.db.Where("user_id = ? AND status = ?", userID, StatusActive).First(&sub).Error; err != nil {
		response.NotFound(c, "active subscription")
		return
	}

	// Cancel in Stripe if applicable
	if sub.StripeSubscriptionID != "" && os.Getenv("STRIPE_SECRET_KEY") != "" {
		stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
		params := &stripe.SubscriptionParams{CancelAtPeriodEnd: stripe.Bool(true)}
		if _, err := subscription.Update(sub.StripeSubscriptionID, params); err != nil {
			slog.Error("subscriptions: stripe cancel failed",
				"sub_id", sub.StripeSubscriptionID, "error", err.Error())
		}
	}

	now := time.Now()
	h.db.Model(&sub).Updates(map[string]interface{}{
		"cancel_at_period_end": true,
		"cancelled_at":         now,
	})

	response.OK(c, gin.H{
		"message":              "Subscription will be cancelled at the end of the billing period",
		"cancel_at_period_end": true,
		"current_period_end":   sub.CurrentPeriodEnd,
	})
}

// HandleStripeSubscriptionUpdated is called from the webhook for subscription events.
func HandleStripeSubscriptionUpdated(db *gorm.DB, stripeSub *stripe.Subscription) {
	var sub Subscription
	err := db.Where("stripe_subscription_id = ?", stripeSub.ID).First(&sub).Error
	if err != nil {
		slog.Warn("subscriptions: webhook — sub not found", "stripe_sub_id", stripeSub.ID)
		return
	}

	updates := map[string]interface{}{
		"status":                mapStripeStatus(stripeSub.Status),
		"cancel_at_period_end":  stripeSub.CancelAtPeriodEnd,
		"updated_at":            time.Now(),
	}
	if stripeSub.CurrentPeriodStart > 0 {
		t := time.Unix(stripeSub.CurrentPeriodStart, 0)
		updates["current_period_start"] = t
	}
	if stripeSub.CurrentPeriodEnd > 0 {
		t := time.Unix(stripeSub.CurrentPeriodEnd, 0)
		updates["current_period_end"] = t
	}
	if stripeSub.Status == stripe.SubscriptionStatusActive {
		updates["status"] = StatusActive
	}

	db.Model(&sub).Updates(updates)
	slog.Info("subscriptions: webhook updated", "stripe_sub_id", stripeSub.ID, "status", string(stripeSub.Status))
}

// HandleStripeSubscriptionDeleted marks a subscription as cancelled when deleted via Stripe.
func HandleStripeSubscriptionDeleted(db *gorm.DB, stripeSub *stripe.Subscription) {
	now := time.Now()
	db.Model(&Subscription{}).
		Where("stripe_subscription_id = ?", stripeSub.ID).
		Updates(map[string]interface{}{
			"status":       StatusCancelled,
			"cancelled_at": now,
			"updated_at":   now,
		})
	slog.Info("subscriptions: webhook deleted", "stripe_sub_id", stripeSub.ID)
}

// upsertLocalSubscription is used in non-Stripe environments to create subscriptions directly.
func (h *Handler) upsertLocalSubscription(userID uuid.UUID, plan Plan) {
	now := time.Now()
	end := now.AddDate(0, 1, 0) // 1 month
	s := Subscription{
		UserID:             userID,
		PlanID:             plan.ID,
		Status:             StatusActive,
		CurrentPeriodStart: &now,
		CurrentPeriodEnd:   &end,
	}
	h.db.Where("user_id = ?", userID).Assign(s).FirstOrCreate(&s)
}

func mapStripeStatus(s stripe.SubscriptionStatus) SubscriptionStatus {
	switch s {
	case stripe.SubscriptionStatusActive:
		return StatusActive
	case stripe.SubscriptionStatusCanceled:
		return StatusCancelled
	case stripe.SubscriptionStatusPastDue:
		return StatusPastDue
	case stripe.SubscriptionStatusTrialing:
		return StatusTrialing
	case stripe.SubscriptionStatusIncomplete:
		return StatusIncomplete
	case stripe.SubscriptionStatusUnpaid:
		return StatusUnpaid
	default:
		return StatusActive
	}
}
