package billing

import "fmt"

// unitPrices maps each event type to its per-unit cost in USD.
// All costs are in USD (not cents) for readability; we convert when computing.
var unitPrices = map[EventType]float64{
	EventRequests:      0.01 / 1000,       // $0.01 per 1K requests
	EventKafkaEvents:   0.10 / 1_000_000,  // $0.10 per 1M events
	EventAIOpsIncident: 5.00,              // $5.00 per incident resolved
	EventChaosRun:      1.00,              // $1.00 per chaos/stress run
	EventStorageGBHour: 0.05 / 730,        // $0.05/GB/month ÷ 730h
}

// LineItem is a single priced entry on an invoice.
type LineItem struct {
	EventType   EventType `json:"event_type"`
	Quantity    int64     `json:"quantity"`
	UnitPriceUS float64   `json:"unit_price_usd"`
	TotalCents  int64     `json:"total_cents"`
	Description string    `json:"description"`
}

// InvoiceBreakdown is the full itemised cost breakdown for a billing period.
type InvoiceBreakdown struct {
	Lines         []LineItem `json:"lines"`
	SubtotalCents int64      `json:"subtotal_cents"` // usage charges only
	PlanCents     int64      `json:"plan_cents"`     // base subscription fee
	TotalCents    int64      `json:"total_cents"`
	TotalUSD      string     `json:"total_usd"`
}

// Compute calculates the invoice breakdown from a usage summary and the tenant's plan.
func Compute(usage UsageSummary, plan Plan) InvoiceBreakdown {
	var lines []LineItem
	var subtotal int64

	for et, qty := range usage.Events {
		up, ok := unitPrices[et]
		if !ok || qty == 0 {
			continue
		}
		totalCents := int64(float64(qty) * up * 100)
		lines = append(lines, LineItem{
			EventType:   et,
			Quantity:    qty,
			UnitPriceUS: up,
			TotalCents:  totalCents,
			Description: fmt.Sprintf("%s × %d units", et, qty),
		})
		subtotal += totalCents
	}

	total := subtotal + int64(plan.MonthlyPrice)
	return InvoiceBreakdown{
		Lines:         lines,
		SubtotalCents: subtotal,
		PlanCents:     int64(plan.MonthlyPrice),
		TotalCents:    total,
		TotalUSD:      fmt.Sprintf("$%.2f", float64(total)/100),
	}
}
