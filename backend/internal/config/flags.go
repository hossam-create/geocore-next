package config

import (
	"os"
	"strconv"
	"sync"
)

// ════════════════════════════════════════════════════════════════════════════
// Feature Flags (Kill Switches)
// Toggle risky features instantly via ENV. All flags default to enabled.
// Set ENV to "false" to disable.
// ════════════════════════════════════════════════════════════════════════════

// FeatureFlags controls which features are active.
type FeatureFlags struct {
	EnableAutoOffers          bool
	EnableDealCloser          bool
	EnableP2PMatching         bool
	EnableDynamicFees         bool
	EnableRetention           bool
	EnableBootstrap           bool
	EnableProhibitedCheck     bool
	EnableAdminLiveControl    bool
	EnableAuctionDeposit      bool
	EnableLiveFomo            bool
	EnableLiveNudges          bool
	EnableSmartBuyNow         bool
	EnablePinnedItems         bool
	EnableQuickBid            bool
	EnableLiveFees            bool
	EnableLiveBoost           bool
	EnablePremiumAuctions     bool
	EnableEntryFee            bool
	EnableDynamicLiveFees     bool
	EnableRevenueFlywheel     bool
	EnablePriorityBid         bool
	EnableCreatorSplit        bool
	EnableBehavioralEngine    bool
	EnableMonetizedNudges     bool
	EnableBidderQualityGate   bool
	EnableAIAssistant         bool
	EnableAIMonetizationHints bool
	EnableAIDropPrevention    bool
	EnableReferrals           bool
	EnableLiveInvites         bool
	EnableStreaks             bool
	EnableGroupBuy            bool
	EnableShareRewards        bool

	// Sprint 16: Creator Economy
	EnableCreators        bool
	EnableCreatorMatching bool
	EnableCreatorBonuses  bool

	// Sprint 17: Marketplace Brain
	EnableMarketplaceBrain  bool
	EnableSmartRanking      bool
	EnableTrafficAllocation bool

	// Sprint 18: Discovery / Search / Category Tree
	EnableSearch                   bool
	EnableAutocomplete             bool
	EnableCategoryTree             bool
	EnableLiveInSearch             bool
	EnableSavedSearch              bool
	EnableSavedSearchNotifications bool // Sprint 18.2: dedicated flag for notification dispatch

	// Sprint 19/20: Community Exchange (VIP)
	EnableP2PExchange     bool // ENABLE_P2P_EXCHANGE — base exchange layer
	EnableExchangeSystem  bool // ENABLE_EXCHANGE_SYSTEM — full VIP exchange product (superset)
	EnableRateHints       bool // ENABLE_RATE_HINTS — advisory rate guidance
	EnableLiquidityEngine bool // ENABLE_LIQUIDITY_ENGINE — pair liquidity tracking
	EnableExchangeRisk    bool // ENABLE_EXCHANGE_RISK — risk/fraud engine for exchange

	// Sprint 20: Private Invite & Referral Network
	EnableInviteOnly     bool // ENABLE_INVITE_ONLY — require invite code at signup
	EnablePrivateNetwork bool // ENABLE_PRIVATE_NETWORK — private member liquidity/matching boost

	// Sprint 21: Waitlist + Hype Engine
	EnableWaitlist          bool // ENABLE_WAITLIST — open pre-launch queue
	EnableWaitlistReferrals bool // ENABLE_WAITLIST_REFERRALS — referral boost logic
	EnablePriorityQueue     bool // ENABLE_PRIORITY_QUEUE — periodic position recalculation

	// Sprint 22: Data Protection & Incident Response
	EnableEmergencyMode bool // ENABLE_EMERGENCY_MODE — blocks all write ops on sensitive paths

	// Sprint 23: Security Monitoring & Admin Observability
	EnableSecurityMonitoring bool // ENABLE_SECURITY_MONITORING — record risk profile + security events
	EnableAutoFreeze         bool // ENABLE_AUTO_FREEZE — auto-freeze users whose risk_score > 70
	EnableSecurityAlerts     bool // ENABLE_SECURITY_ALERTS — send spike / threshold alerts

	// Sprint 25: Red Team Simulation (off by default; admin-only tool).
	EnableRedTeam bool // ENABLE_REDTEAM — allow internal attack simulators to run

	// Cancellation Insurance Engine
	EnableCancellationInsurance bool // ENABLE_CANCELLATION_INSURANCE — opt-in insurance at checkout

	// Travel Guarantee + Protection Engine
	EnableTravelGuarantee bool // ENABLE_TRAVEL_GUARANTEE — full protection system (guarantee + insurance + A/B)

	// Dynamic Insurance Pricing AI
	EnableDynamicPricing bool // ENABLE_DYNAMIC_PRICING — AI-driven per-user insurance pricing

	// Growth Engine
	EnableMessaging      bool // ENABLE_MESSAGING — smart push/email/in-app messaging
	EnableExperiments    bool // ENABLE_EXPERIMENTS — A/B testing + bandit optimization
	EnableBandits        bool // ENABLE_BANDITS — Thompson Sampling / UCB1 for optimization
	EnableDopamineEngine bool // ENABLE_DOPAMINE_ENGINE — dopamine loop tracking + feed adaptation
	EnableReengagement   bool // ENABLE_REENGAGEMENT — churn detection + re-engagement planning
	EnableDecisionEngine bool // ENABLE_DECISION_ENGINE — unified DecideNextBestAction
}

var (
	flags     FeatureFlags
	flagsOnce sync.Once
)

// GetFlags returns the current feature flags (loaded once from ENV).
func GetFlags() FeatureFlags {
	flagsOnce.Do(func() {
		flags = FeatureFlags{
			EnableAutoOffers:          envBool("ENABLE_AUTO_OFFERS", true),
			EnableDealCloser:          envBool("ENABLE_DEAL_CLOSER", true),
			EnableP2PMatching:         envBool("ENABLE_P2P", true),
			EnableDynamicFees:         envBool("ENABLE_DYNAMIC_FEES", true),
			EnableRetention:           envBool("ENABLE_RETENTION", true),
			EnableBootstrap:           envBool("ENABLE_BOOTSTRAP", true),
			EnableProhibitedCheck:     envBool("ENABLE_PROHIBITED_CHECK", true),
			EnableAdminLiveControl:    envBool("ENABLE_ADMIN_LIVE_CONTROL", true),
			EnableAuctionDeposit:      envBool("ENABLE_AUCTION_DEPOSIT", true),
			EnableLiveFomo:            envBool("ENABLE_LIVE_FOMO", true),
			EnableLiveNudges:          envBool("ENABLE_LIVE_NUDGES", true),
			EnableSmartBuyNow:         envBool("ENABLE_SMART_BUY_NOW", true),
			EnablePinnedItems:         envBool("ENABLE_PINNED_ITEMS", true),
			EnableQuickBid:            envBool("ENABLE_QUICK_BID", true),
			EnableLiveFees:            envBool("ENABLE_LIVE_FEES", true),
			EnableLiveBoost:           envBool("ENABLE_LIVE_BOOST", true),
			EnablePremiumAuctions:     envBool("ENABLE_PREMIUM_AUCTIONS", true),
			EnableEntryFee:            envBool("ENABLE_ENTRY_FEE", true),
			EnableDynamicLiveFees:     envBool("ENABLE_DYNAMIC_LIVE_FEES", true),
			EnableRevenueFlywheel:     envBool("ENABLE_REVENUE_FLYWHEEL", true),
			EnablePriorityBid:         envBool("ENABLE_PRIORITY_BID", true),
			EnableCreatorSplit:        envBool("ENABLE_CREATOR_SPLIT", true),
			EnableBehavioralEngine:    envBool("ENABLE_BEHAVIORAL_ENGINE", true),
			EnableAIAssistant:         envBool("ENABLE_AI_ASSISTANT", true),
			EnableAIMonetizationHints: envBool("ENABLE_AI_MONETIZATION_HINTS", true),
			EnableAIDropPrevention:    envBool("ENABLE_AI_DROP_PREVENTION", true),
			EnableMonetizedNudges:     envBool("ENABLE_MONETIZED_NUDGES", true),
			EnableBidderQualityGate:   envBool("ENABLE_BIDDER_QUALITY_GATE", true),
			EnableReferrals:           envBool("ENABLE_REFERRALS", true),
			EnableLiveInvites:         envBool("ENABLE_LIVE_INVITES", true),
			EnableStreaks:             envBool("ENABLE_STREAKS", true),
			EnableGroupBuy:            envBool("ENABLE_GROUP_BUY", true),
			EnableShareRewards:        envBool("ENABLE_SHARE_REWARDS", true),

			// Sprint 16: Creator Economy
			EnableCreators:        envBool("ENABLE_CREATORS", true),
			EnableCreatorMatching: envBool("ENABLE_CREATOR_MATCHING", true),
			EnableCreatorBonuses:  envBool("ENABLE_CREATOR_BONUSES", true),

			// Sprint 17: Marketplace Brain
			EnableMarketplaceBrain:  envBool("ENABLE_MARKETPLACE_BRAIN", true),
			EnableSmartRanking:      envBool("ENABLE_SMART_RANKING", true),
			EnableTrafficAllocation: envBool("ENABLE_TRAFFIC_ALLOCATION", true),

			// Sprint 18: Discovery
			EnableSearch:                   envBool("ENABLE_SEARCH", true),
			EnableAutocomplete:             envBool("ENABLE_AUTOCOMPLETE", true),
			EnableCategoryTree:             envBool("ENABLE_CATEGORY_TREE", true),
			EnableLiveInSearch:             envBool("ENABLE_LIVE_IN_SEARCH", true),
			EnableSavedSearch:              envBool("ENABLE_SAVED_SEARCH", true),
			EnableSavedSearchNotifications: envBool("ENABLE_SAVED_SEARCH_NOTIFICATIONS", true),

			// Sprint 19/20: Community Exchange (VIP) — defaults false (opt-in)
			EnableP2PExchange:     envBool("ENABLE_P2P_EXCHANGE", false),
			EnableExchangeSystem:  envBool("ENABLE_EXCHANGE_SYSTEM", false),
			EnableRateHints:       envBool("ENABLE_RATE_HINTS", false),
			EnableLiquidityEngine: envBool("ENABLE_LIQUIDITY_ENGINE", false),
			EnableExchangeRisk:    envBool("ENABLE_EXCHANGE_RISK", false),

			// Sprint 20: Private Invite & Referral Network
			EnableInviteOnly:     envBool("ENABLE_INVITE_ONLY", false),
			EnablePrivateNetwork: envBool("ENABLE_PRIVATE_NETWORK", false),

			// Sprint 21: Waitlist + Hype Engine
			EnableWaitlist:          envBool("ENABLE_WAITLIST", false),
			EnableWaitlistReferrals: envBool("ENABLE_WAITLIST_REFERRALS", true),
			EnablePriorityQueue:     envBool("ENABLE_PRIORITY_QUEUE", true),

			// Sprint 22: Data Protection & Incident Response
			EnableEmergencyMode: envBool("ENABLE_EMERGENCY_MODE", false),

			// Sprint 23: Security Monitoring & Admin Observability
			EnableSecurityMonitoring: envBool("ENABLE_SECURITY_MONITORING", true),
			EnableAutoFreeze:         envBool("ENABLE_AUTO_FREEZE", true),
			EnableSecurityAlerts:     envBool("ENABLE_SECURITY_ALERTS", true),

			// Sprint 25: Red Team Simulation — OFF by default in prod.
			EnableRedTeam: envBool("ENABLE_REDTEAM", false),

			// Cancellation Insurance Engine
			EnableCancellationInsurance: envBool("ENABLE_CANCELLATION_INSURANCE", true),

			// Travel Guarantee + Protection Engine
			EnableTravelGuarantee: envBool("ENABLE_TRAVEL_GUARANTEE", true),

			// Dynamic Insurance Pricing AI
			EnableDynamicPricing: envBool("ENABLE_DYNAMIC_PRICING", true),

			// Growth Engine
			EnableMessaging:      envBool("ENABLE_MESSAGING", true),
			EnableExperiments:    envBool("ENABLE_EXPERIMENTS", true),
			EnableBandits:        envBool("ENABLE_BANDITS", true),
			EnableDopamineEngine: envBool("ENABLE_DOPAMINE_ENGINE", true),
			EnableReengagement:   envBool("ENABLE_REENGAGEMENT", true),
			EnableDecisionEngine: envBool("ENABLE_DECISION_ENGINE", true),
		}
	})
	return flags
}

// ReloadFlags forces a re-read of ENV (for runtime flag changes).
func ReloadFlags() FeatureFlags {
	flagsOnce = sync.Once{}
	return GetFlags()
}

// SetFlag overrides a flag at runtime (for admin API).
func SetFlag(name string, value bool) {
	f := GetFlags()
	switch name {
	case "auto_offers":
		f.EnableAutoOffers = value
	case "deal_closer":
		f.EnableDealCloser = value
	case "p2p_matching":
		f.EnableP2PMatching = value
	case "dynamic_fees":
		f.EnableDynamicFees = value
	case "retention":
		f.EnableRetention = value
	case "bootstrap":
		f.EnableBootstrap = value
	case "prohibited_check":
		f.EnableProhibitedCheck = value
	case "admin_live_control":
		f.EnableAdminLiveControl = value
	case "auction_deposit":
		f.EnableAuctionDeposit = value
	case "live_fomo":
		f.EnableLiveFomo = value
	case "live_nudges":
		f.EnableLiveNudges = value
	case "smart_buy_now":
		f.EnableSmartBuyNow = value
	case "pinned_items":
		f.EnablePinnedItems = value
	case "quick_bid":
		f.EnableQuickBid = value
	case "live_fees":
		f.EnableLiveFees = value
	case "live_boost":
		f.EnableLiveBoost = value
	case "premium_auctions":
		f.EnablePremiumAuctions = value
	case "entry_fee":
		f.EnableEntryFee = value
	case "dynamic_live_fees":
		f.EnableDynamicLiveFees = value
	case "revenue_flywheel":
		f.EnableRevenueFlywheel = value
	case "priority_bid":
		f.EnablePriorityBid = value
	case "creator_split":
		f.EnableCreatorSplit = value
	case "behavioral_engine":
		f.EnableBehavioralEngine = value
	case "monetized_nudges":
		f.EnableMonetizedNudges = value
	case "bidder_quality_gate":
		f.EnableBidderQualityGate = value
	case "ai_assistant":
		f.EnableAIAssistant = value
	case "ai_monetization_hints":
		f.EnableAIMonetizationHints = value
	case "ai_drop_prevention":
		f.EnableAIDropPrevention = value
	case "referrals":
		f.EnableReferrals = value
	case "live_invites":
		f.EnableLiveInvites = value
	case "streaks":
		f.EnableStreaks = value
	case "group_buy":
		f.EnableGroupBuy = value
	case "share_rewards":
		f.EnableShareRewards = value

	// Sprint 16: Creator Economy
	case "creators":
		f.EnableCreators = value
	case "creator_matching":
		f.EnableCreatorMatching = value
	case "creator_bonuses":
		f.EnableCreatorBonuses = value

	// Sprint 17: Marketplace Brain
	case "marketplace_brain":
		f.EnableMarketplaceBrain = value
	case "smart_ranking":
		f.EnableSmartRanking = value
	case "traffic_allocation":
		f.EnableTrafficAllocation = value

	// Sprint 18.2: Saved search notifications
	case "saved_search_notifications":
		f.EnableSavedSearchNotifications = value

	// Sprint 19/20: Community Exchange
	case "p2p_exchange":
		f.EnableP2PExchange = value
	case "exchange_system":
		f.EnableExchangeSystem = value
	case "rate_hints":
		f.EnableRateHints = value
	case "liquidity_engine":
		f.EnableLiquidityEngine = value
	case "exchange_risk":
		f.EnableExchangeRisk = value

	// Sprint 20: Private Invite & Referral Network
	case "invite_only":
		f.EnableInviteOnly = value
	case "private_network":
		f.EnablePrivateNetwork = value

	// Sprint 21: Waitlist + Hype Engine
	case "waitlist":
		f.EnableWaitlist = value
	case "waitlist_referrals":
		f.EnableWaitlistReferrals = value
	case "priority_queue":
		f.EnablePriorityQueue = value

	// Sprint 22: Data Protection & Incident Response
	case "emergency_mode":
		f.EnableEmergencyMode = value

	// Sprint 23: Security Monitoring & Admin Observability
	case "security_monitoring":
		f.EnableSecurityMonitoring = value
	case "auto_freeze":
		f.EnableAutoFreeze = value
	case "security_alerts":
		f.EnableSecurityAlerts = value

	// Sprint 25: Red Team Simulation
	case "redteam":
		f.EnableRedTeam = value

	// Cancellation Insurance Engine
	case "cancellation_insurance":
		f.EnableCancellationInsurance = value

	// Travel Guarantee + Protection Engine
	case "travel_guarantee":
		f.EnableTravelGuarantee = value

	// Dynamic Insurance Pricing AI
	case "dynamic_pricing":
		f.EnableDynamicPricing = value

	// Growth Engine
	case "messaging":
		f.EnableMessaging = value
	case "experiments":
		f.EnableExperiments = value
	case "bandits":
		f.EnableBandits = value
	case "dopamine_engine":
		f.EnableDopamineEngine = value
	case "reengagement":
		f.EnableReengagement = value
	case "decision_engine":
		f.EnableDecisionEngine = value
	}
	flags = f
}

func envBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return b
}
