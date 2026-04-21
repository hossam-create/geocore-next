package aiops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Notifier sends incident alerts to Slack and PagerDuty.
// Both are no-op when the respective env var is not set.
type Notifier struct {
	slackWebhookURL string
	pagerdutyKey    string
	client          *http.Client
}

func NewNotifier() *Notifier {
	return &Notifier{
		slackWebhookURL: os.Getenv("SLACK_WEBHOOK_URL"),
		pagerdutyKey:    os.Getenv("PAGERDUTY_KEY"),
		client:          &http.Client{Timeout: 10 * time.Second},
	}
}

// ── Slack ────────────────────────────────────────────────────────────────────

type slackPayload struct {
	Text        string            `json:"text"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
}

type slackAttachment struct {
	Color  string `json:"color"`
	Title  string `json:"title"`
	Text   string `json:"text"`
	Footer string `json:"footer"`
	Ts     int64  `json:"ts"`
}

// Send posts an incident alert to Slack.
func (n *Notifier) Send(ctx context.Context, inc *Incident) error {
	if n.slackWebhookURL == "" {
		return nil
	}

	color, emoji := "#ffcc00", "⚠️"
	switch inc.Severity {
	case SeverityP0:
		color, emoji = "#ff0000", "🚨"
	case SeverityP2:
		color, emoji = "#36a64f", "📊"
	}

	headerText := fmt.Sprintf("%s *[%s] %s*\nService: `%s` | Metric: `%s` = `%.4f` (baseline `%.4f`)",
		emoji, inc.Severity, inc.Title, inc.Service, inc.Metric, inc.Value, inc.Baseline)

	payload := slackPayload{
		Text: headerText,
		Attachments: []slackAttachment{
			{
				Color:  color,
				Title:  "🧠 Root Cause Analysis",
				Text:   truncate(inc.RCA, 500),
				Footer: "GeoCore AIOps",
				Ts:     inc.DetectedAt.Unix(),
			},
			{
				Color:  "#0066cc",
				Title:  "🧯 Suggested Runbook",
				Text:   truncate(inc.Runbook, 500),
				Footer: fmt.Sprintf("Incident ID: %s | Approve: POST /api/v1/aiops/incidents/%s/resolve", inc.ID, inc.ID),
				Ts:     inc.DetectedAt.Unix(),
			},
		},
	}

	return n.postJSON(ctx, n.slackWebhookURL, payload)
}

// ── PagerDuty ────────────────────────────────────────────────────────────────

type pdPayload struct {
	Summary       string            `json:"summary"`
	Severity      string            `json:"severity"`
	Source        string            `json:"source"`
	CustomDetails map[string]string `json:"custom_details"`
}

type pdEvent struct {
	RoutingKey  string    `json:"routing_key"`
	EventAction string    `json:"event_action"`
	Payload     pdPayload `json:"payload"`
}

// SendPagerDuty fires a PagerDuty event for P0 incidents.
func (n *Notifier) SendPagerDuty(ctx context.Context, inc *Incident) error {
	if n.pagerdutyKey == "" || inc.Severity != SeverityP0 {
		return nil
	}

	event := pdEvent{
		RoutingKey:  n.pagerdutyKey,
		EventAction: "trigger",
		Payload: pdPayload{
			Summary:  fmt.Sprintf("[P0] %s", inc.Title),
			Severity: strings.ToLower(string(inc.Severity)),
			Source:   "geocore-aiops",
			CustomDetails: map[string]string{
				"service":    inc.Service,
				"metric":     inc.Metric,
				"value":      fmt.Sprintf("%.4f", inc.Value),
				"baseline":   fmt.Sprintf("%.4f", inc.Baseline),
				"incident_id": inc.ID,
				"runbook":    truncate(inc.Runbook, 200),
			},
		},
	}

	return n.postJSON(ctx, "https://events.pagerduty.com/v2/enqueue", event)
}

// ── internal ─────────────────────────────────────────────────────────────────

func (n *Notifier) postJSON(ctx context.Context, endpoint string, payload interface{}) error {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
