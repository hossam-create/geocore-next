package observability

import (
	"errors"
	"testing"
)

// ════════════════════════════════════════════════════════════════════════════
// STEP 4: Error Tracking Tests
// ════════════════════════════════════════════════════════════════════════════

func TestCaptureError_Nil(t *testing.T) {
	// Should not panic on nil error
	CaptureError(nil, map[string]interface{}{"test": true})
}

func TestCaptureError_WithMessage(t *testing.T) {
	// Should not panic
	CaptureError(errors.New("test error"), map[string]interface{}{
		"user_id":  "123",
		"action":   "payment",
		"amount":   100,
	})
}

func TestCaptureError_NilContext(t *testing.T) {
	CaptureError(errors.New("test"), nil)
}

func TestFinancialLog(t *testing.T) {
	// Should not panic
	FinancialLog("escrow_hold", "user-123", "req-456", 10000, "success")
}

func TestSafeGo_NoPanic(t *testing.T) {
	done := make(chan bool, 1)
	SafeGo(func() {
		done <- true
	}, "test")
	<-done
}

func TestSafeGo_WithPanic(t *testing.T) {
	done := make(chan bool, 1)
	SafeGo(func() {
		panic("test panic")
	}, "test_panic")
	// Give goroutine time to recover
	SafeGo(func() {
		done <- true
	}, "done")
	<-done
}
