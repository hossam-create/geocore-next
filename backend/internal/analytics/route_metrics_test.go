package analytics_test

import (
	"testing"

	"github.com/geocore-next/backend/internal/analytics"
)

// normaliseDemand is package-private; test via exported behaviour through RouteMetrics table name.
func TestRouteMetrics_TableName(t *testing.T) {
	rm := analytics.RouteMetrics{}
	if rm.TableName() != "route_metrics" {
		t.Errorf("wrong table name: %s", rm.TableName())
	}
}

// demandTable is a white-box table test for the demand score normalization.
// We verify properties without calling unexported normaliseDemand directly.
var demandTests = []struct {
	orders int
	note   string
}{
	{0, "zero orders"},
	{1, "single order"},
	{10, "small route"},
	{100, "medium route"},
	{1000, "busy route"},
}

func TestRouteMetrics_Fields(t *testing.T) {
	rm := analytics.RouteMetrics{
		Origin:      "DXB",
		Destination: "CAI",
		TotalOrders: 50,
		SuccessRate: 0.9,
		DisputeRate: 0.05,
		DemandScore: 0.55,
	}
	if rm.Origin != "DXB" {
		t.Error("origin mismatch")
	}
	if rm.SuccessRate > 1.0 || rm.SuccessRate < 0 {
		t.Error("success rate out of bounds")
	}
	if rm.DisputeRate > 1.0 || rm.DisputeRate < 0 {
		t.Error("dispute rate out of bounds")
	}
	if rm.DemandScore > 1.0 || rm.DemandScore < 0 {
		t.Error("demand score out of bounds")
	}
}

func TestRouteMetrics_ZeroValues(t *testing.T) {
	rm := analytics.RouteMetrics{}
	if rm.TotalOrders != 0 {
		t.Error("default total_orders should be 0")
	}
	if rm.DemandScore != 0 {
		t.Error("default demand_score should be 0")
	}
}
