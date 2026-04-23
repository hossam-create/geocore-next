package push

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/geocore-next/backend/pkg/circuit"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

const tracerName = "push-service"

// PushService is the production-grade push notification orchestrator.
// Pipeline: idempotency → rate-limit → WS-bridge → Firebase → log → Kafka audit.
type PushService struct {
	db     *gorm.DB
	rdb    *redis.Client
	fcm    FirebaseSender
	hub    WSBridge
	tracer trace.Tracer
}

// FirebaseSender abstracts the FCM delivery backend for testability.
type FirebaseSender interface {
	Send(ctx context.Context, token, title, body string, data map[string]string, priority string) (*FCMResult, error)
}

// FCMResult holds the response from a Firebase send call.
type FCMResult struct {
	MessageID string
	Error     string // non-empty if FCM returned an error for this specific token
}

// WSBridge checks if a user is online and delivers via WebSocket.
type WSBridge interface {
	IsOnline(userID string) bool
	SendWS(userID string, msg any) error
}

// NewPushService creates the service. fcm and hub may be nil (graceful no-op).
func NewPushService(db *gorm.DB, rdb *redis.Client, fcm FirebaseSender, hub WSBridge) *PushService {
	return &PushService{
		db:     db,
		rdb:    rdb,
		fcm:    fcm,
		hub:    hub,
		tracer: otel.GetTracerProvider().Tracer(tracerName),
	}
}

// ════════════════════════════════════════════════════════════════════════════
// Public API
// ════════════════════════════════════════════════════════════════════════════

// Send dispatches a push notification to all active devices for a user.
// Non-blocking — enqueues to the async worker if running, else sends synchronously.
func (s *PushService) Send(ctx context.Context, msg *PushMessage) error {
	if msg.Priority == "" {
		msg.Priority = ResolvePriority(msg.NotificationType)
	}
	if msg.IdempotencyKey == "" {
		msg.IdempotencyKey = fmt.Sprintf("push:%s:%s:%d", msg.UserID.String(), msg.NotificationType, time.Now().UnixMilli())
	}

	// ── Idempotency check ────────────────────────────────────────────────────
	if s.rdb != nil {
		set, err := s.rdb.SetNX(ctx, "push:idem:"+msg.IdempotencyKey, "1", 24*time.Hour).Result()
		if err == nil && !set {
			slog.Debug("push: duplicate suppressed", "key", msg.IdempotencyKey)
			return nil
		}
	}

	// ── Rate limit check ─────────────────────────────────────────────────────
	if s.rdb != nil {
		if !checkPushRateLimit(ctx, s.rdb, msg.UserID.String(), msg.NotificationType) {
			slog.Warn("push: rate limited", "user_id", msg.UserID, "type", msg.NotificationType)
			return fmt.Errorf("push rate limit exceeded for user %s type %s", msg.UserID, msg.NotificationType)
		}
	}

	// ── Try async enqueue ────────────────────────────────────────────────────
	if pushWorker != nil {
		pushWorker.enqueue(msg)
		return nil
	}

	// Fallback: synchronous delivery
	return s.deliver(ctx, msg)
}

// SendBatch sends a push to multiple users (fan-out).
func (s *PushService) SendBatch(ctx context.Context, userIDs []uuid.UUID, notificationType, title, body string, data map[string]string) error {
	for _, uid := range userIDs {
		msg := &PushMessage{
			UserID:           uid,
			NotificationType: notificationType,
			Priority:         ResolvePriority(notificationType),
			Title:            title,
			Body:             body,
			Data:             data,
		}
		if err := s.Send(ctx, msg); err != nil {
			slog.Warn("push: batch send failed for user", "user_id", uid, "error", err)
		}
	}
	return nil
}

// ════════════════════════════════════════════════════════════════════════════
// Delivery pipeline (called by worker or sync fallback)
// ════════════════════════════════════════════════════════════════════════════

// deliver executes the full delivery pipeline for a single PushMessage.
func (s *PushService) deliver(ctx context.Context, msg *PushMessage) error {
	ctx, span := s.tracer.Start(ctx, "push.deliver",
		trace.WithAttributes(
			attribute.String("push.user_id", msg.UserID.String()),
			attribute.String("push.type", msg.NotificationType),
			attribute.String("push.priority", msg.Priority),
		),
	)
	defer span.End()

	// ── 1. WebSocket bridge — if user is online, deliver via WS first ────────
	if s.hub != nil && s.hub.IsOnline(msg.UserID.String()) {
		wsPayload := map[string]any{
			"type":  msg.NotificationType,
			"title": msg.Title,
			"body":  msg.Body,
			"data":  msg.Data,
		}
		if err := s.hub.SendWS(msg.UserID.String(), wsPayload); err == nil {
			slog.Debug("push: delivered via WS", "user_id", msg.UserID, "type", msg.NotificationType)
		}
		// Still send push — user may have other devices not connected via WS
	}

	// ── 2. Load active devices ───────────────────────────────────────────────
	var devices []UserDevice
	if s.db != nil {
		s.db.Where("user_id = ? AND is_active = true", msg.UserID).Find(&devices)
	}
	if len(devices) == 0 {
		slog.Debug("push: no active devices", "user_id", msg.UserID)
		return nil
	}

	// ── 3. Firebase delivery with retry ──────────────────────────────────────
	var lastErr error
	for _, dev := range devices {
		if err := s.sendWithRetry(ctx, msg, dev); err != nil {
			lastErr = err
		}
	}

	// ── 4. Kafka audit event ─────────────────────────────────────────────────
	s.publishAuditEvent(msg, lastErr)

	if lastErr != nil {
		return lastErr
	}
	return nil
}

// sendWithRetry sends to a single device with exponential backoff.
func (s *PushService) sendWithRetry(ctx context.Context, msg *PushMessage, dev UserDevice) error {
	const maxAttempts = 3
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var fcmResult *FCMResult

		err := circuit.EmailBreaker.Execute(func(ctx context.Context) error {
			if s.fcm == nil {
				slog.Debug("push: FCM not configured, logging only",
					"user_id", msg.UserID, "token_prefix", safeTokenPrefix(dev.DeviceToken))
				return nil
			}
			var sendErr error
			fcmResult, sendErr = s.fcm.Send(ctx, dev.DeviceToken, msg.Title, msg.Body, msg.Data, msg.Priority)
			return sendErr
		})

		if err != nil {
			lastErr = err
			slog.Warn("push: attempt failed",
				"attempt", attempt,
				"user_id", msg.UserID,
				"device_id", dev.ID,
				"error", err,
			)
			if attempt < maxAttempts {
				backoff := time.Duration(attempt*attempt) * time.Second
				jitter := time.Duration(rand.Intn(500)) * time.Millisecond
				time.Sleep(backoff + jitter)
				continue
			}
		}

		// ── Handle FCM errors (token-level) ──────────────────────────────────
		if fcmResult != nil && fcmResult.Error != "" {
			s.handleFCMTokenError(dev, fcmResult.Error)
		}

		// ── Log delivery ─────────────────────────────────────────────────────
		status := PushStatusSent
		errReason := ""
		if err != nil {
			status = PushStatusFailed
			errReason = lastErr.Error()
		} else if fcmResult != nil && fcmResult.Error != "" {
			status = PushStatusBounced
			errReason = fcmResult.Error
		}
		s.logDelivery(msg, dev, status, fcmResult, errReason, attempt)

		if err == nil {
			return nil
		}
	}

	return lastErr
}

// ════════════════════════════════════════════════════════════════════════════
// Token lifecycle management
// ════════════════════════════════════════════════════════════════════════════

// handleFCMTokenError processes FCM error responses and invalidates bad tokens.
func (s *PushService) handleFCMTokenError(dev UserDevice, fcmErr string) {
	switch {
	case contains(fcmErr, "NotRegistered"),
		contains(fcmErr, "InvalidRegistration"),
		contains(fcmErr, "invalid-argument"):
		slog.Info("push: invalidating device token",
			"device_id", dev.ID, "platform", dev.Platform, "reason", fcmErr)
		if s.db != nil {
			s.db.Model(&UserDevice{}).Where("id = ?", dev.ID).
				Updates(map[string]any{"is_active": false})
		}

	case contains(fcmErr, "TooManyMessages"):
		slog.Warn("push: rate limited by FCM", "device_id", dev.ID)

	case contains(fcmErr, "internal-error"):
		slog.Warn("push: FCM internal error (will retry)", "device_id", dev.ID)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// Observability
// ════════════════════════════════════════════════════════════════════════════

// logDelivery writes a PushLog row for observability and debugging.
func (s *PushService) logDelivery(msg *PushMessage, dev UserDevice, status string, fcmResult *FCMResult, errReason string, attempts int) {
	if s.db == nil {
		return
	}
	dataJSON := "{}"
	if msg.Data != nil {
		if b, err := json.Marshal(msg.Data); err == nil {
			dataJSON = string(b)
		}
	}
	providerMsgID := ""
	if fcmResult != nil {
		providerMsgID = fcmResult.MessageID
	}

	log := PushLog{
		UserID:           msg.UserID,
		DeviceToken:      dev.DeviceToken,
		Platform:         dev.Platform,
		NotificationType: msg.NotificationType,
		Priority:         msg.Priority,
		Title:            msg.Title,
		Body:             msg.Body,
		Data:             dataJSON,
		Status:           status,
		ProviderMsgID:    providerMsgID,
		ErrorReason:      errReason,
		Attempts:         attempts,
		IdempotencyKey:   msg.IdempotencyKey,
	}
	if err := s.db.Create(&log).Error; err != nil {
		slog.Error("push: failed to write delivery log", "error", err)
	}
}

// publishAuditEvent fires a Kafka event for the push delivery.
func (s *PushService) publishAuditEvent(msg *PushMessage, err error) {
	if msg == nil {
		return
	}
	status := "sent"
	if err != nil {
		status = "failed"
	}
	evt := kafka.New(
		"push."+status,
		msg.UserID.String(),
		"push",
		kafka.Actor{Type: "system", ID: "push-service"},
		map[string]any{
			"notification_type": msg.NotificationType,
			"priority":          msg.Priority,
			"title":             msg.Title,
		},
		kafka.EventMeta{Source: "push-service"},
	)
	kafka.PublishAsync(kafka.TopicNotifications, evt)
}

// ════════════════════════════════════════════════════════════════════════════
// Device management
// ════════════════════════════════════════════════════════════════════════════

// RegisterDevice upserts a device token for a user.
func (s *PushService) RegisterDevice(ctx context.Context, userID uuid.UUID, token, platform, appVersion string) (*UserDevice, error) {
	if s.db == nil {
		return nil, fmt.Errorf("push: DB not configured")
	}
	dev := UserDevice{
		UserID:      userID,
		DeviceToken: token,
		Platform:    platform,
		AppVersion:  appVersion,
		IsActive:    true,
		LastSeenAt:  time.Now(),
	}
	result := s.db.Where("device_token = ?", token).
		Assign(map[string]any{
			"user_id":      userID,
			"platform":     platform,
			"app_version":  appVersion,
			"is_active":    true,
			"last_seen_at": time.Now(),
		}).FirstOrCreate(&dev)
	if result.Error != nil {
		return nil, result.Error
	}
	return &dev, nil
}

// UnregisterDevice soft-deletes a device token.
func (s *PushService) UnregisterDevice(ctx context.Context, deviceID uuid.UUID) error {
	if s.db == nil {
		return fmt.Errorf("push: DB not configured")
	}
	return s.db.Model(&UserDevice{}).Where("id = ?", deviceID).
		Update("is_active", false).Error
}

// CleanupStaleDevices marks devices as inactive if not seen in 90 days.
func (s *PushService) CleanupStaleDevices(ctx context.Context) (int64, error) {
	if s.db == nil {
		return 0, nil
	}
	cutoff := time.Now().Add(-90 * 24 * time.Hour)
	result := s.db.Model(&UserDevice{}).
		Where("is_active = true AND last_seen_at < ?", cutoff).
		Update("is_active", false)
	return result.RowsAffected, result.Error
}

// ════════════════════════════════════════════════════════════════════════════
// Helpers
// ════════════════════════════════════════════════════════════════════════════

func ResolvePriority(notificationType string) string {
	if p, ok := NotificationTypePriority[notificationType]; ok {
		return p
	}
	return PriorityMedium
}

func safeTokenPrefix(token string) string {
	if len(token) > 8 {
		return token[:8] + "..."
	}
	return "****"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsInner(s, substr))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ════════════════════════════════════════════════════════════════════════════
// Convenience functions — domain-specific push notifications
//
// These are the primary API for other packages to send push notifications.
// All are async (non-blocking) and go through the full pipeline:
// idempotency → rate limit → WS bridge → FCM → log → Kafka audit.
// ════════════════════════════════════════════════════════════════════════════

// SendPushAuctionBid notifies a seller that a new bid was placed.
func SendPushAuctionBid(userID uuid.UUID, auctionTitle string, bidAmount float64, currency string) error {
	return Default().Send(context.Background(), &PushMessage{
		UserID:           userID,
		NotificationType: "new_bid",
		Title:            "New Bid Received",
		Body:             fmt.Sprintf("A new bid of %.2f %s was placed on %s", bidAmount, currency, auctionTitle),
		Data:             map[string]string{"auction_title": auctionTitle, "bid_amount": fmt.Sprintf("%.2f", bidAmount), "currency": currency},
	})
}

// SendPushOutbid notifies a user they were outbid.
func SendPushOutbid(userID uuid.UUID, auctionTitle string, newBid float64, currency string) error {
	return Default().Send(context.Background(), &PushMessage{
		UserID:           userID,
		NotificationType: "outbid",
		Title:            "You've Been Outbid!",
		Body:             fmt.Sprintf("Someone bid %.2f %s on %s. Bid again to stay in the lead!", newBid, currency, auctionTitle),
		Data:             map[string]string{"auction_title": auctionTitle, "new_bid": fmt.Sprintf("%.2f", newBid), "currency": currency},
	})
}

// SendPushAuctionWon notifies the winner of an auction.
func SendPushAuctionWon(userID uuid.UUID, auctionTitle string, winningBid float64, currency string) error {
	return Default().Send(context.Background(), &PushMessage{
		UserID:           userID,
		NotificationType: "auction_won",
		Title:            "You Won the Auction!",
		Body:             fmt.Sprintf("Congratulations! You won %s for %.2f %s", auctionTitle, winningBid, currency),
		Data:             map[string]string{"auction_title": auctionTitle, "winning_bid": fmt.Sprintf("%.2f", winningBid), "currency": currency},
	})
}

// SendPushAuctionEnded notifies a seller their auction ended.
func SendPushAuctionEnded(userID uuid.UUID, auctionTitle string, hasWinner bool) error {
	body := fmt.Sprintf("Your auction for %s ended without a winner.", auctionTitle)
	if hasWinner {
		body = fmt.Sprintf("Your auction for %s ended with a winner!", auctionTitle)
	}
	return Default().Send(context.Background(), &PushMessage{
		UserID:           userID,
		NotificationType: "auction_ended",
		Title:            "Auction Ended",
		Body:             body,
		Data:             map[string]string{"auction_title": auctionTitle, "has_winner": fmt.Sprintf("%v", hasWinner)},
	})
}

// SendPushNewMessage notifies a user of a new chat message.
func SendPushNewMessage(userID uuid.UUID, senderName, messagePreview string) error {
	if len(messagePreview) > 100 {
		messagePreview = messagePreview[:97] + "..."
	}
	return Default().Send(context.Background(), &PushMessage{
		UserID:           userID,
		NotificationType: "new_message",
		Title:            senderName,
		Body:             messagePreview,
		Data:             map[string]string{"sender_name": senderName},
	})
}

// SendPushOfferCreated notifies a seller of a new offer.
func SendPushOfferCreated(userID uuid.UUID, itemTitle string, offerAmount float64, currency string) error {
	return Default().Send(context.Background(), &PushMessage{
		UserID:           userID,
		NotificationType: "offer_created",
		Title:            "New Offer Received",
		Body:             fmt.Sprintf("You received an offer of %.2f %s for %s", offerAmount, currency, itemTitle),
		Data:             map[string]string{"item_title": itemTitle, "offer_amount": fmt.Sprintf("%.2f", offerAmount), "currency": currency},
	})
}

// SendPushOfferCountered notifies a buyer their offer was countered.
func SendPushOfferCountered(userID uuid.UUID, itemTitle string, counterAmount float64, currency string) error {
	return Default().Send(context.Background(), &PushMessage{
		UserID:           userID,
		NotificationType: "offer_countered",
		Title:            "Offer Countered",
		Body:             fmt.Sprintf("The seller countered your offer for %s with %.2f %s", itemTitle, counterAmount, currency),
		Data:             map[string]string{"item_title": itemTitle, "counter_amount": fmt.Sprintf("%.2f", counterAmount), "currency": currency},
	})
}

// SendPushOfferAccepted notifies a buyer their offer was accepted.
func SendPushOfferAccepted(userID uuid.UUID, itemTitle string) error {
	return Default().Send(context.Background(), &PushMessage{
		UserID:           userID,
		NotificationType: "offer_accepted",
		Title:            "Offer Accepted!",
		Body:             fmt.Sprintf("Your offer for %s has been accepted. Complete your purchase now!", itemTitle),
		Data:             map[string]string{"item_title": itemTitle},
	})
}

// SendPushOfferRejected notifies a buyer their offer was rejected.
func SendPushOfferRejected(userID uuid.UUID, itemTitle string) error {
	return Default().Send(context.Background(), &PushMessage{
		UserID:           userID,
		NotificationType: "offer_rejected",
		Title:            "Offer Declined",
		Body:             fmt.Sprintf("Your offer for %s was declined by the seller.", itemTitle),
		Data:             map[string]string{"item_title": itemTitle},
	})
}

// SendPushPaymentSuccess notifies a user of a successful payment.
func SendPushPaymentSuccess(userID uuid.UUID, orderID string, amount float64, currency string) error {
	return Default().Send(context.Background(), &PushMessage{
		UserID:           userID,
		NotificationType: "payment_success",
		Title:            "Payment Confirmed",
		Body:             fmt.Sprintf("Your payment of %.2f %s for order #%s has been confirmed.", amount, currency, orderID),
		Data:             map[string]string{"order_id": orderID, "amount": fmt.Sprintf("%.2f", amount), "currency": currency},
	})
}

// SendPushPaymentFailed notifies a user of a failed payment.
func SendPushPaymentFailed(userID uuid.UUID, orderID string, reason string) error {
	return Default().Send(context.Background(), &PushMessage{
		UserID:           userID,
		NotificationType: "payment_failed",
		Title:            "Payment Failed",
		Body:             fmt.Sprintf("Payment for order #%s failed. Reason: %s", orderID, reason),
		Data:             map[string]string{"order_id": orderID, "reason": reason},
	})
}

// SendPushEscrowReleased notifies a user that their escrow was released.
func SendPushEscrowReleased(userID uuid.UUID, escrowID string, amount float64, currency string) error {
	return Default().Send(context.Background(), &PushMessage{
		UserID:           userID,
		NotificationType: "escrow_released",
		Title:            "Funds Released",
		Body:             fmt.Sprintf("Your escrow of %.2f %s has been released to your wallet.", amount, currency),
		Data:             map[string]string{"escrow_id": escrowID, "amount": fmt.Sprintf("%.2f", amount), "currency": currency},
	})
}

// SendPushSystem sends a generic system notification.
func SendPushSystem(userID uuid.UUID, title, body string) error {
	return Default().Send(context.Background(), &PushMessage{
		UserID:           userID,
		NotificationType: "announcement",
		Title:            title,
		Body:             body,
	})
}

// SendPushOTP sends a high-priority OTP push notification.
func SendPushOTP(userID uuid.UUID, otp string, expiresMin int) error {
	return Default().Send(context.Background(), &PushMessage{
		UserID:           userID,
		NotificationType: "otp",
		Title:            "Verification Code",
		Body:             fmt.Sprintf("Your verification code is %s. It expires in %d minutes.", otp, expiresMin),
		Data:             map[string]string{"otp": otp, "expires_min": fmt.Sprintf("%d", expiresMin)},
		Silent:           false,
	})
}
