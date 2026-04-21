package exchange

import "errors"

const (
	PaymentMethodInstapay = "instapay"
	PaymentMethodBank     = "bank"
	PaymentMethodPayPal   = "paypal"
	PaymentMethodCash     = "cash"
)

type PaymentMethodConfig struct {
	Name              string
	ConfirmationHours int
	AllowedProofTypes []string
	FraudRiskWeight   float64 // 0–1, higher = more risky
}

var paymentMethodConfigs = map[string]PaymentMethodConfig{
	PaymentMethodInstapay: {"Instapay", 1, []string{"screenshot", "transaction_id"}, 0.10},
	PaymentMethodBank:     {"Bank Transfer", 24, []string{"receipt", "transaction_id", "screenshot"}, 0.15},
	PaymentMethodPayPal:   {"PayPal", 1, []string{"screenshot", "transaction_id"}, 0.25},
	PaymentMethodCash:     {"Cash", 0, []string{"photo"}, 0.40},
}

func ValidatePaymentMethod(method string) error {
	if _, ok := paymentMethodConfigs[method]; !ok {
		return errors.New("payment_method: must be instapay, bank, paypal, or cash")
	}
	return nil
}

func GetPaymentMethodConfig(method string) (PaymentMethodConfig, bool) {
	c, ok := paymentMethodConfigs[method]
	return c, ok
}

func PaymentMethodRiskLabel(method string) string {
	cfg, ok := paymentMethodConfigs[method]
	if !ok {
		return "unknown"
	}
	switch {
	case cfg.FraudRiskWeight >= 0.35:
		return RiskHigh
	case cfg.FraudRiskWeight >= 0.20:
		return RiskMedium
	default:
		return RiskLow
	}
}
