package notifications

import (
	"errors"
	"testing"
)

// ════════════════════════════════════════════════════════════════════════════
// STEP 1: Notification Provider Tests
// ════════════════════════════════════════════════════════════════════════════

func TestNotificationProvider_Interface(t *testing.T) {
	// Verify FirebaseProvider implements NotificationProvider
	var _ NotificationProvider = &FirebaseProvider{}
	// Verify EmailProvider implements NotificationProvider
	var _ NotificationProvider = &EmailProvider{}
}

func TestFirebaseProvider_SendPush_NoClient(t *testing.T) {
	p := &FirebaseProvider{client: nil}
	err := p.SendPush("token", "title", "body", map[string]interface{}{"key": "val"})
	if err == nil {
		t.Error("should error when FCM client is nil")
	}
}

func TestFirebaseProvider_SendEmail_Noop(t *testing.T) {
	p := &FirebaseProvider{}
	err := p.SendEmail("test@test.com", "subject", "body")
	if err != nil {
		t.Error("FirebaseProvider.SendEmail should be noop")
	}
}

func TestEmailProvider_SendPush_Noop(t *testing.T) {
	p := &EmailProvider{}
	err := p.SendPush("user", "title", "body", nil)
	if err != nil {
		t.Error("EmailProvider.SendPush should be noop")
	}
}

func TestProviderConfig_Defaults(t *testing.T) {
	cfg := LoadProviderConfig()
	if !cfg.EnablePush {
		t.Error("push should be enabled by default")
	}
	if !cfg.EnableEmail {
		t.Error("email should be enabled by default")
	}
}

func TestRetrySend_Success(t *testing.T) {
	called := 0
	retrySend(func() error {
		called++
		return nil
	}, "test", map[string]interface{}{"test": true})
	if called != 1 {
		t.Errorf("expected 1 call on success, got %d", called)
	}
}

func TestRetrySend_RetryOnFailure(t *testing.T) {
	called := 0
	retrySend(func() error {
		called++
		if called < 3 {
			return errors.New("temporary failure")
		}
		return nil
	}, "test", nil)
	if called != 3 {
		t.Errorf("expected 3 calls with retry, got %d", called)
	}
}

func TestRetrySend_MaxRetries(t *testing.T) {
	called := 0
	retrySend(func() error {
		called++
		return errors.New("permanent failure")
	}, "test", nil)
	if called != maxRetries {
		t.Errorf("expected %d calls (max retries), got %d", maxRetries, called)
	}
}

func TestSafeGo_NoPanic(t *testing.T) {
	done := make(chan bool, 1)
	SafeGo(func() {
		done <- true
	}, "test")
	<-done
}

func TestSafeGo_PanicRecovery(t *testing.T) {
	done := make(chan bool, 1)
	SafeGo(func() {
		panic("test panic")
	}, "test_panic")
	// If we get here, the panic was recovered
	SafeGo(func() {
		done <- true
	}, "done")
	<-done
}

func TestToInterfaceMap(t *testing.T) {
	m := map[string]string{"key": "val", "foo": "bar"}
	result := toInterfaceMap(m)
	if len(result) != 2 {
		t.Errorf("expected 2 keys, got %d", len(result))
	}
	if result["key"] != "val" {
		t.Error("key mismatch")
	}
}

func TestDeliveryStats_Initial(t *testing.T) {
	stats := DeliveryStats{}
	if stats.PushSent != 0 || stats.PushFailed != 0 || stats.EmailSent != 0 || stats.EmailFailed != 0 {
		t.Error("initial stats should be zero")
	}
}

func TestIncrementDeliveryStat(t *testing.T) {
	// Reset
	deliveryStats = DeliveryStats{}
	IncrementDeliveryStat("push_sent")
	IncrementDeliveryStat("push_sent")
	IncrementDeliveryStat("email_failed")
	if deliveryStats.PushSent != 2 {
		t.Errorf("expected push_sent=2, got %d", deliveryStats.PushSent)
	}
	if deliveryStats.EmailFailed != 1 {
		t.Errorf("expected email_failed=1, got %d", deliveryStats.EmailFailed)
	}
}
