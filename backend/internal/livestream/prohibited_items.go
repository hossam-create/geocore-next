package livestream

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/geocore-next/backend/internal/freeze"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Prohibited Items Enforcement (Sprint 10)
//
// Prevents illegal/restricted goods from being sold in live auctions.
// Feature-flagged via ENABLE_PROHIBITED_CHECK env var (default: true).
// ════════════════════════════════════════════════════════════════════════════

// ComplianceResult is the output of CheckItemCompliance.
type ComplianceResult struct {
	Verdict    ComplianceVerdict `json:"verdict"`     // allowed / flagged / blocked
	ReasonCode string            `json:"reason_code"` // e.g. "weapons_keyword", "drugs_category"
	Confidence ConfidenceLevel   `json:"confidence"`  // high / medium / low
	MatchedOn  string            `json:"matched_on"`  // the keyword or category that triggered
	RiskScore  int               `json:"risk_score"`  // 0–100 semantic risk score
}

type ComplianceVerdict string

const (
	VerdictAllowed ComplianceVerdict = "allowed"
	VerdictFlagged ComplianceVerdict = "flagged" // requires admin review
	VerdictBlocked ComplianceVerdict = "blocked" // creation denied
)

type ConfidenceLevel string

const (
	ConfidenceHigh   ConfidenceLevel = "high"   // → block
	ConfidenceMedium ConfidenceLevel = "medium" // → flag for review
	ConfidenceLow    ConfidenceLevel = "low"    // → allow
)

// ── Prohibited Categories ────────────────────────────────────────────────────

var ProhibitedCategories = map[string]ConfidenceLevel{
	// Weapons
	"weapons":       ConfidenceHigh,
	"firearms":      ConfidenceHigh,
	"ammunition":    ConfidenceHigh,
	"explosives":    ConfidenceHigh,
	"knives_combat": ConfidenceHigh,
	"tasers":        ConfidenceHigh,
	"pepper_spray":  ConfidenceMedium,

	// Drugs
	"drugs":              ConfidenceHigh,
	"narcotics":          ConfidenceHigh,
	"controlled_subs":    ConfidenceHigh,
	"prescription_drugs": ConfidenceHigh,
	"steroids":           ConfidenceHigh,
	"cbd_products":       ConfidenceMedium,

	// Illegal electronics
	"jamming_devices":  ConfidenceHigh,
	"surveillance_spy": ConfidenceHigh,
	"counterfeit_tech": ConfidenceHigh,
	"radar_detectors":  ConfidenceMedium,

	// Restricted imports
	"endangered_species": ConfidenceHigh,
	"ivory":              ConfidenceHigh,
	"counterfeit_goods":  ConfidenceHigh,
	"unlicensed_media":   ConfidenceMedium,
}

// ── Prohibited Keywords (multi-language) ─────────────────────────────────────

var ProhibitedKeywords = map[string]ConfidenceLevel{
	// English
	"gun":           ConfidenceHigh,
	"rifle":         ConfidenceHigh,
	"pistol":        ConfidenceHigh,
	"shotgun":       ConfidenceHigh,
	"cocaine":       ConfidenceHigh,
	"heroin":        ConfidenceHigh,
	"meth":          ConfidenceHigh,
	"fentanyl":      ConfidenceHigh,
	"bomb":          ConfidenceHigh,
	"grenade":       ConfidenceHigh,
	"counterfeit":   ConfidenceMedium,
	"fake":          ConfidenceMedium,
	"replica brand": ConfidenceMedium,
	"stolen":        ConfidenceHigh,
	"hack":          ConfidenceMedium,
	"cheat":         ConfidenceMedium,
	"exploit":       ConfidenceMedium,

	// Arabic
	"سلاح":    ConfidenceHigh,
	"مسدس":    ConfidenceHigh,
	"بندقية":  ConfidenceHigh,
	"مخدرات":  ConfidenceHigh,
	"حشيش":    ConfidenceHigh,
	"ترياق":   ConfidenceHigh,
	"متفجرات": ConfidenceHigh,
	"قنبلة":   ConfidenceHigh,
	"مزيف":    ConfidenceMedium,
	"مقلد":    ConfidenceMedium,
	"مسروق":   ConfidenceHigh,

	// French
	"arme":        ConfidenceHigh,
	"pistolet":    ConfidenceHigh,
	"drogue":      ConfidenceHigh,
	"stupéfiant":  ConfidenceHigh,
	"contrefaçon": ConfidenceMedium,
	"explosif":    ConfidenceHigh,
}

// IsProhibitedCheckEnabled returns true unless explicitly disabled via env.
func IsProhibitedCheckEnabled() bool {
	val := os.Getenv("ENABLE_PROHIBITED_CHECK")
	if val == "" {
		return true // default enabled
	}
	return val != "false" && val != "0"
}

// CheckItemCompliance checks an item's title, description, and category
// against prohibited categories and keywords.
func CheckItemCompliance(title, description, category string) ComplianceResult {
	if !IsProhibitedCheckEnabled() {
		return ComplianceResult{Verdict: VerdictAllowed, Confidence: ConfidenceLow}
	}

	titleLower := strings.ToLower(title)
	descLower := strings.ToLower(description)
	catLower := strings.ToLower(category)
	combined := titleLower + " " + descLower + " " + catLower

	// Check categories first (higher signal)
	if level, ok := ProhibitedCategories[catLower]; ok {
		verdict := verdictFromConfidence(level)
		return ComplianceResult{
			Verdict:    verdict,
			ReasonCode: "prohibited_category:" + catLower,
			Confidence: level,
			MatchedOn:  catLower,
		}
	}

	// Check keywords
	bestResult := ComplianceResult{Verdict: VerdictAllowed, Confidence: ConfidenceLow}
	for keyword, level := range ProhibitedKeywords {
		if strings.Contains(combined, keyword) {
			verdict := verdictFromConfidence(level)
			// Keep the highest-confidence match
			if level == ConfidenceHigh || bestResult.Confidence != ConfidenceHigh {
				bestResult = ComplianceResult{
					Verdict:    verdict,
					ReasonCode: "prohibited_keyword:" + keyword,
					Confidence: level,
					MatchedOn:  keyword,
				}
				if level == ConfidenceHigh {
					return bestResult // short-circuit on high confidence
				}
			}
		}
	}

	return bestResult
}

// EnforceCompliance is the full enforcement pipeline:
//   - BLOCKED → deny creation, freeze user, create fraud alert, log audit
//   - FLAGGED → allow creation but mark requires_review=true, log audit
//   - ALLOWED → normal flow
func EnforceCompliance(db *gorm.DB, userID uuid.UUID, title, description, category string) ComplianceResult {
	result := CheckItemCompliance(title, description, category)

	// Compute semantic risk score and overlay verdict
	riskScore := ClassifyItemRisk(title, description, category)
	result.RiskScore = riskScore

	// If risk score verdict is stricter than keyword verdict, upgrade
	riskVerdict := RiskScoreToVerdict(riskScore)
	if riskVerdict == VerdictBlocked && result.Verdict != VerdictBlocked {
		result.Verdict = VerdictBlocked
		result.ReasonCode = "risk_score_blocked:" + fmt.Sprintf("score=%d", riskScore)
		result.Confidence = ConfidenceHigh
	} else if riskVerdict == VerdictFlagged && result.Verdict == VerdictAllowed {
		result.Verdict = VerdictFlagged
		result.ReasonCode = "risk_score_flagged:" + fmt.Sprintf("score=%d", riskScore)
		result.Confidence = ConfidenceMedium
	}

	switch result.Verdict {
	case VerdictBlocked:
		slog.Warn("prohibited-items: BLOCKED item creation",
			"user_id", userID, "title", title, "reason", result.ReasonCode)

		// Freeze user temporarily
		if err := freeze.FreezeUser(db, userID, uuid.Nil, "prohibited_item_attempt:"+result.ReasonCode); err != nil {
			slog.Error("prohibited-items: failed to freeze user", "user_id", userID, "error", err)
		}

		// Log audit
		freeze.LogAudit(db, "prohibited_item_blocked", userID, uuid.Nil,
			"title="+title+" reason="+result.ReasonCode)

	case VerdictFlagged:
		slog.Info("prohibited-items: FLAGGED item for review",
			"user_id", userID, "title", title, "reason", result.ReasonCode)

		// Log audit
		freeze.LogAudit(db, "prohibited_item_flagged", userID, uuid.Nil,
			"title="+title+" reason="+result.ReasonCode)
	}

	return result
}

// ════════════════════════════════════════════════════════════════════════════
// Semantic Risk Classification (Sprint 10 Upgrade)
//
// ClassifyItemRisk produces a 0–100 risk score using rule-based scoring.
// Thresholds: ≥80 → BLOCK, 50–79 → REVIEW, <50 → ALLOW
// ════════════════════════════════════════════════════════════════════════════

// RiskScoreThresholds
const (
	RiskScoreBlock  = 80 // ≥80 → BLOCK
	RiskScoreReview = 50 // 50–79 → REVIEW
)

// SuspiciousPatterns are regex-free heuristics for evasive listings.
var SuspiciousPatterns = []struct {
	Pattern string
	Score   int
}{
	// Evasion signals — deliberate obfuscation
	{"special herbs", 60},
	{"herbal supplement", 30},
	{"research chemical", 70},
	{"not for human consumption", 75},
	{"collector item only", 55},
	{"decorative only", 40},
	{"replica", 45},
	{"unlocked", 25},
	{"jailbroken", 35},
	{"no questions asked", 65},
	{"discrete shipping", 55},
	{"private listing", 40},
	{"for parts", 20},
	{"as is", 15},
	// Arabic evasion
	{"أعشاب خاصة", 60},
	{"شحن سري", 55},
	// French evasion
	{"herbes spéciales", 60},
	{"expédition discrète", 55},
}

// CategoryRiskScores map categories to base risk scores.
var CategoryRiskScores = map[string]int{
	"electronics":  10,
	"fashion":      5,
	"home":         5,
	"vehicles":     15,
	"collectibles": 20,
	"health":       25,
	"supplements":  35,
}

// ClassifyItemRisk produces a 0–100 risk score using multi-signal analysis.
// Signals: prohibited keywords, category, suspicious patterns, text anomalies.
func ClassifyItemRisk(title, description, category string) int {
	if !IsProhibitedCheckEnabled() {
		return 0
	}

	score := 0
	titleLower := strings.ToLower(title)
	descLower := strings.ToLower(description)
	catLower := strings.ToLower(category)
	combined := titleLower + " " + descLower + " " + catLower

	// Signal 1: Prohibited category match (highest weight)
	if level, ok := ProhibitedCategories[catLower]; ok {
		switch level {
		case ConfidenceHigh:
			score += 85
		case ConfidenceMedium:
			score += 55
		}
	}

	// Signal 2: Prohibited keyword match
	for keyword, level := range ProhibitedKeywords {
		if strings.Contains(combined, keyword) {
			switch level {
			case ConfidenceHigh:
				score += 40
			case ConfidenceMedium:
				score += 20
			}
		}
	}

	// Signal 3: Suspicious pattern match
	for _, sp := range SuspiciousPatterns {
		if strings.Contains(combined, sp.Pattern) {
			score += sp.Score
		}
	}

	// Signal 4: Category base risk
	if base, ok := CategoryRiskScores[catLower]; ok {
		score += base
	}

	// Signal 5: Text anomaly detection
	// Very short title + long description = potential evasion
	if utf8.RuneCountInString(title) < 5 && utf8.RuneCountInString(description) > 100 {
		score += 15
	}
	// Excessive special characters (obfuscation)
	specialCount := 0
	for _, r := range combined {
		if r == '🔥' || r == '💊' || r == '🔫' || r == '💣' || r == '⚡' {
			specialCount++
		}
	}
	if specialCount > 0 {
		score += specialCount * 10
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}
	return score
}

// RiskScoreToVerdict converts a risk score to a compliance verdict.
func RiskScoreToVerdict(score int) ComplianceVerdict {
	if score >= RiskScoreBlock {
		return VerdictBlocked
	}
	if score >= RiskScoreReview {
		return VerdictFlagged
	}
	return VerdictAllowed
}

func verdictFromConfidence(level ConfidenceLevel) ComplianceVerdict {
	switch level {
	case ConfidenceHigh:
		return VerdictBlocked
	case ConfidenceMedium:
		return VerdictFlagged
	default:
		return VerdictAllowed
	}
}
