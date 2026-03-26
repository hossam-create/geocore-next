// ── GeoCore AI Recommendation Engine ─────────────────────────────────────────
// Client-side collaborative filtering + content-based scoring
// Tracks user behavior in localStorage and ranks listings accordingly.

export interface Listing {
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
  relevance_score?: number;
  ai_reason?: string;
}

export interface UserPreferences {
  viewedCategories: Record<string, number>;   // category → view count
  viewedListings: string[];                   // listing IDs (most recent first)
  priceRange: { min: number; max: number };
  preferredLocations: Record<string, number>; // location → view count
  searchHistory: string[];
  lastUpdated: number;
}

const PREFS_KEY = "geocore_user_prefs_v2";
const MAX_HISTORY = 50;

// ── Default GCC marketplace listings ─────────────────────────────────────────
export const ALL_LISTINGS: Listing[] = [
  { id:"lst_001", title:"iPhone 15 Pro Max 256GB - Like New", price:4200, currency:"AED", category:"Electronics", location:"Dubai, UAE", condition:"Like New", image:"https://picsum.photos/seed/iphone15/400/300", seller:"Ahmed Al Mansoori", rating:4.9, created_at:"2026-03-20" },
  { id:"lst_002", title:"Samsung Galaxy S24 Ultra - Black", price:3800, currency:"AED", category:"Electronics", location:"Abu Dhabi, UAE", condition:"New", image:"https://picsum.photos/seed/samsung/400/300", seller:"Sara Tech Store", rating:4.7, created_at:"2026-03-19" },
  { id:"lst_003", title:"MacBook Pro M3 14-inch 512GB", price:8500, currency:"AED", category:"Electronics", location:"Dubai, UAE", condition:"New", image:"https://picsum.photos/seed/macbook/400/300", seller:"iStore Dubai", rating:5.0, created_at:"2026-03-18" },
  { id:"lst_004", title:"PS5 Console + 2 Controllers", price:2100, currency:"AED", category:"Electronics", location:"Sharjah, UAE", condition:"Good", image:"https://picsum.photos/seed/ps5/400/300", seller:"GameZone", rating:4.6, created_at:"2026-03-17" },
  { id:"lst_005", title:"Toyota Land Cruiser 2024 GXR", price:320000, currency:"AED", category:"Vehicles", location:"Dubai, UAE", condition:"New", image:"https://picsum.photos/seed/landcruiser/400/300", seller:"Premier Motors", rating:4.8, created_at:"2026-03-16" },
  { id:"lst_006", title:"Rolex Submariner - Stainless Steel", price:45000, currency:"AED", category:"Watches", location:"Dubai, UAE", condition:"Like New", image:"https://picsum.photos/seed/rolex/400/300", seller:"Luxury Timepieces", rating:4.9, created_at:"2026-03-15" },
  { id:"lst_007", title:"شقة فاخرة في دبي مارينا - 2 غرف", price:120000, currency:"AED", category:"Real Estate", location:"Dubai Marina, UAE", condition:"Like New", image:"https://picsum.photos/seed/apartment/400/300", seller:"Prime Properties", rating:4.7, created_at:"2026-03-14" },
  { id:"lst_008", title:"DJI Mavic 3 Pro Drone - Full Kit", price:5800, currency:"AED", category:"Electronics", location:"Abu Dhabi, UAE", condition:"New", image:"https://picsum.photos/seed/drone/400/300", seller:"Fly Tech UAE", rating:4.8, created_at:"2026-03-13" },
  { id:"lst_009", title:"iPhone 14 Pro 128GB Space Black", price:3200, currency:"AED", category:"Electronics", location:"Dubai, UAE", condition:"Good", image:"https://picsum.photos/seed/iphone14/400/300", seller:"Mobile World", rating:4.5, created_at:"2026-03-12" },
  { id:"lst_010", title:"Nike Air Jordan 1 Retro High OG", price:850, currency:"AED", category:"Clothing", location:"Dubai, UAE", condition:"New", image:"https://picsum.photos/seed/jordan/400/300", seller:"Sneaker Lab", rating:4.6, created_at:"2026-03-11" },
  { id:"lst_011", title:"iPad Pro 12.9 M2 256GB WiFi", price:4600, currency:"AED", category:"Electronics", location:"Abu Dhabi, UAE", condition:"New", image:"https://picsum.photos/seed/ipad/400/300", seller:"iStore Abu Dhabi", rating:4.9, created_at:"2026-03-10" },
  { id:"lst_012", title:"Sony PlayStation VR2 Headset", price:1800, currency:"AED", category:"Electronics", location:"Sharjah, UAE", condition:"Like New", image:"https://picsum.photos/seed/psvr/400/300", seller:"GameZone UAE", rating:4.4, created_at:"2026-03-09" },
  { id:"lst_013", title:"Porsche 911 Carrera S 2023", price:680000, currency:"AED", category:"Vehicles", location:"Dubai, UAE", condition:"Like New", image:"https://picsum.photos/seed/porsche/400/300", seller:"Prestige Auto", rating:5.0, created_at:"2026-03-08" },
  { id:"lst_014", title:"Rolex Daytona Chronograph", price:95000, currency:"AED", category:"Watches", location:"Abu Dhabi, UAE", condition:"New", image:"https://picsum.photos/seed/daytona/400/300", seller:"Watch House", rating:4.9, created_at:"2026-03-07" },
  { id:"lst_015", title:"Hermès Birkin Bag 30cm - Gold", price:85000, currency:"AED", category:"Fashion", location:"Dubai, UAE", condition:"Like New", image:"https://picsum.photos/seed/hermes/400/300", seller:"Luxury Closet", rating:4.8, created_at:"2026-03-06" },
  { id:"lst_016", title:"Villa in Palm Jumeirah - 5 BR", price:8500000, currency:"AED", category:"Real Estate", location:"Palm Jumeirah, UAE", condition:"New", image:"https://picsum.photos/seed/villa/400/300", seller:"Emaar Properties", rating:5.0, created_at:"2026-03-05" },
  { id:"lst_017", title:"Apple Watch Ultra 2 - Titanium", price:3200, currency:"AED", category:"Electronics", location:"Dubai, UAE", condition:"New", image:"https://picsum.photos/seed/applewatch/400/300", seller:"iStore Dubai", rating:4.8, created_at:"2026-03-04" },
  { id:"lst_018", title:"Samsung 85\" QLED 8K Smart TV", price:12000, currency:"AED", category:"Electronics", location:"Sharjah, UAE", condition:"New", image:"https://picsum.photos/seed/samsung_tv/400/300", seller:"Carrefour UAE", rating:4.7, created_at:"2026-03-03" },
  { id:"lst_019", title:"Yamaha YZF-R1 2024 - Sport Bike", price:85000, currency:"AED", category:"Vehicles", location:"Dubai, UAE", condition:"New", image:"https://picsum.photos/seed/yamaha/400/300", seller:"Desert Bikes", rating:4.6, created_at:"2026-03-02" },
  { id:"lst_020", title:"Louis Vuitton Neverfull MM - Monogram", price:8500, currency:"AED", category:"Fashion", location:"Dubai, UAE", condition:"Good", image:"https://picsum.photos/seed/lv/400/300", seller:"Luxury Resale", rating:4.5, created_at:"2026-03-01" },
];

// ── Preference Management ─────────────────────────────────────────────────────

export function loadPreferences(): UserPreferences {
  try {
    const raw = localStorage.getItem(PREFS_KEY);
    if (raw) return JSON.parse(raw);
  } catch { /* ignore */ }
  return {
    viewedCategories: {},
    viewedListings: [],
    priceRange: { min: 0, max: 1_000_000 },
    preferredLocations: {},
    searchHistory: [],
    lastUpdated: Date.now(),
  };
}

export function savePreferences(prefs: UserPreferences) {
  try {
    localStorage.setItem(PREFS_KEY, JSON.stringify({ ...prefs, lastUpdated: Date.now() }));
  } catch { /* ignore */ }
}

export function trackListing(listing: Listing) {
  const prefs = loadPreferences();

  // Track category
  prefs.viewedCategories[listing.category] = (prefs.viewedCategories[listing.category] || 0) + 1;

  // Track listing ID (deduplicate, keep most recent)
  prefs.viewedListings = [listing.id, ...prefs.viewedListings.filter(id => id !== listing.id)].slice(0, MAX_HISTORY);

  // Track location city
  const city = listing.location.split(",")[0].trim();
  prefs.preferredLocations[city] = (prefs.preferredLocations[city] || 0) + 1;

  // Update price range dynamically
  if (listing.price > 0) {
    if (prefs.viewedListings.length <= 3) {
      prefs.priceRange = { min: listing.price * 0.3, max: listing.price * 2 };
    } else {
      prefs.priceRange.min = Math.min(prefs.priceRange.min, listing.price * 0.5);
      prefs.priceRange.max = Math.max(prefs.priceRange.max, listing.price * 1.5);
    }
  }

  savePreferences(prefs);
}

export function trackSearch(query: string) {
  const prefs = loadPreferences();
  prefs.searchHistory = [query, ...prefs.searchHistory.filter(q => q !== query)].slice(0, 20);
  savePreferences(prefs);
}

// ── Scoring Engine ────────────────────────────────────────────────────────────

export interface ScoredListing extends Listing {
  score: number;
  ai_reason: string;
  reason_tag: "Top Pick" | "Near You" | "Similar Category" | "Price Match" | "Trending" | "New Arrival";
}

function scoreListing(listing: Listing, prefs: UserPreferences, exclude: string[]): number {
  if (exclude.includes(listing.id)) return -1;

  let score = 0;

  // 1. Category preference (0-40 pts)
  const catViews = prefs.viewedCategories[listing.category] || 0;
  const maxCatViews = Math.max(...Object.values(prefs.viewedCategories), 1);
  score += (catViews / maxCatViews) * 40;

  // 2. Location preference (0-20 pts)
  const city = listing.location.split(",")[0].trim();
  const locViews = prefs.preferredLocations[city] || 0;
  const maxLocViews = Math.max(...Object.values(prefs.preferredLocations), 1);
  score += (locViews / maxLocViews) * 20;

  // 3. Price range match (0-25 pts)
  if (listing.price >= prefs.priceRange.min && listing.price <= prefs.priceRange.max) {
    score += 25;
  } else {
    const diff = Math.min(
      Math.abs(listing.price - prefs.priceRange.min),
      Math.abs(listing.price - prefs.priceRange.max)
    );
    const rangeMid = (prefs.priceRange.min + prefs.priceRange.max) / 2;
    score += Math.max(0, 15 - (diff / rangeMid) * 15);
  }

  // 4. Rating bonus (0-10 pts)
  score += (listing.rating / 5) * 10;

  // 5. Recency bonus (0-5 pts)
  const days = (Date.now() - new Date(listing.created_at).getTime()) / 86_400_000;
  score += Math.max(0, 5 - days * 0.2);

  return score;
}

function getReasonTag(listing: Listing, prefs: UserPreferences): { tag: ScoredListing["reason_tag"]; reason: string } {
  const catViews = prefs.viewedCategories[listing.category] || 0;
  const city = listing.location.split(",")[0].trim();
  const locViews = prefs.preferredLocations[city] || 0;
  const days = (Date.now() - new Date(listing.created_at).getTime()) / 86_400_000;

  if (catViews >= 3 && locViews >= 1) return { tag: "Top Pick", reason: `Perfect match for your interest in ${listing.category} in ${city}` };
  if (locViews >= 2) return { tag: "Near You", reason: `Popular in ${city} — a location you frequently browse` };
  if (catViews >= 2) return { tag: "Similar Category", reason: `Based on your interest in ${listing.category}` };
  if (listing.price >= prefs.priceRange.min && listing.price <= prefs.priceRange.max) return { tag: "Price Match", reason: `Within your typical price range (${listing.currency} ${Math.round(prefs.priceRange.min).toLocaleString()}–${Math.round(prefs.priceRange.max).toLocaleString()})` };
  if (days <= 3) return { tag: "New Arrival", reason: "Freshly listed in the last 3 days" };
  return { tag: "Trending", reason: `Trending in the GCC marketplace` };
}

// ── Main Recommendation Functions ─────────────────────────────────────────────

export function getRecommendations(options: {
  limit?: number;
  exclude?: string[];
  category?: string;
}): ScoredListing[] {
  const { limit = 6, exclude = [], category } = options;
  const prefs = loadPreferences();

  let pool = ALL_LISTINGS;
  if (category) pool = pool.filter(l => l.category === category);

  const hasPrefs = Object.keys(prefs.viewedCategories).length > 0;

  let scored: (ScoredListing & { score: number })[];

  if (hasPrefs) {
    scored = pool
      .map(l => {
        const score = scoreListing(l, prefs, exclude);
        const { tag, reason } = getReasonTag(l, prefs);
        return { ...l, score, reason_tag: tag, ai_reason: reason };
      })
      .filter(l => l.score >= 0)
      .sort((a, b) => b.score - a.score);
  } else {
    // Cold start: return top-rated recent listings
    scored = pool
      .filter(l => !exclude.includes(l.id))
      .map(l => ({
        ...l,
        score: l.rating * 20,
        reason_tag: "Trending" as const,
        ai_reason: "Top-rated listing in the GCC marketplace",
      }))
      .sort((a, b) => b.score - a.score);
  }

  return scored.slice(0, limit);
}

export function getSimilarListings(listing: Listing, limit = 4): ScoredListing[] {
  const prefs = loadPreferences();

  // Temporarily boost this listing's category to get better similar matches
  const tempPrefs: UserPreferences = {
    ...prefs,
    viewedCategories: { ...prefs.viewedCategories, [listing.category]: 99 },
    priceRange: { min: listing.price * 0.4, max: listing.price * 2 },
  };

  const scored = ALL_LISTINGS
    .filter(l => l.id !== listing.id)
    .map(l => {
      const score = scoreListing(l, tempPrefs, [listing.id]);
      const sameCategory = l.category === listing.category;
      const similarPrice = Math.abs(l.price - listing.price) / listing.price < 0.5;
      let reason_tag: ScoredListing["reason_tag"] = "Similar Category";
      let ai_reason = `Similar ${listing.category} listing`;

      if (sameCategory && similarPrice) {
        reason_tag = "Top Pick";
        ai_reason = `Same category and similar price range as "${listing.title}"`;
      } else if (sameCategory) {
        reason_tag = "Similar Category";
        ai_reason = `Another ${listing.category} listing you might like`;
      } else if (similarPrice) {
        reason_tag = "Price Match";
        ai_reason = `Similar price point to "${listing.title}"`;
      }

      return { ...l, score: score + (sameCategory ? 30 : 0) + (similarPrice ? 10 : 0), reason_tag, ai_reason };
    })
    .sort((a, b) => b.score - a.score);

  return scored.slice(0, limit);
}

export function getTopCategories(): Array<{ category: string; count: number }> {
  const prefs = loadPreferences();
  return Object.entries(prefs.viewedCategories)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 3)
    .map(([category, count]) => ({ category, count }));
}
