package warroom

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/geocore-next/backend/internal/aiops"
)

// DashboardView is a complete snapshot of the system's operational state.
type DashboardView struct {
	State          SystemState       `json:"state"`
	HealthStatus   string            `json:"health_status"`
	OpenIncidents  int               `json:"open_incidents"`
	StateHistory   []StateTransition `json:"state_history"`
	PendingActions []PendingAction   `json:"pending_actions"`
	LastEvaluated  time.Time         `json:"last_evaluated"`
	Uptime         string            `json:"uptime"`
}

// dashboardBuilder queries live system state from in-process sources.
type dashboardBuilder struct {
	targetURL string
	client    *http.Client
	startTime time.Time
}

func newDashboardBuilder() *dashboardBuilder {
	target := os.Getenv("STRESS_TARGET_URL")
	if target == "" {
		target = "http://localhost:8080"
	}
	return &dashboardBuilder{
		targetURL: target,
		client:    &http.Client{Timeout: 3 * time.Second},
		startTime: time.Now(),
	}
}

// Build assembles a DashboardView from live signals.
func (b *dashboardBuilder) Build(
	state SystemState,
	history []StateTransition,
	pending []PendingAction,
) DashboardView {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	openCount := aiops.GetOpenCount()
	health := b.checkHealth(ctx)

	return DashboardView{
		State:          state,
		HealthStatus:   health,
		OpenIncidents:  openCount,
		StateHistory:   history,
		PendingActions: pending,
		LastEvaluated:  time.Now(),
		Uptime:         formatUptime(time.Since(b.startTime)),
	}
}

func (b *dashboardBuilder) checkHealth(ctx context.Context) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.targetURL+"/health", nil)
	if err != nil {
		return "unknown"
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return "unreachable"
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
		return "healthy"
	case 206:
		return "degraded"
	default:
		return fmt.Sprintf("status_%d", resp.StatusCode)
	}
}

func formatUptime(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dh %dm %ds", h, m, s)
}
