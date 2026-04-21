package crowdshipping

import (
	"log/slog"
	"strings"
	"sync"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// ── Compliance types ──────────────────────────────────────────────────────────

type EnforcementLevel string

const (
	EnforcementSoft EnforcementLevel = "SOFT_LIMIT" // allow but warn
	EnforcementHard EnforcementLevel = "HARD_LIMIT" // block transaction
)

type ComplianceBoundary struct {
	Type             string           `json:"type"`
	Jurisdiction     string           `json:"jurisdiction"`
	Requirement      string           `json:"requirement"`
	Status           string           `json:"status"`
	Explanation      string           `json:"explanation"`
	EnforcementLevel EnforcementLevel `json:"enforcement_level"`
}

type ComplianceThresholds struct {
	KYCThresholdUSD           float64          `json:"kyc_threshold_usd"`
	EnhancedReviewUSD         float64          `json:"enhanced_review_usd"`
	HardBlockUSD              float64          `json:"hard_block_usd"`
	CrossBorderReportUSD      float64          `json:"cross_border_report_usd"`
	DefaultEscrowUSD          float64          `json:"default_escrow_usd"`
	EnhancedReviewEnforcement EnforcementLevel `json:"enhanced_review_enforcement"`
}

func DefaultComplianceThresholds() ComplianceThresholds {
	return ComplianceThresholds{
		KYCThresholdUSD:           1000,
		EnhancedReviewUSD:         10000,
		HardBlockUSD:              50000,
		CrossBorderReportUSD:      10000,
		DefaultEscrowUSD:          200,
		EnhancedReviewEnforcement: EnforcementSoft,
	}
}

var globalThresholdsMu sync.RWMutex
var globalThresholds = DefaultComplianceThresholds()

func GetComplianceThresholds() ComplianceThresholds {
	globalThresholdsMu.RLock()
	defer globalThresholdsMu.RUnlock()
	return globalThresholds
}

func SetComplianceThresholds(t ComplianceThresholds) {
	globalThresholdsMu.Lock()
	defer globalThresholdsMu.Unlock()
	globalThresholds = t
}

type ComplianceResult struct {
	Corridor           string               `json:"corridor"`
	AmountUSD          float64              `json:"amount_usd"`
	Allowed            bool                 `json:"allowed"`
	Warnings           []string             `json:"warnings"`
	KYCRequired        bool                 `json:"kyc_required"`
	EscrowRecommended  bool                 `json:"escrow_recommended"`
	MinBuyerTrust      TrustRequirement     `json:"min_buyer_trust"`
	MinTravelerTrust   TrustRequirement     `json:"min_traveler_trust"`
	HighValueThreshold float64              `json:"high_value_threshold"`
	Boundaries         []ComplianceBoundary `json:"boundaries"`
	ItemRestrictions   []string             `json:"item_restrictions,omitempty"`
}

// ── Sanctions list ────────────────────────────────────────────────────────────

var sanctionedCountries = map[string]bool{
	"IR": true,
	"KP": true,
	"SY": true,
	"CU": true,
}

// ── Core compliance check ─────────────────────────────────────────────────────

func CheckCompliance(origin, dest string, amountUSD float64, itemCategories ...string) ComplianceResult {
	return CheckComplianceWithConfig(origin, dest, amountUSD, GetComplianceThresholds(), itemCategories...)
}

func CheckComplianceWithConfig(origin, dest string, amountUSD float64, thresholds ComplianceThresholds, itemCategories ...string) ComplianceResult {
	originUpper := strings.ToUpper(origin)
	destUpper := strings.ToUpper(dest)
	corridorID := originUpper + "_" + destUpper
	result := ComplianceResult{
		Corridor:   corridorID,
		AmountUSD:  amountUSD,
		Allowed:    true,
		Warnings:   make([]string, 0),
		Boundaries: make([]ComplianceBoundary, 0),
	}

	// 1. Sanctions — always HARD block
	if sanctionedCountries[originUpper] || sanctionedCountries[destUpper] {
		result.Allowed = false
		result.Boundaries = append(result.Boundaries, ComplianceBoundary{
			Type:             "SANCTIONS",
			Jurisdiction:     "GLOBAL",
			Requirement:      "Sanctions compliance check",
			Status:           "BLOCKED",
			Explanation:      "Corridor is subject to international sanctions.",
			EnforcementLevel: EnforcementHard,
		})
		result.Warnings = append(result.Warnings, "Corridor blocked due to sanctions restrictions")
		slog.Info("crowdshipping: compliance blocked",
			"corridor", corridorID,
			"amount_usd", amountUSD,
			"allowed", false,
			"reason", "sanctions",
		)
		return result
	}

	cfg := GetCorridorConfig(origin, dest)

	// 2. Hard block — amount exceeds configurable hard limit
	if thresholds.HardBlockUSD > 0 && amountUSD > thresholds.HardBlockUSD {
		result.Allowed = false
		result.Boundaries = append(result.Boundaries, ComplianceBoundary{
			Type:             "HARD_BLOCK",
			Jurisdiction:     "GLOBAL",
			Requirement:      "Transaction exceeds hard block threshold",
			Status:           "BLOCKED",
			Explanation:      "Transactions over $50,000 are prohibited without special authorization.",
			EnforcementLevel: EnforcementHard,
		})
		result.Warnings = append(result.Warnings, "Transaction amount exceeds hard block threshold")
		slog.Info("crowdshipping: compliance blocked",
			"corridor", corridorID,
			"amount_usd", amountUSD,
			"allowed", false,
			"reason", "hard_block",
		)
		return result
	}

	// 3. KYC — soft limit
	if amountUSD > thresholds.KYCThresholdUSD {
		result.KYCRequired = true
		result.Boundaries = append(result.Boundaries, ComplianceBoundary{
			Type:             "KYC_REQUIREMENT",
			Jurisdiction:     origin,
			Requirement:      "Enhanced KYC verification required for transactions over $1000",
			Status:           "PENDING",
			Explanation:      "Identity verification must be completed before proceeding.",
			EnforcementLevel: EnforcementSoft,
		})
	}

	// 4. AML — always compliant
	result.Boundaries = append(result.Boundaries, ComplianceBoundary{
		Type:             "AML_CHECK",
		Jurisdiction:     "GLOBAL",
		Requirement:      "Anti-Money Laundering screening",
		Status:           "COMPLIANT",
		Explanation:      "Standard AML checks will be performed on all transactions.",
		EnforcementLevel: EnforcementSoft,
	})

	// 5. Enhanced review — soft or hard configurable
	if amountUSD > thresholds.EnhancedReviewUSD {
		status := "REVIEW_REQUIRED"
		if thresholds.EnhancedReviewEnforcement == EnforcementHard {
			result.Allowed = false
			status = "BLOCKED"
		}
		result.Boundaries = append(result.Boundaries, ComplianceBoundary{
			Type:             "CROSS_BORDER_LIMIT",
			Jurisdiction:     origin,
			Requirement:      "Cross-border transaction reporting threshold",
			Status:           status,
			Explanation:      "Transactions over $10,000 require additional documentation.",
			EnforcementLevel: thresholds.EnhancedReviewEnforcement,
		})
		result.Warnings = append(result.Warnings, "Prepare source of funds documentation for large transfers")
	}

	// 6. Destination-specific currency restrictions
	if strings.EqualFold(dest, "EG") {
		result.Boundaries = append(result.Boundaries, ComplianceBoundary{
			Type:             "CURRENCY_RESTRICTION",
			Jurisdiction:     "EG",
			Requirement:      "Egyptian Pound conversion restrictions",
			Status:           "COMPLIANT",
			Explanation:      "EGP conversions subject to Central Bank of Egypt regulations.",
			EnforcementLevel: EnforcementSoft,
		})
	}

	// 7. Corridor-specific config
	if cfg != nil {
		result.EscrowRecommended = cfg.EscrowPolicy == EscrowAlwaysRecommended ||
			(cfg.EscrowPolicy == EscrowHighValueOnly && amountUSD >= cfg.Trust.HighValueThreshold)

		result.MinBuyerTrust = cfg.Trust.MinBuyerTrust
		result.MinTravelerTrust = cfg.Trust.MinTravelerTrust
		result.HighValueThreshold = cfg.Trust.HighValueThreshold

		if amountUSD >= cfg.Trust.HighValueThreshold {
			result.Warnings = append(result.Warnings, "High-value transaction: escrow and verified trust recommended")
		}

		for _, cat := range itemCategories {
			if IsRestricted(cfg, cat) {
				result.ItemRestrictions = append(result.ItemRestrictions, cat)
				result.Warnings = append(result.Warnings, "Item category '"+cat+"' is restricted for this corridor")
			}
		}
	} else {
		result.EscrowRecommended = amountUSD >= thresholds.DefaultEscrowUSD
		result.MinBuyerTrust = TrustStandard
		result.MinTravelerTrust = TrustStandard
		result.HighValueThreshold = thresholds.DefaultEscrowUSD
		if amountUSD >= thresholds.DefaultEscrowUSD {
			result.Warnings = append(result.Warnings, "No corridor config found: default escrow recommended for high-value")
		}
	}

	if result.KYCRequired {
		result.Warnings = append(result.Warnings, "KYC verification required for this amount")
	}

	slog.Info("crowdshipping: compliance checked",
		"corridor", corridorID,
		"amount_usd", amountUSD,
		"allowed", result.Allowed,
		"kyc_required", result.KYCRequired,
		"escrow_recommended", result.EscrowRecommended,
		"boundaries", len(result.Boundaries),
		"warnings", len(result.Warnings),
	)

	return result
}

// ── HTTP handler ──────────────────────────────────────────────────────────────

type complianceRequest struct {
	AmountUSD      float64 `form:"amount" binding:"required,gt=0"`
	ItemCategories string  `form:"categories"`
}

func (h *Handler) GetComplianceAdvisory(c *gin.Context) {
	origin := c.Param("origin")
	dest := c.Param("dest")

	var req complianceRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "amount query parameter is required and must be > 0")
		return
	}

	var categories []string
	if req.ItemCategories != "" {
		for _, cat := range strings.Split(req.ItemCategories, ",") {
			trimmed := strings.TrimSpace(cat)
			if trimmed != "" {
				categories = append(categories, trimmed)
			}
		}
	}

	result := CheckCompliance(origin, dest, req.AmountUSD, categories...)
	response.OK(c, result)
}
