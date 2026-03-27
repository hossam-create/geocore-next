package chat

import (
	"github.com/geocore-next/backend/internal/notifications"
	"github.com/google/uuid"
)

// NotificationService is an interface satisfied by *notifications.Service.
type NotificationService interface {
	Notify(input notifications.NotifyInput)
}

var globalNotifSvc NotificationService

// SetNotificationService wires the notification service into this package.
// Called once from main.go after all routes are registered.
func SetNotificationService(svc NotificationService) {
	globalNotifSvc = svc
}

func notifyNewMessage(recipientID uuid.UUID, senderName, snippet, conversationID string) {
	if globalNotifSvc == nil {
		return
	}

	body := snippet
	if len(body) > 100 {
		body = body[:97] + "..."
	}

	go globalNotifSvc.Notify(notifications.NotifyInput{
		UserID: recipientID,
		Type:   notifications.TypeNewMessage,
		Title:  "New message from " + senderName,
		Body:   body,
		Data:   map[string]string{"conversation_id": conversationID},
	})
}
