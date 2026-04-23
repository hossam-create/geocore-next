package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"github.com/geocore-next/backend/pkg/email"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Notification Providers
// Abstraction over push/email delivery with retry + async dispatch.
// ════════════════════════════════════════════════════════════════════════════

// NotificationProvider defines the interface for external notification delivery.
type NotificationProvider interface {
	SendPush(token string, title string, body string, data map[string]interface{}) error
	SendEmail(email string, subject string, body string) error
}

// ProviderConfig controls which channels are enabled via ENV.
type ProviderConfig struct {
	EnablePush  bool
	EnableEmail bool
}

// LoadProviderConfig reads ENV to determine enabled channels.
func LoadProviderConfig() ProviderConfig {
	return ProviderConfig{
		EnablePush:  os.Getenv("ENABLE_PUSH") != "false",
		EnableEmail: os.Getenv("ENABLE_EMAIL") != "false",
	}
}

// FirebaseProvider wraps FCMClient as a NotificationProvider.
type FirebaseProvider struct {
	client *FCMClient
}

func (p *FirebaseProvider) SendPush(token string, title string, body string, data map[string]interface{}) error {
	if p.client == nil {
		return fmt.Errorf("fcm: client not initialised")
	}
	strData := make(map[string]string)
	for k, v := range data {
		strData[k] = fmt.Sprintf("%v", v)
	}
	return p.client.Send(token, title, body, strData)
}

func (p *FirebaseProvider) SendEmail(email string, subject string, body string) error {
	return nil
}

// EmailProvider routes email notifications through the production EmailService.
type EmailProvider struct {
	db *gorm.DB
}

func (p *EmailProvider) SendPush(userID string, title string, body string, data map[string]interface{}) error {
	return nil
}

func (p *EmailProvider) SendEmail(addr string, subject string, body string) error {
	msg := &email.Message{
		To:             addr,
		Subject:        subject,
		TemplateName:   "notification",
		Data:           email.NotificationData("there", subject, body, "", ""),
		IdempotencyKey: "notify:" + addr + ":" + subject,
		CreatedAt:      time.Now(),
	}
	if err := email.Default().SendAsync(context.Background(), msg); err != nil {
		slog.Error("notify: email enqueue failed", "to", addr, "error", err)
		return err
	}
	slog.Info("notify: email enqueued", "to", addr, "subject", subject)
	return nil
}

// ════════════════════════════════════════════════════════════════════════════
// Retry + Async Dispatch
// ════════════════════════════════════════════════════════════════════════════

const maxRetries = 3
const retryDelay = 500 * time.Millisecond

// retrySend attempts to call fn up to maxRetries times with exponential backoff.
// Returns the last error if all retries failed, nil on success.
func retrySend(fn func() error, label string, logCtx map[string]interface{}) error {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := fn(); err != nil {
			slog.Warn("notify: delivery attempt failed",
				"provider", label,
				"attempt", attempt,
				"error", err.Error(),
				"context", logCtx,
			)
			if attempt < maxRetries {
				time.Sleep(retryDelay * time.Duration(attempt))
				continue
			}
			slog.Error("notify: delivery failed after retries",
				"provider", label,
				"attempts", maxRetries,
				"error", err.Error(),
				"context", logCtx,
			)
			return err
		}
		slog.Info("notify: delivery succeeded",
			"provider", label,
			"attempt", attempt,
			"context", logCtx,
		)
		return nil
	}
	return nil
}

// SafeGo runs a function in a goroutine with panic recovery.
func SafeGo(fn func(), label string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("notify: goroutine panic recovered",
					"label", label,
					"panic", fmt.Sprintf("%v", r),
				)
			}
		}()
		fn()
	}()
}

// ════════════════════════════════════════════════════════════════════════════
// Enhanced Service Methods
// ════════════════════════════════════════════════════════════════════════════

// NotifyUser dispatches a notification through all enabled providers with retry.
// This is the production-hardened version of the existing Notify method.
func (s *Service) NotifyUser(input NotifyInput) {
	cfg := LoadProviderConfig()
	logCtx := map[string]interface{}{
		"user_id": input.UserID.String(),
		"type":    input.Type,
	}

	// In-app + WebSocket (existing pipeline)
	s.Notify(input)

	// Push notification via FCM with retry
	if cfg.EnablePush && s.fcm != nil {
		SafeGo(func() {
			provider := &FirebaseProvider{client: s.fcm}
			var tokens []PushToken
			s.db.Where("user_id = ?", input.UserID).Find(&tokens)
			for _, t := range tokens {
				token := t.Token
				if err := retrySend(func() error {
					return provider.SendPush(token, input.Title, input.Body, toInterfaceMap(input.Data))
				}, "fcm", logCtx); err != nil {
					IncrementDeliveryStat("push_failed")
					LogFailedNotification(s.db, input.UserID, "push", input.Type, toInterfaceMap(input.Data), err.Error())
				} else {
					IncrementDeliveryStat("push_sent")
				}
			}
		}, "push:"+input.Type)
	}

	// Email with retry
	if cfg.EnableEmail {
		SafeGo(func() {
			var u struct {
				Email string
				Name  string
			}
			s.db.Table("users").Where("id = ?", input.UserID).Select("email, name").Scan(&u)
			if u.Email != "" {
				msg := &email.Message{
					To:             u.Email,
					ToName:         u.Name,
					UserID:         input.UserID.String(),
					Subject:        input.Title,
					TemplateName:   "notification",
					Data:           email.NotificationData(u.Name, input.Title, input.Body, "", ""),
					IdempotencyKey: "notify:" + input.UserID.String() + ":" + input.Type,
					CreatedAt:      time.Now(),
				}
				if err := retrySend(func() error {
					return email.Default().SendAsync(context.Background(), msg)
				}, "email", logCtx); err != nil {
					IncrementDeliveryStat("email_failed")
					LogFailedNotification(s.db, input.UserID, "email", input.Type, toInterfaceMap(input.Data), err.Error())
				} else {
					IncrementDeliveryStat("email_sent")
				}
			}
		}, "email:"+input.Type)
	}
}

// NotifyBulk dispatches notifications to multiple users.
func (s *Service) NotifyBulk(inputs []NotifyInput) {
	for _, input := range inputs {
		input := input
		SafeGo(func() {
			s.NotifyUser(input)
		}, "bulk:"+input.Type)
	}
}

// DeliveryStats tracks notification delivery metrics.
type DeliveryStats struct {
	PushSent    int64 `json:"push_sent"`
	PushFailed  int64 `json:"push_failed"`
	EmailSent   int64 `json:"email_sent"`
	EmailFailed int64 `json:"email_failed"`
}

var deliveryStats DeliveryStats

// IncrementDeliveryStat atomically increments a delivery counter.
func IncrementDeliveryStat(field string) {
	switch field {
	case "push_sent":
		atomic.AddInt64(&deliveryStats.PushSent, 1)
	case "push_failed":
		atomic.AddInt64(&deliveryStats.PushFailed, 1)
	case "email_sent":
		atomic.AddInt64(&deliveryStats.EmailSent, 1)
	case "email_failed":
		atomic.AddInt64(&deliveryStats.EmailFailed, 1)
	}
}

func (s *Service) GetDeliveryStats() DeliveryStats {
	return DeliveryStats{
		PushSent:    atomic.LoadInt64(&deliveryStats.PushSent),
		PushFailed:  atomic.LoadInt64(&deliveryStats.PushFailed),
		EmailSent:   atomic.LoadInt64(&deliveryStats.EmailSent),
		EmailFailed: atomic.LoadInt64(&deliveryStats.EmailFailed),
	}
}

func toInterfaceMap(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// ensure uuid import used
var _ = uuid.Nil

// ════════════════════════════════════════════════════════════════════════════
// Failed Notification Log (Sprint 8.5)
// Records notification delivery failures for fallback tracking.
// ════════════════════════════════════════════════════════════════════════════

type FailedNotification struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null;index:idx_fn_user" json:"user_id"`
	Channel      string     `gorm:"size:20;not null;index:idx_fn_channel" json:"channel"`
	EventType    string     `gorm:"size:100;not null" json:"event_type"`
	Payload      string     `gorm:"type:jsonb" json:"payload,omitempty"`
	ErrorMessage *string    `gorm:"type:text" json:"error_message,omitempty"`
	RetryCount   int        `gorm:"not null;default:0" json:"retry_count"`
	LastRetryAt  *time.Time `json:"last_retry_at,omitempty"`
	Resolved     bool       `gorm:"not null;default:false;index:idx_fn_unresolved" json:"resolved"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (FailedNotification) TableName() string { return "failed_notifications" }

// LogFailedNotification records a failed notification delivery attempt.
func LogFailedNotification(db *gorm.DB, userID uuid.UUID, channel, eventType string, payload map[string]interface{}, errMsg string) {
	if db == nil {
		return
	}
	msg := errMsg
	entry := FailedNotification{
		ID:           uuid.New(),
		UserID:       userID,
		Channel:      channel,
		EventType:    eventType,
		ErrorMessage: &msg,
	}
	if payload != nil {
		b, _ := json.Marshal(payload)
		entry.Payload = string(b)
	}
	if err := db.Create(&entry).Error; err != nil {
		slog.Error("notifications: failed to log failed_notification", "error", err)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// Retry Worker (Sprint 8.5)
// Periodically retries unresolved failed notifications.
// ════════════════════════════════════════════════════════════════════════════

const maxRetryAttempts = 5

// RetryFailedNotificationsWorker runs a background loop that retries
// unresolved failed notifications every minute.
func RetryFailedNotificationsWorker(db *gorm.DB, svc *Service) {
	if db == nil || svc == nil {
		return
	}
	slog.Info("notifications: retry worker started")
	for {
		var pending []FailedNotification
		if err := db.Where("resolved = ? AND retry_count < ?", false, maxRetryAttempts).
			Order("created_at ASC").Limit(50).Find(&pending).Error; err != nil {
			slog.Error("notifications: retry worker query failed", "error", err)
			time.Sleep(1 * time.Minute)
			continue
		}
		for i := range pending {
			n := &pending[i]
			var payload map[string]interface{}
			if n.Payload != "" {
				_ = json.Unmarshal([]byte(n.Payload), &payload)
			}
			input := NotifyInput{
				UserID: n.UserID,
				Type:   n.EventType,
				Title:  n.EventType, // best-effort; original title not stored
				Body:   "",
				Data:   nil,
			}
			if payload != nil {
				if t, ok := payload["title"].(string); ok {
					input.Title = t
				}
				if b, ok := payload["body"].(string); ok {
					input.Body = b
				}
			}

			// Attempt re-delivery via the appropriate channel
			var retryErr error
			switch n.Channel {
			case "push":
				cfg := LoadProviderConfig()
				if cfg.EnablePush && svc.fcm != nil {
					provider := &FirebaseProvider{client: svc.fcm}
					var tokens []PushToken
					db.Where("user_id = ?", n.UserID).Find(&tokens)
					for _, t := range tokens {
						if err := provider.SendPush(t.Token, input.Title, input.Body, payload); err != nil {
							retryErr = err
						}
					}
				}
			case "email":
				cfg := LoadProviderConfig()
				if cfg.EnableEmail {
					provider := &EmailProvider{db: db}
					var email string
					db.Table("users").Where("id = ?", n.UserID).Select("email").Scan(&email)
					if email != "" {
						retryErr = provider.SendEmail(email, input.Title, input.Body)
					}
				}
			}

			now := time.Now()
			if retryErr != nil {
				// Still failing — increment retry count
				db.Model(n).Updates(map[string]interface{}{
					"retry_count":   n.RetryCount + 1,
					"last_retry_at": &now,
				})
				slog.Warn("notifications: retry still failing", "id", n.ID, "channel", n.Channel, "attempt", n.RetryCount+1)
			} else {
				// Success — mark resolved
				db.Model(n).Updates(map[string]interface{}{
					"resolved":      true,
					"retry_count":   n.RetryCount + 1,
					"last_retry_at": &now,
				})
				slog.Info("notifications: retry succeeded", "id", n.ID, "channel", n.Channel)
			}
		}
		time.Sleep(1 * time.Minute)
	}
}
