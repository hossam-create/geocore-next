package disputes

import (
	"strconv"
	"time"

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

// CreateDispute opens a new dispute
func (h *Handler) CreateDispute(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))

	var req struct {
		OrderID     *string       `json:"order_id"`
		AuctionID   *string       `json:"auction_id"`
		EscrowID    *string       `json:"escrow_id"`
		SellerID    string        `json:"seller_id" binding:"required"`
		Reason      DisputeReason `json:"reason" binding:"required"`
		Description string        `json:"description" binding:"required,min=20"`
		Amount      float64       `json:"amount" binding:"required,min=0"`
		Currency    string        `json:"currency"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	sellerID, err := uuid.Parse(req.SellerID)
	if err != nil {
		response.BadRequest(c, "Invalid seller ID")
		return
	}

	if sellerID == userID {
		response.BadRequest(c, "Cannot open dispute against yourself")
		return
	}

	dispute := Dispute{
		ID:          uuid.New(),
		BuyerID:     userID,
		SellerID:    sellerID,
		Reason:      req.Reason,
		Description: req.Description,
		Amount:      req.Amount,
		Currency:    defaultStr(req.Currency, "USD"),
		Status:      StatusOpen,
		Priority:    5,
	}

	if req.OrderID != nil {
		id, _ := uuid.Parse(*req.OrderID)
		dispute.OrderID = &id
	}
	if req.AuctionID != nil {
		id, _ := uuid.Parse(*req.AuctionID)
		dispute.AuctionID = &id
	}
	if req.EscrowID != nil {
		id, _ := uuid.Parse(*req.EscrowID)
		dispute.EscrowID = &id
	}

	// Set response deadline (48 hours)
	deadline := time.Now().Add(48 * time.Hour)
	dispute.ResponseDeadline = &deadline

	if err := h.db.Create(&dispute).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Log activity
	h.logActivity(dispute.ID, userID, "dispute_opened", "Dispute opened by buyer")

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
		Resolution       ResolutionType `json:"resolution" binding:"required"`
		ResolutionAmount *float64       `json:"resolution_amount"`
		ResolutionNotes  string         `json:"resolution_notes"`
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
	updates := map[string]interface{}{
		"status":           StatusResolved,
		"resolution":       req.Resolution,
		"resolution_notes": req.ResolutionNotes,
		"resolved_by":      adminID,
		"resolved_at":      now,
	}

	if req.ResolutionAmount != nil {
		updates["resolution_amount"] = *req.ResolutionAmount
	}

	if err := h.db.Model(&dispute).Updates(updates).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	h.logActivity(disputeID, adminID, "dispute_resolved", string(req.Resolution))

	// TODO: Process refund/release based on resolution
	// - If full_refund: Release escrow to buyer
	// - If no_refund: Release escrow to seller
	// - If partial_refund: Split escrow

	response.OK(c, gin.H{
		"message":    "Dispute resolved",
		"resolution": req.Resolution,
	})
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
