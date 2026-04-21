package disputes

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/pkg/events"
	"github.com/geocore-next/backend/pkg/jobs"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/refund"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

type orderRef struct {
	ID              uuid.UUID
	BuyerID         uuid.UUID
	SellerID        uuid.UUID
	Status          string
	Total           float64
	Currency        string
	PaymentIntentID string
}

type paymentRef struct {
	ID                    uuid.UUID
	StripePaymentIntentID string
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// CreateDispute opens a new dispute
func (h *Handler) CreateDispute(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))

	var req struct {
		OrderID  string        `json:"order_id" binding:"required"`
		Reason   DisputeReason `json:"reason" binding:"required"`
		Evidence string        `json:"evidence" binding:"required,min=20"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}

	var ord orderRef
	if err := h.db.Table("orders").
		Select("id, buyer_id, seller_id, status, total, currency, payment_intent_id").
		Where("id = ?", orderID).
		First(&ord).Error; err != nil {
		response.NotFound(c, "Order")
		return
	}

	if ord.BuyerID != userID {
		response.Forbidden(c)
		return
	}
	if ord.Status == "cancelled" {
		response.BadRequest(c, "Cannot dispute a cancelled order")
		return
	}

	var existing int64
	h.db.Model(&Dispute{}).Where("order_id = ? AND status IN ?", ord.ID, []DisputeStatus{StatusOpen, StatusUnderReview, StatusEscalated, StatusAwaitingResponse}).Count(&existing)
	if existing > 0 {
		response.BadRequest(c, "An active dispute already exists for this order")
		return
	}

	dispute := Dispute{
		ID:          uuid.New(),
		BuyerID:     userID,
		SellerID:    ord.SellerID,
		OrderID:     &ord.ID,
		Reason:      req.Reason,
		Description: req.Evidence,
		Amount:      ord.Total,
		Currency:    defaultStr(ord.Currency, "USD"),
		Status:      StatusOpen,
		Priority:    5,
	}

	responseHrs, resolutionHrs := slaHoursForPriority(dispute.Priority)
	responseDeadline := time.Now().Add(time.Duration(responseHrs) * time.Hour)
	resolutionDeadline := time.Now().Add(time.Duration(resolutionHrs) * time.Hour)
	dispute.ResponseDeadline = &responseDeadline
	dispute.ResolutionDeadline = &resolutionDeadline

	if err := h.db.Create(&dispute).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Mark order as disputed while dispute is open.
	h.db.Table("orders").Where("id = ?", ord.ID).Update("status", "disputed")

	// Log activity
	h.logActivity(dispute.ID, userID, "dispute_opened", "Dispute opened by buyer")

	// Publish domain event for in-process consumers
	events.Publish(events.Event{
		Type: events.EventDisputeOpened,
		Payload: map[string]interface{}{
			"dispute_id":    dispute.ID.String(),
			"order_id":      ord.ID.String(),
			"buyer_id":      userID.String(),
			"seller_id":     ord.SellerID.String(),
			"reason":        string(req.Reason),
			"amount":        ord.Total,
			"currency":      ord.Currency,
			"respondent_id": ord.SellerID.String(),
		},
	})

	// Transactional outbox for Kafka delivery
	_ = kafka.WriteOutbox(h.db, kafka.TopicModeration, kafka.New(
		"dispute.opened",
		dispute.ID.String(),
		"dispute",
		kafka.Actor{Type: "user", ID: userID.String()},
		map[string]interface{}{
			"dispute_id":    dispute.ID.String(),
			"order_id":      ord.ID.String(),
			"buyer_id":      userID.String(),
			"seller_id":     ord.SellerID.String(),
			"reason":        string(req.Reason),
			"respondent_id": ord.SellerID.String(),
		},
		kafka.EventMeta{Source: "api-service"},
	))

	response.Created(c, dispute)
}

// GetDispute returns a single dispute
func (h *Handler) GetDispute(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	disputeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid dispute ID")
		return
	}

	var dispute Dispute
	if err := h.db.Preload("Messages").Preload("Evidence").
		First(&dispute, "id = ?", disputeID).Error; err != nil {
		response.NotFound(c, "Dispute")
		return
	}

	// Only buyer, seller, or admin can view
	if dispute.BuyerID != userID && dispute.SellerID != userID {
		// Check if admin
		role, _ := c.Get("user_role")
		if role != "admin" {
			response.Forbidden(c)
			return
		}
	}

	response.OK(c, dispute)
}

// ListDisputes returns disputes for the current user
func (h *Handler) ListDisputes(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage := 20
	status := c.Query("status")
	role := c.Query("role") // buyer or seller

	var disputes []Dispute
	var total int64

	q := h.db.Model(&Dispute{})

	if role == "seller" {
		q = q.Where("seller_id = ?", userID)
	} else if role == "buyer" {
		q = q.Where("buyer_id = ?", userID)
	} else {
		q = q.Where("buyer_id = ? OR seller_id = ?", userID, userID)
	}

	if status != "" {
		q = q.Where("status = ?", status)
	}

	q.Count(&total)
	q.Preload("Messages", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC").Limit(1)
	}).Offset((page - 1) * perPage).Limit(perPage).
		Order("created_at DESC").Find(&disputes)

	response.OKMeta(c, disputes, gin.H{"total": total, "page": page})
}

// AddMessage adds a message to a dispute
func (h *Handler) AddMessage(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	disputeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid dispute ID")
		return
	}

	var req struct {
		Message string `json:"message" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var dispute Dispute
	if err := h.db.First(&dispute, "id = ?", disputeID).Error; err != nil {
		response.NotFound(c, "Dispute")
		return
	}

	// Determine sender role
	var senderRole string
	if dispute.BuyerID == userID {
		senderRole = "buyer"
	} else if dispute.SellerID == userID {
		senderRole = "seller"
	} else {
		role, _ := c.Get("user_role")
		if role == "admin" {
			senderRole = "admin"
		} else {
			response.Forbidden(c)
			return
		}
	}

	message := DisputeMessage{
		ID:         uuid.New(),
		DisputeID:  disputeID,
		SenderID:   userID,
		SenderRole: senderRole,
		Message:    req.Message,
		IsInternal: false,
		CreatedAt:  time.Now(),
	}

	if err := h.db.Create(&message).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Update dispute status if seller responds
	if senderRole == "seller" && dispute.Status == StatusOpen {
		h.db.Model(&dispute).Update("status", StatusUnderReview)
	}

	h.logActivity(disputeID, userID, "message_added", "Message added by "+senderRole)

	response.Created(c, message)
}

// AddEvidence adds evidence to a dispute
func (h *Handler) AddEvidence(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	disputeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid dispute ID")
		return
	}

	var req struct {
		Type        string `json:"type" binding:"required"` // image, document, screenshot, video
		URL         string `json:"url" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var dispute Dispute
	if err := h.db.First(&dispute, "id = ?", disputeID).Error; err != nil {
		response.NotFound(c, "Dispute")
		return
	}

	// Only buyer or seller can add evidence
	if dispute.BuyerID != userID && dispute.SellerID != userID {
		response.Forbidden(c)
		return
	}

	evidence := DisputeEvidence{
		ID:          uuid.New(),
		DisputeID:   disputeID,
		SubmittedBy: userID,
		Type:        req.Type,
		URL:         req.URL,
		Description: req.Description,
		CreatedAt:   time.Now(),
	}

	if err := h.db.Create(&evidence).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	h.logActivity(disputeID, userID, "evidence_added", "Evidence added: "+req.Type)

	response.Created(c, evidence)
}

// ResolveDispute resolves a dispute (admin only)
func (h *Handler) ResolveDispute(c *gin.Context) {
	adminID, _ := uuid.Parse(c.MustGet("user_id").(string))
	disputeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid dispute ID")
		return
	}

	var req struct {
		Outcome string `json:"outcome" binding:"required,oneof=refund_buyer release_seller"`
		Notes   string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var dispute Dispute
	if err := h.db.First(&dispute, "id = ?", disputeID).Error; err != nil {
		response.NotFound(c, "Dispute")
		return
	}

	if dispute.Status == StatusResolved || dispute.Status == StatusClosed {
		response.BadRequest(c, "Dispute already resolved")
		return
	}

	now := time.Now()
	var mappedResolution ResolutionType
	if req.Outcome == "refund_buyer" {
		mappedResolution = ResolutionFullRefund
	} else {
		mappedResolution = ResolutionNoRefund
	}

	updates := map[string]interface{}{
		"status":           StatusResolved,
		"resolution":       mappedResolution,
		"resolution_notes": req.Notes,
		"resolved_by":      adminID,
		"resolved_at":      now,
	}

	if err := h.db.Model(&dispute).Updates(updates).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	if dispute.OrderID == nil {
		response.BadRequest(c, "Resolve flow currently requires order-linked dispute")
		return
	}

	var ord orderRef
	if err := h.db.Table("orders").
		Select("id, buyer_id, seller_id, status, total, currency, payment_intent_id").
		Where("id = ?", *dispute.OrderID).
		First(&ord).Error; err != nil {
		response.NotFound(c, "Order")
		return
	}

	switch req.Outcome {
	case "refund_buyer":
		if err := h.refundBuyerForOrder(c.Request.Context(), ord); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		h.db.Table("orders").Where("id = ?", ord.ID).Update("status", "refunded")

	case "release_seller":
		if err := h.releaseEscrowForOrder(c.Request.Context(), ord); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		h.db.Table("orders").Where("id = ?", ord.ID).Update("status", "completed")
	}

	h.logActivity(disputeID, adminID, "dispute_resolved", req.Outcome)

	response.OK(c, gin.H{
		"message":    "Dispute resolved",
		"resolution": req.Outcome,
	})
}

func (h *Handler) refundBuyerForOrder(ctx context.Context, ord orderRef) error {
	if ord.PaymentIntentID == "" {
		return fmt.Errorf("order has no payment_intent_id")
	}

	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(ord.PaymentIntentID),
		Reason:        stripe.String("requested_by_customer"),
	}
	if _, err := refund.New(params); err != nil {
		return fmt.Errorf("stripe refund failed: %w", err)
	}

	return h.db.Table("payments").
		Where("stripe_payment_intent_id = ?", ord.PaymentIntentID).
		Updates(map[string]interface{}{"status": "refunded", "refunded_at": time.Now()}).Error
}

func (h *Handler) releaseEscrowForOrder(ctx context.Context, ord orderRef) error {
	if ord.PaymentIntentID == "" {
		return fmt.Errorf("order has no payment_intent_id")
	}

	var p paymentRef
	if err := h.db.Table("payments").
		Select("id, stripe_payment_intent_id").
		Where("stripe_payment_intent_id = ?", ord.PaymentIntentID).
		First(&p).Error; err != nil {
		return fmt.Errorf("payment not found for order: %w", err)
	}

	deps := jobs.HandlerDependencies{DB: h.db}
	job := &jobs.Job{
		Type:    jobs.JobTypeEscrowRelease,
		Payload: map[string]interface{}{"payment_id": p.ID.String()},
	}
	return deps.HandleEscrowRelease(ctx, job)
}

// EscalateDispute escalates a dispute (buyer only)
func (h *Handler) EscalateDispute(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	disputeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid dispute ID")
		return
	}

	var dispute Dispute
	if err := h.db.First(&dispute, "id = ?", disputeID).Error; err != nil {
		response.NotFound(c, "Dispute")
		return
	}

	if dispute.BuyerID != userID {
		response.Forbidden(c)
		return
	}

	if dispute.Status == StatusEscalated || dispute.Status == StatusResolved {
		response.BadRequest(c, "Cannot escalate this dispute")
		return
	}

	// Check if response deadline passed
	if dispute.ResponseDeadline != nil && time.Now().Before(*dispute.ResponseDeadline) {
		response.BadRequest(c, "Cannot escalate before response deadline")
		return
	}

	now := time.Now()
	h.db.Model(&dispute).Updates(map[string]interface{}{
		"status":          StatusEscalated,
		"escalation_date": now,
		"priority":        1, // Highest priority
	})

	h.logActivity(disputeID, userID, "dispute_escalated", "Escalated by buyer")

	response.OK(c, gin.H{"message": "Dispute escalated to admin review"})
}

// CloseDispute closes a dispute (buyer or admin)
func (h *Handler) CloseDispute(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	disputeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid dispute ID")
		return
	}

	var dispute Dispute
	if err := h.db.First(&dispute, "id = ?", disputeID).Error; err != nil {
		response.NotFound(c, "Dispute")
		return
	}

	// Only buyer or admin can close
	role, _ := c.Get("user_role")
	if dispute.BuyerID != userID && role != "admin" {
		response.Forbidden(c)
		return
	}

	h.db.Model(&dispute).Update("status", StatusClosed)
	h.logActivity(disputeID, userID, "dispute_closed", "Closed by user")

	response.OK(c, gin.H{"message": "Dispute closed"})
}

// GetActivity returns activity log for a dispute
func (h *Handler) GetActivity(c *gin.Context) {
	disputeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid dispute ID")
		return
	}

	var activities []DisputeActivity
	h.db.Where("dispute_id = ?", disputeID).
		Order("created_at DESC").
		Find(&activities)

	response.OK(c, activities)
}

// AdminListDisputes returns all disputes for admin
func (h *Handler) AdminListDisputes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage := 20
	status := c.Query("status")
	priority := c.Query("priority")

	var disputes []Dispute
	var total int64

	q := h.db.Model(&Dispute{})

	if status != "" {
		q = q.Where("status = ?", status)
	}
	if priority != "" {
		p, _ := strconv.Atoi(priority)
		q = q.Where("priority = ?", p)
	}

	q.Count(&total)
	q.Offset((page - 1) * perPage).Limit(perPage).
		Order("priority ASC, created_at ASC").Find(&disputes)

	response.OKMeta(c, disputes, gin.H{"total": total, "page": page})
}

// AdminAssignDispute assigns a dispute to an admin
func (h *Handler) AdminAssignDispute(c *gin.Context) {
	adminID, _ := uuid.Parse(c.MustGet("user_id").(string))
	disputeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid dispute ID")
		return
	}

	var req struct {
		AssignTo *string `json:"assign_to"` // nil = assign to self
	}
	c.ShouldBindJSON(&req)

	assignTo := adminID
	if req.AssignTo != nil {
		assignTo, _ = uuid.Parse(*req.AssignTo)
	}

	h.db.Model(&Dispute{}).Where("id = ?", disputeID).
		Update("assigned_to", assignTo)

	h.logActivity(disputeID, adminID, "dispute_assigned", "Assigned to admin")

	response.OK(c, gin.H{"message": "Dispute assigned"})
}

func (h *Handler) logActivity(disputeID, actorID uuid.UUID, action, details string) {
	activity := DisputeActivity{
		ID:        uuid.New(),
		DisputeID: disputeID,
		ActorID:   actorID,
		Action:    action,
		Details:   details,
		CreatedAt: time.Now(),
	}
	h.db.Create(&activity)
}

func defaultStr(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func slaHoursForPriority(priority int) (responseHours int, resolutionHours int) {
	switch {
	case priority <= 2:
		return 12, 24
	case priority <= 4:
		return 24, 48
	default:
		return 36, 72
	}
}

func (h *Handler) MarkSLABreaches() (int64, error) {
	now := time.Now()
	var breached []Dispute
	if err := h.db.
		Where("sla_breached = ? AND status <> ? AND resolution_deadline IS NOT NULL AND resolution_deadline < ?", false, StatusResolved, now).
		Find(&breached).Error; err != nil {
		return 0, err
	}

	for _, d := range breached {
		h.db.Model(&Dispute{}).Where("id = ?", d.ID).Update("sla_breached", true)
		metrics.IncDisputesSLABreachedTotal()
		h.logActivity(d.ID, d.BuyerID, "sla_breached", "Resolution SLA breached")
		h.db.Create(&notifications.Notification{
			UserID: d.BuyerID,
			Type:   "dispute_sla_breached",
			Title:  "Dispute SLA Breached",
			Body:   "Your dispute exceeded resolution SLA and was escalated.",
			Data:   fmt.Sprintf(`{"dispute_id":"%s"}`, d.ID.String()),
		})
		slog.Error("dispute SLA breached",
			"severity", "CRITICAL",
			"dispute_id", d.ID.String(),
		)
	}

	return int64(len(breached)), nil
}

func StartSLAWorker(db *gorm.DB, notifSvc any) {
	h := NewHandler(db)
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := markSLABreachesWithRetry(h, 3); err != nil {
				slog.Error("dispute SLA worker failed", "error", err.Error())
			}
			_ = notifSvc
		}
	}()
}

func markSLABreachesWithRetry(h *Handler, maxRetries int) (int64, error) {
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		count, err := h.MarkSLABreaches()
		if err == nil {
			return count, nil
		}
		lastErr = err
		if attempt < maxRetries {
			backoff := time.Duration(1<<(attempt-1)) * time.Second
			time.Sleep(backoff)
		}
	}
	return 0, lastErr
}
