package monetization

import (
        "fmt"
        "log/slog"
        "net/http"
        "os"
        "time"

        "github.com/geocore-next/backend/pkg/response"
        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "math"

        "github.com/stripe/stripe-go/v79"
        "github.com/stripe/stripe-go/v79/customer"
        "github.com/stripe/stripe-go/v79/paymentintent"
        strprice "github.com/stripe/stripe-go/v79/price"
        strsub "github.com/stripe/stripe-go/v79/subscription"
        "gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Handler
// ════════════════════════════════════════════════════════════════════════════

type Handler struct {
        db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
        return &Handler{db: db}
}

// stripeEnabled reports whether the Stripe secret key has been configured.
func stripeEnabled() bool {
        return os.Getenv("STRIPE_SECRET_KEY") != ""
}

// ════════════════════════════════════════════════════════════════════════════
// POST /listings/:id/boost — create a Stripe PaymentIntent to feature a listing
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) BoostListing(c *gin.Context) {
        if !stripeEnabled() {
                c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
                        "error":   "stripe_not_configured",
                        "message": "Listing boosts require Stripe to be configured. Set STRIPE_SECRET_KEY.",
                })
                return
        }

        listingID, err := uuid.Parse(c.Param("id"))
        if err != nil {
                response.BadRequest(c, "invalid listing id")
                return
        }

        userID := c.GetString("user_id")

        // Verify listing exists and belongs to the requester
        var listing struct {
                ID     uuid.UUID
                UserID uuid.UUID
                Title  string
        }
        if err := h.db.Table("listings").
                Select("id, user_id, title").
                Where("id = ? AND deleted_at IS NULL", listingID).
                First(&listing).Error; err != nil {
                response.NotFound(c, "listing")
                return
        }
        if listing.UserID.String() != userID {
                response.Forbidden(c)
                return
        }

        settings := GetSettings(h.db)
        boostFee := settings.BoostFeeUSD
        if boostFee <= 0 {
                boostFee = BoostFee
        }

        stripeCustomerID, err := h.ensureStripeCustomer(userID)
        if err != nil {
                slog.Error("monetization: failed to ensure Stripe customer for boost",
                        "user_id", userID, "error", err.Error())
                response.InternalError(c, err)
                return
        }

        // Create Stripe PaymentIntent for the boost fee
        amountSmallest := int64(math.Round(boostFee * 100))
        piParams := &stripe.PaymentIntentParams{
                Amount:      stripe.Int64(amountSmallest),
                Currency:    stripe.String(BoostCurrency),
                Description: stripe.String(fmt.Sprintf("Listing boost: %s", listing.Title)),
                AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
                        Enabled: stripe.Bool(true),
                },
        }
        if stripeCustomerID != "" {
                piParams.Customer = stripe.String(stripeCustomerID)
        }
        piParams.AddMetadata("listing_id", listingID.String())
        piParams.AddMetadata("user_id", userID)
        piParams.AddMetadata("kind", "boost")

        pi, err := paymentintent.New(piParams)
        if err != nil {
                slog.Error("monetization: Stripe boost payment intent failed",
                        "listing_id", listingID, "user_id", userID, "error", err.Error())
                response.BadRequest(c, "Stripe error: "+err.Error())
                return
        }

        // Persist a payments record for the boost
        h.db.Table("payments").Create(map[string]interface{}{
                "id":                       uuid.New(),
                "user_id":                  userID,
                "listing_id":               listingID,
                "kind":                     "boost",
                "stripe_payment_intent_id": pi.ID,
                "stripe_client_secret":     pi.ClientSecret,
                "amount":                   boostFee,
                "currency":                 "USD",
                "status":                   "pending",
                "description":              fmt.Sprintf("Listing boost (%d days): %s", BoostDays, listing.Title),
                "created_at":               time.Now(),
                "updated_at":               time.Now(),
        })

        slog.Info("monetization: boost payment intent created",
                "listing_id", listingID, "user_id", userID, "amount", boostFee)

        response.Created(c, gin.H{
                "payment_intent_id": pi.ID,
                "client_secret":     pi.ClientSecret,
                "amount":            boostFee,
                "currency":          "USD",
                "boost_days":        BoostDays,
        })
}

// ════════════════════════════════════════════════════════════════════════════
// POST /listings/:id/boost/confirm — confirm boost payment, activate feature
// ════════════════════════════════════════════════════════════════════════════

type ConfirmBoostReq struct {
        PaymentIntentID string `json:"payment_intent_id" binding:"required"`
}

func (h *Handler) ConfirmBoost(c *gin.Context) {
        if !stripeEnabled() {
                c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
                        "error":   "stripe_not_configured",
                        "message": "Listing boosts require Stripe to be configured.",
                })
                return
        }

        listingID, err := uuid.Parse(c.Param("id"))
        if err != nil {
                response.BadRequest(c, "invalid listing id")
                return
        }

        var req ConfirmBoostReq
        if err := c.ShouldBindJSON(&req); err != nil {
                response.BadRequest(c, err.Error())
                return
        }

        userID := c.GetString("user_id")

        // Verify ownership — use Scan into a struct to avoid GORM Pluck scalar issue
        var listingOwner struct{ UserID string }
        if err := h.db.Table("listings").Select("user_id").
                Where("id = ? AND deleted_at IS NULL", listingID).
                Scan(&listingOwner).Error; err != nil || listingOwner.UserID == "" {
                response.NotFound(c, "listing")
                return
        }
        if listingOwner.UserID != userID {
                response.Forbidden(c)
                return
        }

        // Validate PI belongs to this caller for this listing (prevents PI reuse)
        var pendingBoost struct {
                ID        string
                ListingID *string
        }
        if err := h.db.Table("payments").
                Select("id, listing_id").
                Where("stripe_payment_intent_id = ? AND user_id = ? AND kind = ? AND status = ?",
                        req.PaymentIntentID, userID, "boost", "pending").
                First(&pendingBoost).Error; err != nil {
                c.AbortWithStatusJSON(403, gin.H{
                        "error":   "payment_not_found",
                        "message": "Payment not found or does not belong to you.",
                })
                return
        }
        if pendingBoost.ListingID == nil || *pendingBoost.ListingID != listingID.String() {
                c.AbortWithStatusJSON(403, gin.H{
                        "error":   "payment_listing_mismatch",
                        "message": "Payment was created for a different listing.",
                })
                return
        }

        // Retrieve PI status from Stripe
        pi, err := paymentintent.Get(req.PaymentIntentID, nil)
        if err != nil {
                response.BadRequest(c, "Stripe error: "+err.Error())
                return
        }

        if pi.Status != stripe.PaymentIntentStatusSucceeded {
                response.BadRequest(c, fmt.Sprintf("payment not yet succeeded (status: %s)", pi.Status))
                return
        }

        // Mark listing as featured with expiry
        featuredUntil := time.Now().Add(BoostDays * 24 * time.Hour)
        if err := h.db.Table("listings").
                Where("id = ?", listingID).
                Updates(map[string]interface{}{
                        "is_featured":    true,
                        "featured_until": featuredUntil,
                }).Error; err != nil {
                response.InternalError(c, err)
                return
        }

        // Mark the boost payment as succeeded
        h.db.Table("payments").
                Where("stripe_payment_intent_id = ?", req.PaymentIntentID).
                Updates(map[string]interface{}{"status": "succeeded"})

        slog.Info("monetization: listing boosted",
                "listing_id", listingID, "user_id", userID, "featured_until", featuredUntil)

        response.OK(c, gin.H{
                "listing_id":     listingID,
                "is_featured":    true,
                "featured_until": featuredUntil,
                "message":        fmt.Sprintf("Your listing is now featured for %d days.", BoostDays),
        })
}

// ════════════════════════════════════════════════════════════════════════════
// POST /subscriptions/upgrade — upgrade seller to Pro or Business tier
// ════════════════════════════════════════════════════════════════════════════

type UpgradeReq struct {
        Tier TierName `json:"tier" binding:"required"`
}

func (h *Handler) UpgradeSubscription(c *gin.Context) {
        if !stripeEnabled() {
                c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
                        "error":   "stripe_not_configured",
                        "message": "Seller subscriptions require Stripe to be configured. Set STRIPE_SECRET_KEY.",
                })
                return
        }

        var req UpgradeReq
        if err := c.ShouldBindJSON(&req); err != nil {
                response.BadRequest(c, err.Error())
                return
        }
        if req.Tier != TierPro && req.Tier != TierBusiness {
                response.BadRequest(c, "tier must be 'pro' or 'business'")
                return
        }

        userID := c.GetString("user_id")

        var fee float64
        var productName string
        switch req.Tier {
        case TierPro:
                fee = ProMonthlyFee
                productName = "GeoCore Pro"
        case TierBusiness:
                fee = BusinessMonthlyFee
                productName = "GeoCore Business"
        }

        stripeCustomerID, err := h.ensureStripeCustomer(userID)
        if err != nil {
                response.InternalError(c, err)
                return
        }

        // Create a real Stripe Subscription with default_incomplete so the frontend
        // collects payment before the subscription activates. Inline price data avoids
        // requiring a pre-configured Price in the Stripe dashboard.
        priceID, err := h.ensureStripePrice(req.Tier, fee, productName)
        if err != nil {
                slog.Error("monetization: failed to ensure Stripe price",
                        "user_id", userID, "tier", req.Tier, "error", err.Error())
                response.BadRequest(c, "Stripe error: "+err.Error())
                return
        }

        subParams := &stripe.SubscriptionParams{
                Customer:        stripe.String(stripeCustomerID),
                PaymentBehavior: stripe.String("default_incomplete"),
                Items: []*stripe.SubscriptionItemsParams{
                        {Price: stripe.String(priceID)},
                },
                Expand: []*string{stripe.String("latest_invoice.payment_intent")},
        }
        subParams.AddMetadata("user_id", userID)
        subParams.AddMetadata("tier", string(req.Tier))

        sub, err := strsub.New(subParams)
        if err != nil {
                slog.Error("monetization: Stripe subscription creation failed",
                        "user_id", userID, "tier", req.Tier, "error", err.Error())
                response.BadRequest(c, "Stripe error: "+err.Error())
                return
        }

        // Extract client_secret from the latest invoice's payment intent
        var clientSecret string
        var piID string
        if sub.LatestInvoice != nil && sub.LatestInvoice.PaymentIntent != nil {
                clientSecret = sub.LatestInvoice.PaymentIntent.ClientSecret
                piID = sub.LatestInvoice.PaymentIntent.ID
        }

        // Persist a payment record for accounting and amount validation on confirm
        h.db.Table("payments").Create(map[string]interface{}{
                "id":                       uuid.New(),
                "user_id":                  userID,
                "kind":                     "subscription",
                "stripe_payment_intent_id": piID,
                "stripe_client_secret":     clientSecret,
                "amount":                   fee,
                "currency":                 "USD",
                "status":                   "pending",
                "description":              fmt.Sprintf("Subscription upgrade to %s tier", req.Tier),
                "created_at":               time.Now(),
                "updated_at":               time.Now(),
        })

        // Create a pending SellerSubscription record with stripe_sub_id for lifecycle tracking
        userUUID, _ := uuid.Parse(userID)
        var existingSub SellerSubscription
        if err := h.db.Where("user_id = ?", userUUID).First(&existingSub).Error; err != nil {
                h.db.Create(&SellerSubscription{
                        UserID:      userUUID,
                        Tier:        req.Tier,
                        StripeSubID: sub.ID,
                        StartsAt:    time.Now(),
                })
        } else {
                h.db.Model(&existingSub).Updates(map[string]interface{}{
                        "tier":          req.Tier,
                        "stripe_sub_id": sub.ID,
                        "starts_at":     time.Now(),
                })
        }

        slog.Info("monetization: Stripe subscription created",
                "user_id", userID, "tier", req.Tier, "stripe_sub_id", sub.ID, "status", sub.Status)

        response.Created(c, gin.H{
                "stripe_subscription_id": sub.ID,
                "stripe_status":          sub.Status,
                "client_secret":          clientSecret,
                "tier":                   req.Tier,
                "amount":                 fee,
                "currency":               "USD",
                "message":                "Complete payment to activate your subscription.",
        })
}

// ════════════════════════════════════════════════════════════════════════════
// POST /subscriptions/confirm — activate tier after payment succeeds
// ════════════════════════════════════════════════════════════════════════════

type ConfirmSubReq struct {
        StripeSubID string   `json:"stripe_subscription_id" binding:"required"`
        Tier        TierName `json:"tier" binding:"required"`
}

func (h *Handler) ConfirmSubscription(c *gin.Context) {
        if !stripeEnabled() {
                c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
                        "error":   "stripe_not_configured",
                        "message": "Seller subscriptions require Stripe to be configured.",
                })
                return
        }

        var req ConfirmSubReq
        if err := c.ShouldBindJSON(&req); err != nil {
                response.BadRequest(c, err.Error())
                return
        }
        if req.Tier != TierPro && req.Tier != TierBusiness {
                response.BadRequest(c, "tier must be 'pro' or 'business'")
                return
        }

        userID := c.GetString("user_id")
        userUUID, _ := uuid.Parse(userID)

        // Validate the Stripe Subscription belongs to this caller
        var subRecord SellerSubscription
        if err := h.db.Where("stripe_sub_id = ? AND user_id = ?", req.StripeSubID, userUUID).
                First(&subRecord).Error; err != nil {
                c.AbortWithStatusJSON(403, gin.H{
                        "error":   "subscription_not_found",
                        "message": "Subscription not found or does not belong to you.",
                })
                return
        }
        // Verify requested tier matches what was stored
        if subRecord.Tier != req.Tier {
                c.AbortWithStatusJSON(403, gin.H{
                        "error":   "tier_mismatch",
                        "message": "Subscription was created for a different tier.",
                })
                return
        }

        // Retrieve subscription from Stripe and verify it is active
        sub, err := strsub.Get(req.StripeSubID, &stripe.SubscriptionParams{
                Expand: []*string{stripe.String("latest_invoice.payment_intent")},
        })
        if err != nil {
                response.BadRequest(c, "Stripe error: "+err.Error())
                return
        }
        if sub.Status != stripe.SubscriptionStatusActive {
                response.BadRequest(c, fmt.Sprintf("subscription not yet active (status: %s)", sub.Status))
                return
        }
        // Verify Stripe metadata tier matches
        if metaTier, ok := sub.Metadata["tier"]; ok && metaTier != string(req.Tier) {
                c.AbortWithStatusJSON(403, gin.H{
                        "error":   "tier_mismatch",
                        "message": "Subscription tier does not match.",
                })
                return
        }

        now := time.Now()
        // Expiry = next billing cycle (1 month from now as approximation; real expiry is driven by Stripe webhooks)
        expiresAt := time.Unix(sub.CurrentPeriodEnd, 0)

        // Activate the SellerSubscription record
        h.db.Model(&subRecord).Updates(map[string]interface{}{
                "tier":       req.Tier,
                "starts_at":  now,
                "expires_at": expiresAt,
        })

        // Mirror tier and expiry on user record for fast lookup at listing-creation time
        h.db.Table("users").Where("id = ?", userID).
                Updates(map[string]interface{}{
                        "subscription_tier":       string(req.Tier),
                        "subscription_expires_at": expiresAt,
                })

        // Mark the pending payment record as succeeded (matched via stripe_sub_id payment record)
        if sub.LatestInvoice != nil && sub.LatestInvoice.PaymentIntent != nil {
                h.db.Table("payments").
                        Where("stripe_payment_intent_id = ?", sub.LatestInvoice.PaymentIntent.ID).
                        Updates(map[string]interface{}{"status": "succeeded"})
        }

        slog.Info("monetization: subscription activated",
                "user_id", userID, "tier", req.Tier, "stripe_sub_id", req.StripeSubID, "expires_at", expiresAt)

        limits := Limits(req.Tier)
        response.OK(c, gin.H{
                "tier":                   req.Tier,
                "stripe_subscription_id": req.StripeSubID,
                "starts_at":              now,
                "expires_at":             expiresAt,
                "limits":                 limits,
                "message":                fmt.Sprintf("Subscription upgraded to %s.", req.Tier),
        })
}
// ════════════════════════════════════════════════════════════════════════════
// GET /subscriptions/me — current user subscription info
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) GetMySubscription(c *gin.Context) {
        userID := c.GetString("user_id")

        var sub struct {
                SubscriptionTier      string
                SubscriptionExpiresAt *time.Time
        }
        h.db.Table("users").
                Select("subscription_tier, subscription_expires_at").
                Where("id = ? AND deleted_at IS NULL", userID).
                Scan(&sub)

        tier := TierName(sub.SubscriptionTier)
        if tier == "" {
                tier = TierBasic
        }
        limits := Limits(tier)

        isActive := true
        if sub.SubscriptionExpiresAt != nil && sub.SubscriptionExpiresAt.Before(time.Now()) && tier != TierBasic {
                isActive = false
                tier = TierBasic
                limits = Limits(TierBasic)
        }

        response.OK(c, gin.H{
                "tier":       tier,
                "expires_at": sub.SubscriptionExpiresAt,
                "is_active":  isActive,
                "limits":     limits,
        })
}

// ════════════════════════════════════════════════════════════════════════════
// Internal helpers
// ════════════════════════════════════════════════════════════════════════════

// ensureStripeCustomer returns the user's existing Stripe customer ID or creates
// a new one, persisting the result on the users table.
func (h *Handler) ensureStripeCustomer(userID string) (string, error) {
        var row struct {
                Email            string
                Name             string
                StripeCustomerID string
        }
        if err := h.db.Table("users").
                Select("email, name, stripe_customer_id").
                Where("id = ? AND deleted_at IS NULL", userID).
                Scan(&row).Error; err != nil {
                return "", fmt.Errorf("load user: %w", err)
        }
        if row.StripeCustomerID != "" {
                return row.StripeCustomerID, nil
        }

        cust, err := customer.New(&stripe.CustomerParams{
                Email: stripe.String(row.Email),
                Name:  stripe.String(row.Name),
        })
        if err != nil {
                return "", fmt.Errorf("stripe: create customer: %w", err)
        }

        h.db.Table("users").Where("id = ?", userID).
                Update("stripe_customer_id", cust.ID)
        return cust.ID, nil
}

// ensureStripePrice returns a Stripe Price ID for the given tier, creating one if it
// does not already exist. A deterministic lookup key ensures at most one Price per
// tier exists in the Stripe account.
func (h *Handler) ensureStripePrice(tier TierName, fee float64, productName string) (string, error) {
        lookupKey := fmt.Sprintf("geocore-%s-monthly", tier)

        // Try to find an existing price by lookup key
        iter := strprice.List(&stripe.PriceListParams{
                LookupKeys: []*string{stripe.String(lookupKey)},
        })
        for iter.Next() {
                return iter.Price().ID, nil
        }
        if err := iter.Err(); err != nil {
                return "", fmt.Errorf("stripe: list prices: %w", err)
        }

        // Create a new monthly price with inline product data
        p, err := strprice.New(&stripe.PriceParams{
                Currency:   stripe.String("usd"),
                UnitAmount: stripe.Int64(int64(math.Round(fee * 100))),
                Recurring: &stripe.PriceRecurringParams{
                        Interval: stripe.String(string(stripe.PriceRecurringIntervalMonth)),
                },
                LookupKey: stripe.String(lookupKey),
                ProductData: &stripe.PriceProductDataParams{
                        Name: stripe.String(productName),
                },
        })
        if err != nil {
                return "", fmt.Errorf("stripe: create price for %s: %w", tier, err)
        }
        return p.ID, nil
}
