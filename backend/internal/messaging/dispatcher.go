package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/push"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ── Message Dispatcher ────────────────────────────────────────────────────────────
//
// Smart message dispatch with anti-spam, cooldowns, quiet hours, and opt-out.
// Triggers: inactivity, outbid, item ending, win, loss.

type Dispatcher struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewDispatcher(db *gorm.DB, rdb *redis.Client) *Dispatcher {
	return &Dispatcher{db: db, rdb: rdb}
}

// DispatchResult is the outcome of a dispatch attempt.
type DispatchResult struct {
	Sent      bool   `json:"sent"`
	MessageID string `json:"message_id"`
	Channel   string `json:"channel"`
	Reason    string `json:"reason"` // why it was (not) sent
}

// Dispatch sends a message to a user with all safety checks.
func (d *Dispatcher) Dispatch(userID uuid.UUID, msgType, channel string, metadata map[string]interface{}) *DispatchResult {
	// ── 1. Load user preferences ──────────────────────────────────────────────
	prefs := d.loadPrefs(userID)

	// ── 2. Opt-out check ────────────────────────────────────────────────────────
	if prefs.OptOutAll {
		return &DispatchResult{Reason: "user_opted_out_all"}
	}
	if channel == ChannelPush && prefs.OptOutPush {
		channel = ChannelInApp // fallback to in-app
	}
	if channel == ChannelEmail && prefs.OptOutEmail {
		channel = ChannelInApp
	}

	// ── 3. Quiet hours check ────────────────────────────────────────────────────
	if isInQuietHours(prefs.QuietHoursStart, prefs.QuietHoursEnd) {
		return &DispatchResult{Reason: "quiet_hours"}
	}

	// ── 4. Per-hour rate limit ──────────────────────────────────────────────────
	hourCount := d.getMessagesSentThisHour(userID)
	maxPerHour := prefs.MaxPerHour
	if maxPerHour == 0 {
		maxPerHour = DefaultMaxPerHour
	}
	if hourCount >= maxPerHour {
		return &DispatchResult{Reason: fmt.Sprintf("hourly_limit_reached (%d/%d)", hourCount, maxPerHour)}
	}

	// ── 5. Per-type cooldown ────────────────────────────────────────────────────
	if !d.checkCooldown(userID, msgType) {
		return &DispatchResult{Reason: fmt.Sprintf("cooldown_active_for_%s", msgType)}
	}

	// ── 6. Build message from template ──────────────────────────────────────────
	tmpl := GetTemplate(msgType, channel)
	title := "Notification"
	body := "You have a new notification."
	if tmpl != nil {
		title = tmpl.Title
		body = tmpl.Body
	}

	// Replace template variables
	if metadata != nil {
		title = replaceVars(title, metadata)
		body = replaceVars(body, metadata)
	}

	metadataJSON, _ := json.Marshal(metadata)

	// ── 7. Create and send message ──────────────────────────────────────────────
	msg := Message{
		UserID:   userID,
		Type:     msgType,
		Title:    title,
		Body:     body,
		Priority: typeToPriority(msgType),
		Channel:  channel,
		Metadata: string(metadataJSON),
		Status:   "pending",
	}
	d.db.Create(&msg)

	// ── 8. Dispatch to channel ──────────────────────────────────────────────────
	now := time.Now()
	switch channel {
	case ChannelPush:
		d.sendPush(userID, &msg)
	case ChannelEmail:
		d.sendEmail(userID, &msg)
	case ChannelInApp:
		d.sendInApp(userID, &msg)
	}

	d.db.Model(&msg).Updates(map[string]interface{}{
		"status":  "sent",
		"sent_at": now,
	})

	// ── 9. Update cooldown ──────────────────────────────────────────────────────
	d.updateCooldown(userID, msgType)

	return &DispatchResult{
		Sent:      true,
		MessageID: msg.ID.String(),
		Channel:   channel,
		Reason:    "sent",
	}
}

// ── Smart Triggers ──────────────────────────────────────────────────────────────────

// TriggerInactive sends a nudge when user is inactive > 10 min.
func (d *Dispatcher) TriggerInactive(userID uuid.UUID) *DispatchResult {
	return d.Dispatch(userID, "nudge", ChannelPush, map[string]interface{}{
		"reason": "inactive_10m",
	})
}

// TriggerOutbid sends an instant push when user is outbid.
func (d *Dispatcher) TriggerOutbid(userID uuid.UUID, itemName string) *DispatchResult {
	return d.Dispatch(userID, "loss", ChannelPush, map[string]interface{}{
		"item_name": itemName,
		"reason":    "outbid",
	})
}

// TriggerItemEnding sends urgency when item is about to end.
func (d *Dispatcher) TriggerItemEnding(userID uuid.UUID, itemName, timeLeft string) *DispatchResult {
	return d.Dispatch(userID, "reminder", ChannelPush, map[string]interface{}{
		"item_name": itemName,
		"time_left": timeLeft,
		"reason":    "ending_soon",
	})
}

// TriggerWin sends a dopamine boost on winning.
func (d *Dispatcher) TriggerWin(userID uuid.UUID, itemName, price string) *DispatchResult {
	return d.Dispatch(userID, "win", ChannelPush, map[string]interface{}{
		"item_name": itemName,
		"price":     price,
		"reason":    "won_auction",
	})
}

// TriggerLoss sends re-engagement after losing.
func (d *Dispatcher) TriggerLoss(userID uuid.UUID, itemName string) *DispatchResult {
	return d.Dispatch(userID, "loss", ChannelPush, map[string]interface{}{
		"item_name": itemName,
		"reason":    "lost_auction",
	})
}

// ── Channel Implementations ─────────────────────────────────────────────────────────

func (d *Dispatcher) sendPush(userID uuid.UUID, msg *Message) {
	// Delegate to the production PushService pipeline:
	// idempotency → rate limit → WS bridge → Firebase FCM → log → Kafka audit
	pushSvc := push.Default()
	pushMsg := &push.PushMessage{
		UserID:           userID,
		NotificationType: msg.Type,
		Priority:         push.ResolvePriority(msg.Type),
		Title:            msg.Title,
		Body:             msg.Body,
	}
	if err := pushSvc.Send(context.Background(), pushMsg); err != nil {
		slog.Warn("dispatcher: push send failed", "user_id", userID, "type", msg.Type, "error", err)
	}
}

func (d *Dispatcher) sendEmail(userID uuid.UUID, msg *Message) {
	// In production: integrate with SendGrid/SES
	// For now: mark as sent (email worker picks up pending emails)
}

func (d *Dispatcher) sendInApp(userID uuid.UUID, msg *Message) {
	// Publish to WebSocket channel
	ctx := context.Background()
	payload, _ := json.Marshal(map[string]interface{}{
		"type":    msg.Type,
		"title":   msg.Title,
		"body":    msg.Body,
		"user_id": userID.String(),
	})
	d.rdb.Publish(ctx, fmt.Sprintf("inapp:%s", userID), payload)
}

// ── Helpers ──────────────────────────────────────────────────────────────────────────

func (d *Dispatcher) loadPrefs(userID uuid.UUID) *UserMessagingPrefs {
	var prefs UserMessagingPrefs
	if err := d.db.Where("user_id = ?", userID).First(&prefs).Error; err != nil {
		prefs = UserMessagingPrefs{
			UserID:          userID,
			MaxPerHour:      DefaultMaxPerHour,
			QuietHoursStart: 22,
			QuietHoursEnd:   8,
		}
		d.db.Create(&prefs)
	}
	return &prefs
}

func (d *Dispatcher) getMessagesSentThisHour(userID uuid.UUID) int {
	var count int64
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	d.db.Model(&Message{}).Where("user_id = ? AND sent_at > ?", userID, oneHourAgo).Count(&count)
	return int(count)
}

func (d *Dispatcher) checkCooldown(userID uuid.UUID, msgType string) bool {
	var cooldown MessageCooldown
	if err := d.db.Where("user_id = ? AND msg_type = ?", userID, msgType).First(&cooldown).Error; err != nil {
		return true // no cooldown = allowed
	}

	duration, ok := CooldownDurations[msgType]
	if !ok {
		duration = 5 * time.Minute // default cooldown
	}

	return time.Since(cooldown.LastSentAt) >= duration
}

func (d *Dispatcher) updateCooldown(userID uuid.UUID, msgType string) {
	var cooldown MessageCooldown
	if err := d.db.Where("user_id = ? AND msg_type = ?", userID, msgType).First(&cooldown).Error; err != nil {
		d.db.Create(&MessageCooldown{
			UserID:     userID,
			MsgType:    msgType,
			LastSentAt: time.Now(),
		})
	} else {
		d.db.Model(&cooldown).Update("last_sent_at", time.Now())
	}
}

func isInQuietHours(start, end int) bool {
	hour := time.Now().Hour()
	if start > end { // crosses midnight
		return hour >= start || hour < end
	}
	return hour >= start && hour < end
}

func typeToPriority(msgType string) string {
	switch msgType {
	case "win", "loss", "outbid":
		return "high"
	case "reminder", "nudge":
		return "normal"
	case "promo":
		return "low"
	default:
		return "normal"
	}
}

func replaceVars(s string, vars map[string]interface{}) string {
	// Simple template replacement
	for k, v := range vars {
		placeholder := fmt.Sprintf("{{%s}}", k)
		s = replaceAll(s, placeholder, fmt.Sprintf("%v", v))
	}
	return s
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); {
		if i <= len(s)-len(old) && s[i:i+len(old)] == old {
			result += new
			i += len(old)
		} else {
			result += string(s[i])
			i++
		}
	}
	return result
}

// ── Messaging Metrics ──────────────────────────────────────────────────────────────

type MessagingMetrics struct {
	TotalSent     int64            `json:"total_sent"`
	DeliveredRate float64          `json:"delivered_rate"`
	OpenRate      float64          `json:"open_rate"`
	ByType        map[string]int64 `json:"by_type"`
	ByChannel     map[string]int64 `json:"by_channel"`
}

func GetMessagingMetrics(db *gorm.DB) *MessagingMetrics {
	var total int64
	db.Model(&Message{}).Where("status = ?", "sent").Count(&total)

	var delivered, opened int64
	db.Model(&Message{}).Where("delivered_at IS NOT NULL").Count(&delivered)
	db.Model(&Message{}).Where("opened_at IS NOT NULL").Count(&opened)

	deliveredRate := 0.0
	openRate := 0.0
	if total > 0 {
		deliveredRate = float64(delivered) / float64(total)
		openRate = float64(opened) / float64(total)
	}

	byType := map[string]int64{}
	var typeResults []struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
	}
	db.Model(&Message{}).Select("type, COUNT(*) as count").Where("status = ?", "sent").
		Group("type").Scan(&typeResults)
	for _, r := range typeResults {
		byType[r.Type] = r.Count
	}

	byChannel := map[string]int64{}
	var chanResults []struct {
		Channel string `json:"channel"`
		Count   int64  `json:"count"`
	}
	db.Model(&Message{}).Select("channel, COUNT(*) as count").Where("status = ?", "sent").
		Group("channel").Scan(&chanResults)
	for _, r := range chanResults {
		byChannel[r.Channel] = r.Count
	}

	return &MessagingMetrics{
		TotalSent:     total,
		DeliveredRate: deliveredRate,
		OpenRate:      openRate,
		ByType:        byType,
		ByChannel:     byChannel,
	}
}
