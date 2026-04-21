package controlplane

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"
)

var pagerURL string

func init() {
	pagerURL = os.Getenv("PAGER_URL") // e.g. Opsgenie, PagerDuty webhook
}

// TriggerPager sends an alert to the configured pager service.
func TriggerPager(message string) {
	if pagerURL == "" {
		slog.Error("controlplane: pager triggered but PAGER_URL not configured", "message", message)
		return
	}

	payload := map[string]any{
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"source":    "geocore-controlplane",
		"severity":  "critical",
	}

	body, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(pagerURL, "application/json", bytes.NewReader(body))
	if err != nil {
		slog.Error("controlplane: pager failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		slog.Error("controlplane: pager returned non-2xx", "status", resp.StatusCode)
		return
	}

	slog.Info("controlplane: pager triggered successfully", "message", message)
}

// TriggerPagerWithSeverity sends an alert with a custom severity level.
func TriggerPagerWithSeverity(message, severity string) {
	if pagerURL == "" {
		slog.Warn("controlplane: pager triggered but PAGER_URL not configured",
			"message", message, "severity", severity)
		return
	}

	payload := map[string]any{
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"source":    "geocore-controlplane",
		"severity":  severity,
	}

	body, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(pagerURL, "application/json", bytes.NewReader(body))
	if err != nil {
		slog.Error("controlplane: pager failed", "error", err)
		return
	}
	defer resp.Body.Close()

	slog.Info("controlplane: pager triggered", "message", message, "severity", severity)
}
