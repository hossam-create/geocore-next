package constants

// CountryComplianceConfig is a static regional compliance table used by financial and KYC-sensitive flows.
type CountryComplianceConfig struct {
	CountryCode            string
	CountryName            string
	Currency               string
	KYCRequiredAbove       float64
	WithdrawalsAllowed     bool
	EscrowAllowed          bool
	CryptoAllowed          bool
	SanctionsScreeningTier string
}

var CountryComplianceTable = map[string]CountryComplianceConfig{
	"AE": {
		CountryCode:            "AE",
		CountryName:            "United Arab Emirates",
		Currency:               "AED",
		KYCRequiredAbove:       2000,
		WithdrawalsAllowed:     true,
		EscrowAllowed:          true,
		CryptoAllowed:          true,
		SanctionsScreeningTier: "enhanced",
	},
	"SA": {
		CountryCode:            "SA",
		CountryName:            "Saudi Arabia",
		Currency:               "SAR",
		KYCRequiredAbove:       2000,
		WithdrawalsAllowed:     true,
		EscrowAllowed:          true,
		CryptoAllowed:          false,
		SanctionsScreeningTier: "enhanced",
	},
	"KW": {
		CountryCode:            "KW",
		CountryName:            "Kuwait",
		Currency:               "KWD",
		KYCRequiredAbove:       2000,
		WithdrawalsAllowed:     true,
		EscrowAllowed:          true,
		CryptoAllowed:          false,
		SanctionsScreeningTier: "enhanced",
	},
	"QA": {
		CountryCode:            "QA",
		CountryName:            "Qatar",
		Currency:               "QAR",
		KYCRequiredAbove:       2000,
		WithdrawalsAllowed:     true,
		EscrowAllowed:          true,
		CryptoAllowed:          false,
		SanctionsScreeningTier: "enhanced",
	},
	"BH": {
		CountryCode:            "BH",
		CountryName:            "Bahrain",
		Currency:               "BHD",
		KYCRequiredAbove:       2000,
		WithdrawalsAllowed:     true,
		EscrowAllowed:          true,
		CryptoAllowed:          false,
		SanctionsScreeningTier: "enhanced",
	},
	"OM": {
		CountryCode:            "OM",
		CountryName:            "Oman",
		Currency:               "OMR",
		KYCRequiredAbove:       2000,
		WithdrawalsAllowed:     true,
		EscrowAllowed:          true,
		CryptoAllowed:          false,
		SanctionsScreeningTier: "enhanced",
	},
}

func GetCountryCompliance(code string) (CountryComplianceConfig, bool) {
	cfg, ok := CountryComplianceTable[code]
	return cfg, ok
}
