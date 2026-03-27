import { Router } from "express";
import OpenAI from "openai";

const router = Router();

// OpenAI via Replit AI Integration proxy
const openai = new OpenAI({
  apiKey: process.env.AI_INTEGRATIONS_OPENAI_API_KEY ?? process.env.OPENAI_API_KEY ?? "no-key",
  baseURL: process.env.AI_INTEGRATIONS_OPENAI_BASE_URL ?? "https://api.openai.com/v1",
});

// ── GCC Marketplace listings catalog ─────────────────────────────────────────
const CATALOG = [
  { id:"lst_001", title:"iPhone 15 Pro Max 256GB - Like New", price:4200, currency:"AED", category:"Electronics", location:"Dubai, UAE", condition:"Like New", image:"https://picsum.photos/seed/iphone15/400/300", seller:"Ahmed Al Mansoori", rating:4.9 },
  { id:"lst_002", title:"Samsung Galaxy S24 Ultra - Black", price:3800, currency:"AED", category:"Electronics", location:"Abu Dhabi, UAE", condition:"New", image:"https://picsum.photos/seed/samsung/400/300", seller:"Sara Tech Store", rating:4.7 },
  { id:"lst_003", title:"MacBook Pro M3 14-inch 512GB", price:8500, currency:"AED", category:"Electronics", location:"Dubai, UAE", condition:"New", image:"https://picsum.photos/seed/macbook/400/300", seller:"iStore Dubai", rating:5.0 },
  { id:"lst_004", title:"PS5 Console + 2 Controllers", price:2100, currency:"AED", category:"Electronics", location:"Sharjah, UAE", condition:"Good", image:"https://picsum.photos/seed/ps5/400/300", seller:"GameZone", rating:4.6 },
  { id:"lst_005", title:"Toyota Land Cruiser 2024 GXR", price:320000, currency:"AED", category:"Vehicles", location:"Dubai, UAE", condition:"New", image:"https://picsum.photos/seed/landcruiser/400/300", seller:"Premier Motors", rating:4.8 },
  { id:"lst_006", title:"Rolex Submariner - Stainless Steel", price:45000, currency:"AED", category:"Watches", location:"Dubai, UAE", condition:"Like New", image:"https://picsum.photos/seed/rolex/400/300", seller:"Luxury Timepieces", rating:4.9 },
  { id:"lst_007", title:"شقة فاخرة في دبي مارينا - 2 غرف", price:120000, currency:"AED", category:"Real Estate", location:"Dubai Marina, UAE", condition:"Like New", image:"https://picsum.photos/seed/apartment/400/300", seller:"Prime Properties", rating:4.7 },
  { id:"lst_008", title:"DJI Mavic 3 Pro Drone - Full Kit", price:5800, currency:"AED", category:"Electronics", location:"Abu Dhabi, UAE", condition:"New", image:"https://picsum.photos/seed/drone/400/300", seller:"Fly Tech UAE", rating:4.8 },
  { id:"lst_009", title:"iPhone 14 Pro 128GB Space Black", price:3200, currency:"AED", category:"Electronics", location:"Dubai, UAE", condition:"Good", image:"https://picsum.photos/seed/iphone14/400/300", seller:"Mobile World", rating:4.5 },
  { id:"lst_010", title:"Nike Air Jordan 1 Retro High OG", price:850, currency:"AED", category:"Clothing", location:"Dubai, UAE", condition:"New", image:"https://picsum.photos/seed/jordan/400/300", seller:"Sneaker Lab", rating:4.6 },
  { id:"lst_011", title:"iPad Pro 12.9 M2 256GB WiFi", price:4600, currency:"AED", category:"Electronics", location:"Abu Dhabi, UAE", condition:"New", image:"https://picsum.photos/seed/ipad/400/300", seller:"iStore Abu Dhabi", rating:4.9 },
  { id:"lst_012", title:"Sony PlayStation VR2 Headset", price:1800, currency:"AED", category:"Electronics", location:"Sharjah, UAE", condition:"Like New", image:"https://picsum.photos/seed/psvr/400/300", seller:"GameZone UAE", rating:4.4 },
  { id:"lst_013", title:"Porsche 911 Carrera S 2023", price:680000, currency:"AED", category:"Vehicles", location:"Dubai, UAE", condition:"Like New", image:"https://picsum.photos/seed/porsche/400/300", seller:"Prestige Auto", rating:5.0 },
  { id:"lst_014", title:"Rolex Daytona Chronograph", price:95000, currency:"AED", category:"Watches", location:"Abu Dhabi, UAE", condition:"New", image:"https://picsum.photos/seed/daytona/400/300", seller:"Watch House", rating:4.9 },
  { id:"lst_015", title:"Hermès Birkin Bag 30cm - Gold", price:85000, currency:"AED", category:"Fashion", location:"Dubai, UAE", condition:"Like New", image:"https://picsum.photos/seed/hermes/400/300", seller:"Luxury Closet", rating:4.8 },
  { id:"lst_016", title:"Villa in Palm Jumeirah - 5 BR", price:8500000, currency:"AED", category:"Real Estate", location:"Palm Jumeirah, UAE", condition:"New", image:"https://picsum.photos/seed/villa/400/300", seller:"Emaar Properties", rating:5.0 },
  { id:"lst_017", title:"Apple Watch Ultra 2 - Titanium", price:3200, currency:"AED", category:"Electronics", location:"Dubai, UAE", condition:"New", image:"https://picsum.photos/seed/applewatch/400/300", seller:"iStore Dubai", rating:4.8 },
  { id:"lst_018", title:"Samsung 85\" QLED 8K Smart TV", price:12000, currency:"AED", category:"Electronics", location:"Sharjah, UAE", condition:"New", image:"https://picsum.photos/seed/samsung_tv/400/300", seller:"Carrefour UAE", rating:4.7 },
  { id:"lst_019", title:"Yamaha YZF-R1 2024 - Sport Bike", price:85000, currency:"AED", category:"Vehicles", location:"Dubai, UAE", condition:"New", image:"https://picsum.photos/seed/yamaha/400/300", seller:"Desert Bikes", rating:4.6 },
  { id:"lst_020", title:"Louis Vuitton Neverfull MM - Monogram", price:8500, currency:"AED", category:"Fashion", location:"Dubai, UAE", condition:"Good", image:"https://picsum.photos/seed/lv/400/300", seller:"Luxury Resale", rating:4.5 },
];

// ── POST /api/v1/ai/recommend ─────────────────────────────────────────────────
// Body: { user_context: { viewed_categories, search_history, price_range, location }, limit, exclude_ids }

router.post("/", async (req, res) => {
  const {
    user_context = {},
    limit = 8,
    exclude_ids = [],
  } = req.body as {
    user_context?: {
      viewed_categories?: Record<string, number>;
      search_history?: string[];
      price_range?: { min: number; max: number };
      location?: string;
    };
    limit?: number;
    exclude_ids?: string[];
  };

  const safeLimit = Math.min(limit, 20);
  const pool = CATALOG.filter(l => !exclude_ids.includes(l.id));

  try {
    // ── AI-enhanced ranking ────────────────────────────────────────────────────
    const contextStr = JSON.stringify({
      viewed_categories: user_context.viewed_categories || {},
      search_history: (user_context.search_history || []).slice(0, 5),
      price_range: user_context.price_range || { min: 0, max: 1_000_000 },
      location: user_context.location || "UAE",
    });

    const catalogSummary = pool
      .slice(0, 20)
      .map(l => `${l.id}|${l.category}|${l.price}|${l.location}|${l.rating}|${l.title}`)
      .join("\n");

    const prompt = `You are a GCC marketplace recommendation AI. Rank these listings for this user.

User Context: ${contextStr}

Listings (id|category|price|location|rating|title):
${catalogSummary}

Return a JSON array of up to ${safeLimit} objects:
[{"id":"lst_xxx","reason":"why this listing suits the user (1 sentence)","tag":"Top Pick|Near You|Similar Category|Price Match|Trending|New Arrival"}]

Prioritize: category matches, price range match, location preference, high rating.
Return ONLY the JSON array, no extra text.`;

    const completion = await openai.chat.completions.create({
      model: "gpt-4o-mini",
      messages: [{ role: "user", content: prompt }],
      max_tokens: 600,
      temperature: 0.3,
    });

    const raw = completion.choices[0]?.message?.content?.trim() || "[]";
    const jsonStr = raw.match(/\[[\s\S]*\]/)?.[0] || "[]";
    const ranked = JSON.parse(jsonStr) as Array<{ id: string; reason: string; tag: string }>;

    const orderedIds = ranked.map(r => r.id);
    const reasonMap: Record<string, { reason: string; tag: string }> = {};
    ranked.forEach(r => { reasonMap[r.id] = { reason: r.reason, tag: r.tag }; });

    const results = orderedIds
      .map(id => {
        const listing = pool.find(l => l.id === id);
        if (!listing) return null;
        return {
          ...listing,
          ai_reason: reasonMap[id]?.reason || "Recommended for you",
          reason_tag: reasonMap[id]?.tag || "Trending",
        };
      })
      .filter(Boolean)
      .slice(0, safeLimit);

    return res.json({
      success: true,
      data: { recommendations: results, ai_powered: true, count: results.length },
    });

  } catch {
    // ── Fallback: rule-based ranking ──────────────────────────────────────────
    const prefs = user_context.viewed_categories || {};
    const priceRange = user_context.price_range || { min: 0, max: 10_000_000 };

    const scored = pool.map(l => {
      let score = l.rating * 10;
      score += (prefs[l.category] || 0) * 20;
      if (l.price >= priceRange.min && l.price <= priceRange.max) score += 15;
      return { ...l, score, ai_reason: "Recommended based on your activity", reason_tag: "Trending" };
    });

    const fallback = scored.sort((a, b) => b.score - a.score).slice(0, safeLimit);

    return res.json({
      success: true,
      data: { recommendations: fallback, ai_powered: false, count: fallback.length },
    });
  }
});

// ── POST /api/v1/ai/recommend/similar ────────────────────────────────────────
// Body: { listing_id, limit }

router.post("/similar", async (req, res) => {
  const { listing_id, limit = 4 } = req.body as { listing_id?: string; limit?: number };

  if (!listing_id) {
    return res.status(400).json({ success: false, message: "listing_id required" });
  }

  const anchor = CATALOG.find(l => l.id === listing_id);
  if (!anchor) {
    return res.status(404).json({ success: false, message: "Listing not found" });
  }

  // Simple content-based filtering: same category first, then price proximity
  const similar = CATALOG
    .filter(l => l.id !== listing_id)
    .map(l => ({
      ...l,
      score:
        (l.category === anchor.category ? 50 : 0) +
        Math.max(0, 30 - Math.abs(l.price - anchor.price) / anchor.price * 30) +
        l.rating * 5,
      ai_reason: l.category === anchor.category
        ? `Similar ${anchor.category} listing in ${l.location.split(",")[0]}`
        : `Comparable price point to "${anchor.title}"`,
      reason_tag: l.category === anchor.category ? "Similar Category" : "Price Match",
    }))
    .sort((a, b) => b.score - a.score)
    .slice(0, limit);

  return res.json({
    success: true,
    data: { anchor, similar, ai_powered: false, count: similar.length },
  });
});

// ── GET /api/v1/ai/recommend/trending ─────────────────────────────────────────

router.get("/trending", (_req, res) => {
  const trending = CATALOG
    .sort((a, b) => b.rating - a.rating)
    .slice(0, 8)
    .map(l => ({ ...l, ai_reason: "Top-rated in the GCC marketplace", reason_tag: "Trending" }));

  res.json({ success: true, data: { trending, count: trending.length } });
});

export default router;
