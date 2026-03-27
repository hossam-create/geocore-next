import { Router, type Request, type Response } from "express";
import OpenAI from "openai";

const router = Router();

// ── OpenAI client ────────────────────────────────────────────────────────────
const AI_CONFIGURED = !!(
  process.env.AI_INTEGRATIONS_OPENAI_BASE_URL &&
  process.env.AI_INTEGRATIONS_OPENAI_API_KEY
);

let openai: OpenAI | null = null;
if (AI_CONFIGURED) {
  openai = new OpenAI({
    baseURL: process.env.AI_INTEGRATIONS_OPENAI_BASE_URL,
    apiKey: process.env.AI_INTEGRATIONS_OPENAI_API_KEY,
  });
}

// ── Types ────────────────────────────────────────────────────────────────────
interface SearchIntent {
  keywords: string[];
  category?: string;
  price_min?: number;
  price_max?: number;
  location?: string;
  condition?: string;
  summary: string;
  suggestions: string[];
}

interface MockListing {
  id: string;
  title: string;
  price: number;
  currency: string;
  category: string;
  location: string;
  condition: string;
  image: string;
  seller: string;
  rating: number;
  created_at: string;
  relevance_score: number;
  ai_reason: string;
}

// ── Mock listing database ────────────────────────────────────────────────────
const MOCK_LISTINGS: Omit<MockListing, "relevance_score" | "ai_reason">[] = [
  { id: "lst_001", title: "iPhone 15 Pro Max 256GB - Like New", price: 4200, currency: "AED", category: "Electronics", location: "Dubai, UAE", condition: "Like New", image: "https://picsum.photos/seed/iphone15/400/300", seller: "Ahmed S.", rating: 4.9, created_at: "2026-03-20T10:00:00Z" },
  { id: "lst_002", title: "Samsung Galaxy S24 Ultra - Sealed Box", price: 3800, currency: "AED", category: "Electronics", location: "Abu Dhabi, UAE", condition: "New", image: "https://picsum.photos/seed/samsung/400/300", seller: "TechStore", rating: 4.7, created_at: "2026-03-22T08:00:00Z" },
  { id: "lst_003", title: "MacBook Pro M3 14\" - Perfect Condition", price: 7500, currency: "AED", category: "Electronics", location: "Dubai, UAE", condition: "Good", image: "https://picsum.photos/seed/macbook/400/300", seller: "Sara M.", rating: 4.8, created_at: "2026-03-18T14:00:00Z" },
  { id: "lst_004", title: "Toyota Land Cruiser 2022 GXR", price: 195000, currency: "AED", category: "Vehicles", location: "Riyadh, KSA", condition: "Good", image: "https://picsum.photos/seed/landcruiser/400/300", seller: "AutoDeals", rating: 4.6, created_at: "2026-03-15T09:00:00Z" },
  { id: "lst_005", title: "Villa 5BR - Palm Jumeirah View", price: 8500000, currency: "AED", category: "Real Estate", location: "Dubai, UAE", condition: "New", image: "https://picsum.photos/seed/villa/400/300", seller: "LuxRealty", rating: 4.9, created_at: "2026-03-10T12:00:00Z" },
  { id: "lst_006", title: "Sony PlayStation 5 + 3 Games Bundle", price: 1400, currency: "AED", category: "Electronics", location: "Sharjah, UAE", condition: "Like New", image: "https://picsum.photos/seed/ps5/400/300", seller: "GamerHub", rating: 4.5, created_at: "2026-03-21T16:00:00Z" },
  { id: "lst_007", title: "Rolex Submariner 2023 - Full Set", price: 42000, currency: "AED", category: "Watches", location: "Dubai, UAE", condition: "New", image: "https://picsum.photos/seed/rolex/400/300", seller: "LuxTime", rating: 5.0, created_at: "2026-03-19T11:00:00Z" },
  { id: "lst_008", title: "DJI Mavic 3 Pro Drone - Complete Kit", price: 5200, currency: "AED", category: "Electronics", location: "Kuwait City, KW", condition: "Like New", image: "https://picsum.photos/seed/drone/400/300", seller: "FlyTech", rating: 4.7, created_at: "2026-03-17T13:00:00Z" },
  { id: "lst_009", title: "Leather Sofa Set 7-Piece - Italian Design", price: 8900, currency: "AED", category: "Furniture", location: "Abu Dhabi, UAE", condition: "New", image: "https://picsum.photos/seed/sofa/400/300", seller: "HomeStyle", rating: 4.4, created_at: "2026-03-16T10:00:00Z" },
  { id: "lst_010", title: "Nike Air Jordan 1 Retro - Size 42 EU", price: 650, currency: "AED", category: "Clothing", location: "Dubai, UAE", condition: "New", image: "https://picsum.photos/seed/jordan/400/300", seller: "SneakerKing", rating: 4.8, created_at: "2026-03-23T15:00:00Z" },
  { id: "lst_011", title: "iPad Pro 12.9\" M2 + Apple Pencil", price: 3600, currency: "AED", category: "Electronics", location: "Doha, QA", condition: "Like New", image: "https://picsum.photos/seed/ipad/400/300", seller: "TabWorld", rating: 4.6, created_at: "2026-03-14T09:00:00Z" },
  { id: "lst_012", title: "Honda Civic 2023 - 15,000 km", price: 78000, currency: "AED", category: "Vehicles", location: "Dubai, UAE", condition: "Good", image: "https://picsum.photos/seed/civic/400/300", seller: "CarZone", rating: 4.5, created_at: "2026-03-13T08:00:00Z" },
];

// ── Helper: parse AI JSON safely ─────────────────────────────────────────────
function safeParseJSON<T>(text: string, fallback: T): T {
  try {
    const match = text.match(/```json\s*([\s\S]*?)```/) || text.match(/(\{[\s\S]*\})/);
    return JSON.parse(match ? match[1] : text) as T;
  } catch {
    return fallback;
  }
}

// ── Helper: text-based relevance scoring ────────────────────────────────────
function scoreListings(
  listings: typeof MOCK_LISTINGS,
  intent: SearchIntent
): MockListing[] {
  return listings
    .map((listing) => {
      let score = 0;
      const text = `${listing.title} ${listing.category} ${listing.location}`.toLowerCase();
      const query = intent.keywords.join(" ").toLowerCase();

      // Keyword matching
      intent.keywords.forEach((kw) => {
        if (text.includes(kw.toLowerCase())) score += 20;
      });

      // Category match
      if (intent.category && listing.category.toLowerCase().includes(intent.category.toLowerCase())) score += 30;

      // Location match
      if (intent.location && listing.location.toLowerCase().includes(intent.location.toLowerCase())) score += 15;

      // Condition match
      if (intent.condition && listing.condition.toLowerCase().includes(intent.condition.toLowerCase())) score += 10;

      // Price range
      if (intent.price_max && listing.price <= intent.price_max) score += 10;
      if (intent.price_min && listing.price >= intent.price_min) score += 5;

      // Boost new/recent listings
      const daysOld = (Date.now() - new Date(listing.created_at).getTime()) / (1000 * 60 * 60 * 24);
      if (daysOld < 3) score += 10;
      else if (daysOld < 7) score += 5;

      // Baseline score if matches generic query
      if (score === 0 && query && text.split(" ").some((w) => query.includes(w))) score = 5;

      return {
        ...listing,
        relevance_score: Math.min(100, score),
        ai_reason: score >= 30 ? "Strong match for your search" : score >= 15 ? "Partial match" : "Possible match",
      };
    })
    .sort((a, b) => b.relevance_score - a.relevance_score);
}

// ── POST /api/v1/ai/search ────────────────────────────────────────────────────
router.post("/v1/ai/search", async (req: Request, res: Response) => {
  const { query, filters = {}, limit = 12 } = req.body;

  if (!query || typeof query !== "string" || query.trim().length < 2) {
    return res.status(400).json({ success: false, message: "query must be at least 2 characters" });
  }

  let intent: SearchIntent = {
    keywords: query.trim().split(/\s+/).filter((w) => w.length > 2),
    summary: `Searching for: "${query}"`,
    suggestions: [],
  };

  // ── AI-powered query understanding ──────────────────────────────────────────
  if (openai) {
    try {
      const systemPrompt = `You are a GCC marketplace search assistant. Analyze search queries and extract structured intent.
Return ONLY valid JSON matching this exact schema:
{
  "keywords": ["keyword1", "keyword2"],
  "category": "Electronics|Vehicles|Real Estate|Clothing|Furniture|Watches|Other|null",
  "price_min": number_or_null,
  "price_max": number_or_null,
  "location": "city_or_country_or_null",
  "condition": "New|Like New|Good|Fair|null",
  "summary": "Human-readable summary of what user wants (in same language as query)",
  "suggestions": ["related search 1", "related search 2", "related search 3"]
}
Currency: assume AED (UAE Dirham). Be smart about context — "cheap" means price_max under 1000, "luxury" means price_min 10000+.`;

      const completion = await openai.chat.completions.create({
        model: "gpt-5-mini",
        messages: [
          { role: "system", content: systemPrompt },
          { role: "user", content: `Search query: "${query}"\nOptional filters: ${JSON.stringify(filters)}` },
        ],
        max_completion_tokens: 500,
      });

      const aiText = completion.choices[0]?.message?.content || "";
      const aiIntent = safeParseJSON<Partial<SearchIntent>>(aiText, {});
      intent = {
        keywords: aiIntent.keywords?.length ? aiIntent.keywords : intent.keywords,
        category: aiIntent.category || undefined,
        price_min: aiIntent.price_min || undefined,
        price_max: aiIntent.price_max || undefined,
        location: aiIntent.location || undefined,
        condition: aiIntent.condition || undefined,
        summary: aiIntent.summary || intent.summary,
        suggestions: aiIntent.suggestions || [],
      };
    } catch (err) {
      // Fall through to basic search
    }
  } else {
    // Simple keyword extraction without AI
    const lq = query.toLowerCase();
    if (lq.includes("car") || lq.includes("vehicle") || lq.includes("سيارة")) intent.category = "Vehicles";
    if (lq.includes("phone") || lq.includes("laptop") || lq.includes("iphone") || lq.includes("samsung")) intent.category = "Electronics";
    if (lq.includes("villa") || lq.includes("apartment") || lq.includes("شقة")) intent.category = "Real Estate";
    if (lq.includes("dubai")) intent.location = "Dubai";
    if (lq.includes("riyadh") || lq.includes("ksa")) intent.location = "Riyadh";
    if (lq.includes("new") || lq.includes("جديد")) intent.condition = "New";
    intent.summary = `Searching for: "${query}"`;
    intent.suggestions = [`${query} price`, `${query} used`, `${query} Dubai`];
  }

  // ── Score and rank listings ──────────────────────────────────────────────────
  const scored = scoreListings(MOCK_LISTINGS, intent);
  const results = scored.slice(0, limit);

  return res.json({
    success: true,
    data: {
      query,
      intent,
      results,
      total: MOCK_LISTINGS.length,
      returned: results.length,
      ai_powered: !!openai,
    },
  });
});

// ── GET /api/v1/ai/search/suggest — autocomplete suggestions ─────────────────
router.get("/v1/ai/search/suggest", async (req: Request, res: Response) => {
  const q = (req.query.q as string || "").trim();

  if (!q || q.length < 2) {
    return res.json({ success: true, data: { suggestions: [] } });
  }

  const defaultSuggestions = [
    `${q} for sale`,
    `${q} Dubai`,
    `${q} cheap`,
    `used ${q}`,
    `${q} new`,
  ].slice(0, 5);

  if (!openai) {
    return res.json({ success: true, data: { suggestions: defaultSuggestions, ai_powered: false } });
  }

  try {
    const completion = await openai.chat.completions.create({
      model: "gpt-5-nano",
      messages: [
        {
          role: "system",
          content: "You are a GCC marketplace search autocomplete engine. Given a partial query, return 5 relevant search suggestions as a JSON array of strings. Keep suggestions short (2-5 words). Match the language of the input (Arabic or English).",
        },
        { role: "user", content: `Partial query: "${q}"` },
      ],
      max_completion_tokens: 150,
    });

    const text = completion.choices[0]?.message?.content || "";
    const suggestions = safeParseJSON<string[]>(text, defaultSuggestions);

    return res.json({
      success: true,
      data: {
        suggestions: Array.isArray(suggestions) ? suggestions.slice(0, 5) : defaultSuggestions,
        ai_powered: true,
      },
    });
  } catch {
    return res.json({ success: true, data: { suggestions: defaultSuggestions, ai_powered: false } });
  }
});

// ── GET /api/v1/ai/search/trending — trending searches ──────────────────────
router.get("/v1/ai/search/trending", (_req: Request, res: Response) => {
  res.json({
    success: true,
    data: {
      trending: [
        { query: "iPhone 15 Pro", count: 1240 },
        { query: "Toyota Land Cruiser", count: 980 },
        { query: "شقة دبي", count: 875 },
        { query: "PS5", count: 760 },
        { query: "MacBook Pro M3", count: 710 },
        { query: "Rolex", count: 620 },
        { query: "سيارة للبيع", count: 590 },
        { query: "DJI Drone", count: 480 },
      ],
    },
  });
});

export default router;
