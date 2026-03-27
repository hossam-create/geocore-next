import { Router, type IRouter } from "express";

const router: IRouter = Router();

const CATEGORY_MULTIPLIERS: Record<string, number> = {
  electronics: 1.15,
  vehicles: 1.08,
  real_estate: 1.05,
  fashion: 1.2,
  jewelry: 1.25,
  watches: 1.3,
  art: 1.18,
  antiques: 1.22,
  sports: 1.12,
  home: 1.1,
  default: 1.12,
};

const STRATEGIES = {
  conservative: { aggression: 0.3, maxPremium: 1.05 },
  balanced: { aggression: 0.6, maxPremium: 1.15 },
  aggressive: { aggression: 0.9, maxPremium: 1.3 },
} as const;

function sigmoid(x: number): number {
  return 1 / (1 + Math.exp(-x));
}

function roundToClean(amount: number): number {
  if (amount >= 10000) return Math.round(amount / 500) * 500;
  if (amount >= 1000) return Math.round(amount / 50) * 50;
  if (amount >= 100) return Math.round(amount / 5) * 5;
  return Math.round(amount);
}

interface PredictInput {
  current_price: number;
  starting_price: number;
  reserve_price?: number;
  estimated_value?: number;
  seconds_remaining: number;
  total_seconds: number;
  bid_count: number;
  category: string;
  strategy?: "conservative" | "balanced" | "aggressive";
  currency?: string;
}

function predict(input: PredictInput) {
  const {
    current_price,
    starting_price,
    reserve_price,
    estimated_value,
    seconds_remaining,
    total_seconds,
    bid_count,
    category,
    strategy = "balanced",
    currency = "AED",
  } = input;

  const catMult = CATEGORY_MULTIPLIERS[category.toLowerCase()] ?? CATEGORY_MULTIPLIERS.default;
  const strat = STRATEGIES[strategy] ?? STRATEGIES.balanced;

  const priceGrowth = current_price / Math.max(starting_price, 1);
  const marketValue =
    estimated_value && estimated_value > 0
      ? estimated_value
      : current_price * catMult * (1 + Math.log(Math.max(priceGrowth, 1)) * 0.1);

  const timeRatio = seconds_remaining / Math.max(total_seconds, 1);
  let urgency = 1.0 - timeRatio;
  if (timeRatio < 0.1) urgency = Math.min(urgency * 1.5, 1.0);

  const bidsPerHour = (bid_count / Math.max(total_seconds - seconds_remaining, 1)) * 3600;
  const competition = Math.min(bidsPerHour / 20.0, 1.0);

  const baseIncrPct = 0.02;
  const totalIncrPct = baseIncrPct + urgency * 0.03 * strat.aggression + competition * 0.02 * strat.aggression;
  let optimalBid = roundToClean(current_price * (1 + totalIncrPct));
  let maxBid = roundToClean(marketValue * strat.maxPremium);

  const valueHeadroom = (marketValue - current_price) / Math.max(marketValue, 1);
  const winProb = sigmoid(valueHeadroom * 3 - urgency * 2 + strat.aggression * 1.5);

  let shouldBid = optimalBid <= maxBid && winProb > 0.3 - strat.aggression * 0.2;
  if (reserve_price && current_price < reserve_price) {
    optimalBid = Math.max(optimalBid, Math.round(reserve_price * 1.01));
    shouldBid = true;
  }

  const explanationParts: string[] = [];
  if (urgency > 0.8) explanationParts.push("auction ending soon");
  else if (urgency > 0.5) explanationParts.push("mid-auction phase");
  if (competition > 0.6) explanationParts.push("high competition detected");
  explanationParts.push(`${strategy} strategy applied`);
  explanationParts.push(`${Math.round(winProb * 100)}% estimated win probability`);

  return {
    should_bid: shouldBid,
    optimal_bid: optimalBid,
    max_bid: maxBid,
    min_increment: Math.round(current_price * 0.01 * 100) / 100,
    estimated_market_value: Math.round(marketValue * 100) / 100,
    win_probability: Math.round(winProb * 1000) / 1000,
    urgency_score: Math.round(urgency * 1000) / 1000,
    competition_score: Math.round(competition * 1000) / 1000,
    strategy,
    explanation: `${shouldBid ? "Recommend bidding" : "Hold — price near ceiling"}. ${explanationParts.join("; ")}.`,
    currency,
  };
}

// ── GET /api/v1/ai/health ─────────────────────────────────────────────────────
router.get("/v1/ai/health", (_req, res) => {
  res.json({ status: "ok", service: "geocore-ai-pricing" });
});

// ── GET /api/v1/ai/strategies ─────────────────────────────────────────────────
router.get("/v1/ai/strategies", (_req, res) => {
  res.json({
    strategies: [
      { id: "conservative", name: "Conservative", description: "Low risk, stay well below market value" },
      { id: "balanced", name: "Balanced", description: "Optimal risk/reward for most auctions" },
      { id: "aggressive", name: "Aggressive", description: "Win at all costs, up to 30% premium" },
    ],
  });
});

// ── POST /api/v1/ai/predict ───────────────────────────────────────────────────
router.post("/v1/ai/predict", (req, res) => {
  const body = req.body as PredictInput;
  const required = ["current_price", "starting_price", "seconds_remaining", "total_seconds", "bid_count", "category"];
  const missing = required.filter((k) => body[k as keyof PredictInput] === undefined);
  if (missing.length > 0) {
    res.status(400).json({ error: `Missing fields: ${missing.join(", ")}` });
    return;
  }
  try {
    const result = predict(body);
    res.json({ success: true, data: result });
  } catch (e) {
    res.status(500).json({ error: String(e) });
  }
});

// ── GET /api/v1/ai/categories ────────────────────────────────────────────────
router.get("/v1/ai/categories", (_req, res) => {
  res.json({ multipliers: CATEGORY_MULTIPLIERS });
});

export default router;
