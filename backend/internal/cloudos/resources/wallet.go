package resources

// WalletResource represents the wallet service as a managed resource.
type WalletResource struct {
	BalanceInvariant bool    `json:"balance_invariant"` // MUST be true
	FraudGuard      bool    `json:"fraud_guard"`       // fraud detection active
	Replicas        int     `json:"replicas"`
	ErrorRate       float64 `json:"error_rate"`
	TxPerSecond     float64 `json:"tx_per_second"`
}

// DefaultWalletResource returns production defaults.
func DefaultWalletResource() WalletResource {
	return WalletResource{
		BalanceInvariant: true,
		FraudGuard:       true,
		Replicas:         3,
	}
}

// IsHealthy returns true if wallet invariants are maintained.
func (w *WalletResource) IsHealthy() bool {
	return w.BalanceInvariant && w.FraudGuard && w.ErrorRate < 0.5
}

// CriticalAlert returns true if balance invariant is violated.
func (w *WalletResource) CriticalAlert() bool {
	return !w.BalanceInvariant
}
