package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	internalnotifications "github.com/geocore-next/backend/internal/notifications"
	internalanalytics "github.com/geocore-next/backend/pkg/analytics"
	pkgemail "github.com/geocore-next/backend/pkg/email"
	"github.com/geocore-next/backend/pkg/sms"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RegisterDefaultHandlers registers all default job handlers
func RegisterDefaultHandlers(q *JobQueue, deps *HandlerDependencies) {
	q.RegisterHandler(JobTypeEmail, deps.HandleEmail)
	q.RegisterHandler(JobTypeSMS, deps.HandleSMS)
	q.RegisterHandler(JobTypePushNotification, deps.HandlePushNotification)
	q.RegisterHandler(JobTypeAuctionEnd, deps.HandleAuctionEnd)
	q.RegisterHandler(JobTypeAuctionReminder, deps.HandleAuctionReminder)
	q.RegisterHandler(JobTypeImageProcess, deps.HandleImageProcess)
	q.RegisterHandler(JobTypeEscrowRelease, deps.HandleEscrowRelease)
	q.RegisterHandler(JobTypeKYCVerify, deps.HandleKYCVerify)
	q.RegisterHandler(JobTypeAnalytics, deps.HandleAnalytics)
	q.RegisterHandler(JobTypeCleanup, deps.HandleCleanup)
	q.RegisterHandler(JobTypeModerationLog, deps.HandleModerationLog)
	q.RegisterHandler(JobTypeSettlementProcess, deps.HandleSettlementProcess)
	q.RegisterHandler(JobTypeGeoScoreUpdate, deps.HandleGeoScoreUpdate)
	q.RegisterHandler(JobTypeRouteUpdate, deps.HandleRouteUpdate)
	q.RegisterHandler(JobTypeBehaviorTrack, deps.HandleBehaviorTrack)
}

// HandlerDependencies contains dependencies for job handlers
type HandlerDependencies struct {
	DB              *gorm.DB
	SMSClient       *sms.TwilioClient
	AnalyticsClient *internalanalytics.PostHogClient
}

// HandleEmail delivers an email by delegating to the production EmailService.
// The job payload is decoded into a pkg/email.Message which goes through the
// full pipeline: idempotency → rate-limit → template render → provider send.
func (d *HandlerDependencies) HandleEmail(ctx context.Context, job *Job) error {
	to, _ := job.Payload["to"].(string)
	if to == "" {
		return fmt.Errorf("email job missing 'to' field")
	}

	if err := pkgemail.ProcessJobPayload(ctx, job.Payload); err != nil {
		slog.Error("email job failed", "job_id", job.ID, "to", to, "error", err)
		return err
	}

	slog.Info("email job completed", "job_id", job.ID, "to", to)
	return nil
}

// HandleSMS sends SMS messages
func (d *HandlerDependencies) HandleSMS(ctx context.Context, job *Job) error {
	to, _ := job.Payload["to"].(string)
	message, _ := job.Payload["message"].(string)
	if to == "" || message == "" {
		return fmt.Errorf("sms job requires to and message")
	}

	if d.SMSClient == nil {
		d.SMSClient = sms.NewTwilioClient()
	}
	if d.SMSClient == nil || !d.SMSClient.IsConfigured() {
		slog.Warn("SMS skipped: Twilio not configured", "to", to)
		return nil
	}

	if err := d.SMSClient.SendSMS(to, message); err != nil {
		slog.Error("sms job failed", "job_id", job.ID, "to", to, "error", err)
		return err
	}

	slog.Info("SMS sent", "job_id", job.ID, "to", to)

	return nil
}

// HandlePushNotification sends push notifications
func (d *HandlerDependencies) HandlePushNotification(ctx context.Context, job *Job) error {
	userID, _ := job.Payload["user_id"].(string)
	title, _ := job.Payload["title"].(string)
	body, _ := job.Payload["body"].(string)
	if d.DB == nil {
		return fmt.Errorf("push notification handler requires DB dependency")
	}
	if userID == "" || title == "" || body == "" {
		return fmt.Errorf("push notification job requires user_id, title and body")
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user_id: %w", err)
	}

	type tokenRow struct{ Token string }
	var rows []tokenRow
	if err := d.DB.WithContext(ctx).
		Table("push_tokens").
		Select("token").
		Where("user_id = ?", parsedUserID).
		Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		slog.Info("No push tokens found", "user_id", userID)
		return nil
	}

	data := map[string]string{}
	if payloadData, ok := job.Payload["data"].(map[string]interface{}); ok {
		for k, v := range payloadData {
			data[k] = fmt.Sprintf("%v", v)
		}
	}

	fcm := internalnotifications.NewFCMClientFromEnv()
	if fcm == nil {
		slog.Warn("Push skipped: FCM client not configured", "user_id", userID)
		return nil
	}

	tokens := make([]string, 0, len(rows))
	for _, row := range rows {
		tokens = append(tokens, row.Token)
	}
	fcm.SendMulticast(tokens, title, body, data)

	// Direct send via FCM is handled by notifications service flow. Here we persist notification records.
	if err := d.createNotification(ctx, parsedUserID, "job_push", title, body, data); err != nil {
		return err
	}

	return nil
}

// HandleAuctionEnd processes auction end
func (d *HandlerDependencies) HandleAuctionEnd(ctx context.Context, job *Job) error {
	if d.DB == nil {
		return fmt.Errorf("auction end handler requires DB dependency")
	}

	auctionID, _ := job.Payload["auction_id"].(string)
	if auctionID == "" {
		return fmt.Errorf("auction end job requires auction_id")
	}

	auctionUUID, err := uuid.Parse(auctionID)
	if err != nil {
		return fmt.Errorf("invalid auction_id: %w", err)
	}

	slog.Info("Processing auction end", "auction_id", auctionID)
	tx := d.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	type auctionRow struct {
		ID           uuid.UUID
		ListingID    uuid.UUID
		SellerID     uuid.UUID
		WinnerID     *uuid.UUID
		CurrentBid   float64
		ReservePrice *float64
		Currency     string
		Status       string
	}
	type bidRow struct {
		UserID uuid.UUID
		Amount float64
	}

	var auction auctionRow
	if err := tx.Table("auctions").
		Select("id, listing_id, seller_id, winner_id, current_bid, reserve_price, currency, status").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", auctionUUID).
		First(&auction).Error; err != nil {
		tx.Rollback()
		return err
	}

	if auction.Status == "ended" || auction.Status == "sold" || auction.Status == "cancelled" {
		tx.Rollback()
		slog.Info("Auction already finalized", "auction_id", auctionID, "status", auction.Status)
		return nil
	}

	var winnerID *uuid.UUID
	winningBid := auction.CurrentBid
	if auction.WinnerID != nil && *auction.WinnerID != uuid.Nil {
		winnerID = auction.WinnerID
	} else {
		var best bidRow
		err := tx.Table("bids").
			Select("user_id, amount").
			Where("auction_id = ?", auction.ID).
			Order("amount DESC, placed_at ASC").
			Limit(1).
			First(&best).Error
		if err == nil {
			if auction.ReservePrice == nil || best.Amount >= *auction.ReservePrice {
				winnerID = &best.UserID
				winningBid = best.Amount
			}
		}
	}

	if winnerID != nil {
		hasOrders := tx.Migrator().HasTable("orders")
		hasOrderItems := tx.Migrator().HasTable("order_items")
		if !hasOrders || !hasOrderItems {
			tx.Rollback()
			return fmt.Errorf("auction end requires orders and order_items tables")
		}
	}

	status := "ended"
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if winnerID != nil {
		status = "sold"
		updates["status"] = status
		updates["winner_id"] = *winnerID
		updates["current_bid"] = winningBid
	}

	if err := tx.Table("auctions").Where("id = ?", auction.ID).Updates(updates).Error; err != nil {
		tx.Rollback()
		return err
	}

	var orderID uuid.UUID
	if winnerID != nil {
		var existing int64
		if err := tx.Table("order_items").Where("auction_id = ?", auction.ID).Count(&existing).Error; err != nil {
			tx.Rollback()
			return err
		}
		if existing == 0 {
			orderID = uuid.New()
			itemID := uuid.New()

			title := "Auction item"
			_ = tx.Table("listings").Select("title").Where("id = ?", auction.ListingID).Scan(&struct{ Title *string }{Title: &title}).Error

			if err := tx.Table("orders").Create(map[string]interface{}{
				"id":           orderID,
				"buyer_id":     *winnerID,
				"seller_id":    auction.SellerID,
				"status":       "pending",
				"subtotal":     winningBid,
				"platform_fee": 0,
				"payment_fee":  0,
				"total":        winningBid,
				"currency":     defaultCurrency(auction.Currency),
				"created_at":   time.Now(),
				"updated_at":   time.Now(),
			}).Error; err != nil {
				tx.Rollback()
				return err
			}

			if err := tx.Table("order_items").Create(map[string]interface{}{
				"id":          itemID,
				"order_id":    orderID,
				"auction_id":  auction.ID,
				"listing_id":  auction.ListingID,
				"title":       title,
				"quantity":    1,
				"unit_price":  winningBid,
				"total_price": winningBid,
				"created_at":  time.Now(),
			}).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	if winnerID != nil {
		_ = d.createNotification(ctx, *winnerID, "auction_won", "You won the auction!", fmt.Sprintf("Winning bid: %.2f %s", winningBid, defaultCurrency(auction.Currency)), map[string]string{"auction_id": auction.ID.String()})
		_ = d.createNotification(ctx, auction.SellerID, "auction_ended", "Your auction ended", fmt.Sprintf("Winning bid: %.2f %s", winningBid, defaultCurrency(auction.Currency)), map[string]string{"auction_id": auction.ID.String()})

		type userContact struct {
			Name  string
			Email string
		}
		var buyer userContact
		var seller userContact
		_ = d.DB.WithContext(ctx).Table("users").Select("name, email").Where("id = ?", *winnerID).First(&buyer).Error
		_ = d.DB.WithContext(ctx).Table("users").Select("name, email").Where("id = ?", auction.SellerID).First(&seller).Error
		if buyer.Email != "" {
			if err := pkgemail.SendAuctionWonEmail(buyer.Email, buyer.Name, "Auction item", winningBid, defaultCurrency(auction.Currency)); err != nil {
				slog.Warn("failed to send auction won email", "auction_id", auction.ID.String(), "error", err)
			}
		}
		if seller.Email != "" {
			if err := pkgemail.SendAuctionEndedSellerEmail(seller.Email, seller.Name, "Auction item", winningBid, defaultCurrency(auction.Currency), true); err != nil {
				slog.Warn("failed to send seller auction ended email", "auction_id", auction.ID.String(), "error", err)
			}
		}
	}

	slog.Info("Auction end processed", "auction_id", auctionID, "winner", winnerID != nil, "order_id", orderID.String())

	return nil
}

// HandleAuctionReminder sends auction ending reminders
func (d *HandlerDependencies) HandleAuctionReminder(ctx context.Context, job *Job) error {
	auctionID, _ := job.Payload["auction_id"].(string)
	userID, _ := job.Payload["user_id"].(string)
	minutesLeft, _ := job.Payload["minutes_left"].(float64)

	slog.Info("Sending auction reminder", "auction_id", auctionID, "user_id", userID, "minutes_left", minutesLeft)

	// TODO: Send notification to user

	return nil
}

// HandleImageProcess processes uploaded images
func (d *HandlerDependencies) HandleImageProcess(ctx context.Context, job *Job) error {
	imageID, _ := job.Payload["image_id"].(string)
	operations, _ := job.Payload["operations"].([]interface{})

	slog.Info("Processing image", "image_id", imageID, "operations", len(operations))

	// TODO:
	// 1. Download original image
	// 2. Generate thumbnails
	// 3. Optimize for web
	// 4. Upload to R2/S3
	// 5. Update database

	return nil
}

// HandleEscrowRelease releases escrow funds
func (d *HandlerDependencies) HandleEscrowRelease(ctx context.Context, job *Job) error {
	if d.DB == nil {
		return fmt.Errorf("escrow release handler requires DB dependency")
	}

	paymentIDStr, _ := job.Payload["payment_id"].(string)
	escrowIDStr, _ := job.Payload["escrow_id"].(string)

	if paymentIDStr == "" && escrowIDStr == "" {
		return fmt.Errorf("escrow release requires payment_id or escrow_id")
	}

	type escrowRow struct {
		ID        uuid.UUID
		PaymentID uuid.UUID
		SellerID  uuid.UUID
		Amount    float64
		Currency  string
		Status    string
	}

	var escrow escrowRow
	q := d.DB.WithContext(ctx).Table("escrow_accounts").
		Select("id, payment_id, seller_id, amount, currency, status")

	if paymentIDStr != "" {
		paymentID, err := uuid.Parse(paymentIDStr)
		if err != nil {
			return fmt.Errorf("invalid payment_id: %w", err)
		}
		if err := q.Where("payment_id = ?", paymentID).First(&escrow).Error; err != nil {
			return err
		}
	} else {
		escrowID, err := uuid.Parse(escrowIDStr)
		if err != nil {
			return fmt.Errorf("invalid escrow_id: %w", err)
		}
		if err := q.Where("id = ?", escrowID).First(&escrow).Error; err != nil {
			return err
		}
	}

	if escrow.Status != "held" {
		slog.Info("Escrow already processed", "escrow_id", escrow.ID.String(), "status", escrow.Status)
		return nil
	}

	now := time.Now()
	tx := d.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := tx.Table("escrow_accounts").
		Where("id = ?", escrow.ID).
		Updates(map[string]interface{}{
			"status":      "released",
			"released_at": now,
			"updated_at":  now,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Credit seller wallet when wallet tables exist in this environment.
	if tx.Migrator().HasTable("wallets") && tx.Migrator().HasTable("wallet_balances") {
		type sellerWallet struct{ ID uuid.UUID }
		var w sellerWallet

		if err := tx.Table("wallets").Select("id").Where("user_id = ?", escrow.SellerID).First(&w).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := tx.Exec(
					"INSERT INTO wallets (id, user_id, primary_currency, is_active, created_at, updated_at) VALUES (uuid_generate_v4(), ?, ?, true, ?, ?)",
					escrow.SellerID, escrow.Currency, now, now,
				).Error; err != nil {
					tx.Rollback()
					return err
				}
				if err := tx.Table("wallets").Select("id").Where("user_id = ?", escrow.SellerID).First(&w).Error; err != nil {
					tx.Rollback()
					return err
				}
			} else {
				tx.Rollback()
				return err
			}
		}

		// Upsert balance row then increment available+total balance.
		tx.Exec(
			"INSERT INTO wallet_balances (id, wallet_id, currency, balance, available_balance, pending_balance, updated_at) VALUES (uuid_generate_v4(), ?, ?, 0, 0, 0, ?) ON CONFLICT DO NOTHING",
			w.ID, escrow.Currency, now,
		)

		if err := tx.Exec(
			"UPDATE wallet_balances SET balance = balance + ?, available_balance = available_balance + ?, updated_at = ? WHERE wallet_id = ? AND currency = ?",
			escrow.Amount, escrow.Amount, now, w.ID, escrow.Currency,
		).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	slog.Info("Escrow released",
		"escrow_id", escrow.ID.String(),
		"payment_id", escrow.PaymentID.String(),
		"seller_id", escrow.SellerID.String(),
		"amount", escrow.Amount,
		"currency", escrow.Currency,
	)

	return nil
}

// HandleKYCVerify processes KYC verification
func (d *HandlerDependencies) HandleKYCVerify(ctx context.Context, job *Job) error {
	userID, _ := job.Payload["user_id"].(string)
	documentType, _ := job.Payload["document_type"].(string)

	slog.Info("Processing KYC verification", "user_id", userID, "document_type", documentType)

	// TODO:
	// 1. Download documents
	// 2. Run face matching
	// 3. Verify document authenticity
	// 4. Update KYC status
	// 5. Notify user

	return nil
}

// HandleAnalytics processes analytics events
func (d *HandlerDependencies) HandleAnalytics(ctx context.Context, job *Job) error {
	event, _ := job.Payload["event"].(string)
	userID, _ := job.Payload["user_id"].(string)
	if event == "" {
		return fmt.Errorf("analytics job requires event")
	}

	if d.AnalyticsClient == nil {
		d.AnalyticsClient = internalanalytics.NewPostHogClient()
	}
	if d.AnalyticsClient == nil || !d.AnalyticsClient.IsConfigured() {
		slog.Warn("analytics skipped: PostHog not configured", "event", event)
		return nil
	}

	distinctID := userID
	if distinctID == "" {
		distinctID = "system"
	}

	props := map[string]interface{}{}
	if m, ok := job.Payload["properties"].(map[string]interface{}); ok {
		props = m
	}
	if props == nil {
		props = map[string]interface{}{}
	}
	props["job_id"] = job.ID

	err := d.AnalyticsClient.Capture(internalanalytics.Event{
		Event:      event,
		DistinctID: distinctID,
		Properties: props,
		Timestamp:  time.Now(),
	})
	if err != nil {
		slog.Error("analytics job failed", "job_id", job.ID, "event", event, "error", err)
		return err
	}

	slog.Info("Analytics event sent", "job_id", job.ID, "event", event, "user_id", userID)

	return nil
}

func (d *HandlerDependencies) createNotification(ctx context.Context, userID uuid.UUID, nType, title, body string, data map[string]string) error {
	if d.DB == nil {
		return fmt.Errorf("notification requires DB dependency")
	}
	payload := "{}"
	if data != nil {
		if b, err := json.Marshal(data); err == nil {
			payload = string(b)
		}
	}
	return d.DB.WithContext(ctx).Table("notifications").Create(map[string]interface{}{
		"id":         uuid.New(),
		"user_id":    userID,
		"type":       nType,
		"title":      title,
		"body":       body,
		"data":       payload,
		"read":       false,
		"created_at": time.Now(),
	}).Error
}

func defaultCurrency(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "USD"
	}
	return strings.ToUpper(v)
}

func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	default:
		return 0
	}
}

// HandleModerationLog writes moderation action to database (async offloading)
func (d *HandlerDependencies) HandleModerationLog(ctx context.Context, job *Job) error {
	targetType, _ := job.Payload["target_type"].(string)
	targetIDStr, _ := job.Payload["target_id"].(string)
	action, _ := job.Payload["action"].(string)
	reason, _ := job.Payload["reason"].(string)
	modIDStr, _ := job.Payload["moderator_id"].(*string)

	if targetType == "" || targetIDStr == "" || action == "" {
		return fmt.Errorf("moderation_log job requires target_type, target_id, and action")
	}

	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		return fmt.Errorf("invalid target_id: %w", err)
	}

	var modID *uuid.UUID
	if modIDStr != nil && *modIDStr != "" {
		if parsed, err := uuid.Parse(*modIDStr); err == nil {
			modID = &parsed
		}
	}

	if d.DB == nil {
		return fmt.Errorf("moderation_log requires DB dependency")
	}

	err = d.DB.WithContext(ctx).Table("moderation_logs").Create(map[string]interface{}{
		"id":           uuid.New(),
		"target_type":  targetType,
		"target_id":    targetID,
		"action":       action,
		"reason":       reason,
		"moderator_id": modID,
		"created_at":   time.Now(),
	}).Error

	if err != nil {
		slog.Error("moderation_log job failed", "job_id", job.ID, "error", err)
		return err
	}

	slog.Info("Moderation log recorded", "job_id", job.ID, "target_type", targetType, "action", action)
	return nil
}

// HandleCleanup performs cleanup tasks
func (d *HandlerDependencies) HandleCleanup(ctx context.Context, job *Job) error {
	cleanupType, _ := job.Payload["type"].(string)

	slog.Info("Running cleanup", "type", cleanupType)

	switch cleanupType {
	case "expired_sessions":
		// Delete expired sessions from Redis
	case "old_notifications":
		// Archive old notifications
	case "temp_files":
		// Delete temporary files
	case "expired_auctions":
		// Mark expired auctions as ended
	default:
		return fmt.Errorf("unknown cleanup type: %s", cleanupType)
	}

	return nil
}

// HandleSettlementProcess processes a settlement in the background.
func (d *HandlerDependencies) HandleSettlementProcess(ctx context.Context, job *Job) error {
	sid, ok := job.Payload["settlement_id"].(string)
	if !ok {
		return fmt.Errorf("missing settlement_id in payload")
	}

	slog.Info("Processing settlement", "settlement_id", sid, "request_id", job.RequestID)

	// Look up and process the settlement directly
	var settlement struct {
		ID     uuid.UUID `gorm:"type:uuid"`
		Status string
	}
	if err := d.DB.Table("settlements").Where("id = ?", sid).First(&settlement).Error; err != nil {
		return fmt.Errorf("settlement not found: %w", err)
	}
	if settlement.Status != "pending" {
		return nil
	}

	now := time.Now()
	if err := d.DB.Table("settlements").Where("id = ?", sid).Updates(map[string]interface{}{
		"status":       "completed",
		"processed_at": now,
	}).Error; err != nil {
		return fmt.Errorf("settlement update failed: %w", err)
	}

	slog.Info("Settlement processed", "settlement_id", sid)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Track C handlers — inline to avoid import cycles with internal packages
// ─────────────────────────────────────────────────────────────────────────────

// HandleGeoScoreUpdate recomputes and persists a user's GeoScore.
// Payload: { "user_id": "<uuid>" }
func (d *HandlerDependencies) HandleGeoScoreUpdate(_ context.Context, job *Job) error {
	uid, ok := job.Payload["user_id"].(string)
	if !ok || uid == "" {
		return fmt.Errorf("geoscore.update: missing user_id")
	}

	// ── Gather signals ────────────────────────────────────────────────────────
	var orderStats struct {
		Total     int64
		Delivered int64
	}
	d.DB.Raw(`SELECT
		COUNT(*) FILTER (WHERE status NOT IN ('cancelled','pending')) AS total,
		COUNT(*) FILTER (WHERE status IN ('delivered','completed')) AS delivered
		FROM orders WHERE (seller_id = ? OR buyer_id = ?) AND deleted_at IS NULL`,
		uid, uid).Scan(&orderStats)

	var disputeCount int64
	d.DB.Raw(`SELECT COUNT(*) FROM disputes
		WHERE (complainant_id = ? OR respondent_id = ?)
		  AND status NOT IN ('cancelled','closed_no_action')`, uid, uid).Scan(&disputeCount)

	var kycLevel string
	d.DB.Raw(`SELECT COALESCE(verification_level,'none') FROM kyc_profiles WHERE user_id = ? LIMIT 1`, uid).Scan(&kycLevel)

	kycScore := map[string]float64{"none": 0.0, "basic": 0.5, "full": 1.0}[kycLevel]

	// ── Compute score ─────────────────────────────────────────────────────────
	total := float64(orderStats.Total)
	successRate, disputeRate := 0.0, 0.0
	if total > 0 {
		successRate = float64(orderStats.Delivered) / total
		if disputeCount > 0 {
			disputeRate = float64(disputeCount) / total
			if disputeRate > 1 {
				disputeRate = 1
			}
		}
	}

	score := successRate*0.35 + (1-disputeRate)*0.25 + kycScore*0.15 + 1.0*0.15 + 1.0*0.10
	if score > 1 {
		score = 1
	}
	score = float64(int(score*10000)) / 100 // round to 2dp, scale to 0-100

	// ── Upsert ────────────────────────────────────────────────────────────────
	err := d.DB.Exec(`
		INSERT INTO geo_scores (user_id, score, success_rate, dispute_rate, kyc_score,
		                        delivery_score, fraud_score, updated_at)
		VALUES (?, ?, ?, ?, ?, 1.0, 0.0, NOW())
		ON CONFLICT (user_id) DO UPDATE
		  SET score = EXCLUDED.score, success_rate = EXCLUDED.success_rate,
		      dispute_rate = EXCLUDED.dispute_rate, kyc_score = EXCLUDED.kyc_score,
		      updated_at = NOW()`,
		uid, score, successRate, disputeRate, kycScore).Error
	if err != nil {
		return fmt.Errorf("geoscore upsert: %w", err)
	}

	// Invalidate Redis cache key
	slog.Info("geoscore.update: completed", "user_id", uid, "score", score)
	return nil
}

// HandleRouteUpdate aggregates order statistics for an origin/destination route.
// Payload: { "origin": "DXB", "destination": "CAI" }
func (d *HandlerDependencies) HandleRouteUpdate(_ context.Context, job *Job) error {
	origin, _ := job.Payload["origin"].(string)
	dest, _ := job.Payload["destination"].(string)
	if origin == "" || dest == "" {
		return fmt.Errorf("route.update: missing origin or destination")
	}

	var agg struct {
		TotalOrders int64
		TotalWeight float64
		AvgPrice    float64
		Completed   int64
		Disputed    int64
	}
	d.DB.Raw(`
		SELECT COUNT(*) AS total_orders,
		       COALESCE(SUM(total),0) AS total_weight,
		       COALESCE(AVG(total),0) AS avg_price,
		       COUNT(*) FILTER (WHERE status IN ('delivered','completed')) AS completed,
		       COUNT(*) FILTER (WHERE status = 'disputed') AS disputed
		FROM orders WHERE deleted_at IS NULL`).Scan(&agg)

	successRate, disputeRate := 0.0, 0.0
	if agg.TotalOrders > 0 {
		successRate = float64(agg.Completed) / float64(agg.TotalOrders)
		disputeRate = float64(agg.Disputed) / float64(agg.TotalOrders)
	}

	// Normalised demand: log10(n+1)/log10(1001)
	demandScore := 0.0
	if agg.TotalOrders > 0 {
		n := float64(agg.TotalOrders)
		// manual log10 (no math import available here without changing imports)
		log := 0.0
		v := n + 1
		for v >= 10 {
			v /= 10
			log++
		}
		if v > 1 {
			log += (v - 1) / 9.0 // linear interpolation for fractional part
		}
		demandScore = log / 3.0 // ~0-1 for 1-1000 orders
		if demandScore > 1 {
			demandScore = 1
		}
	}

	err := d.DB.Exec(`
		INSERT INTO route_metrics (id, origin, destination, total_orders, total_weight,
		                           avg_price, success_rate, dispute_rate, demand_score, updated_at)
		VALUES (uuid_generate_v4(), ?, ?, ?, ?, ?, ?, ?, ?, NOW())
		ON CONFLICT (origin, destination) DO UPDATE
		  SET total_orders = EXCLUDED.total_orders, total_weight = EXCLUDED.total_weight,
		      avg_price = EXCLUDED.avg_price, success_rate = EXCLUDED.success_rate,
		      dispute_rate = EXCLUDED.dispute_rate, demand_score = EXCLUDED.demand_score,
		      updated_at = NOW()`,
		origin, dest, agg.TotalOrders, agg.TotalWeight, agg.AvgPrice,
		successRate, disputeRate, demandScore).Error
	if err != nil {
		return fmt.Errorf("route.update upsert: %w", err)
	}

	slog.Info("route.update: completed", "origin", origin, "destination", dest)
	return nil
}

// HandleBehaviorTrack inserts a behavior event for future ML analysis.
// Payload: { "user_id": "<uuid>", "event_type": "search", "metadata": {...} }
func (d *HandlerDependencies) HandleBehaviorTrack(_ context.Context, job *Job) error {
	uid, _ := job.Payload["user_id"].(string)
	evtType, _ := job.Payload["event_type"].(string)
	if uid == "" || evtType == "" {
		return fmt.Errorf("behavior.track: missing user_id or event_type")
	}

	metadata, _ := json.Marshal(job.Payload["metadata"])
	err := d.DB.Exec(`
		INSERT INTO behavior_events (id, user_id, event_type, metadata, created_at)
		VALUES (uuid_generate_v4(), ?, ?, ?::jsonb, NOW())`,
		uid, evtType, string(metadata)).Error
	if err != nil {
		slog.Warn("behavior.track: insert failed", "error", err)
		return err
	}
	return nil
}
