'use client'
import { Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useState, useEffect, useRef, useCallback } from "react";
import { Search, Sparkles, TrendingUp, X, Filter, MapPin, Tag, Star, ChevronDown } from "lucide-react";
import api from "@/lib/api";

// ── Mock data for AI search (used when backend is unavailable) ────────────────
const MOCK_LISTINGS = [
  { id:"lst_001", title:"iPhone 15 Pro Max 256GB - Like New", price:4200, currency:"AED", category:"Electronics", location:"Dubai, UAE", condition:"Like New", image:"https://picsum.photos/seed/iphone15/400/300", seller:"Ahmed Al Mansoori", rating:4.9, created_at:"2026-03-20", relevance_score:95, ai_reason:"Excellent semantic match" },
  { id:"lst_002", title:"Samsung Galaxy S24 Ultra - Black", price:3800, currency:"AED", category:"Electronics", location:"Abu Dhabi, UAE", condition:"New", image:"https://picsum.photos/seed/samsung/400/300", seller:"Sara Tech Store", rating:4.7, created_at:"2026-03-19", relevance_score:88, ai_reason:"Strong match for your search" },
  { id:"lst_003", title:"MacBook Pro M3 14-inch 512GB", price:8500, currency:"AED", category:"Electronics", location:"Dubai, UAE", condition:"New", image:"https://picsum.photos/seed/macbook/400/300", seller:"iStore Dubai", rating:5.0, created_at:"2026-03-18", relevance_score:82, ai_reason:"Good match" },
  { id:"lst_004", title:"PS5 Console + 2 Controllers", price:2100, currency:"AED", category:"Electronics", location:"Sharjah, UAE", condition:"Good", image:"https://picsum.photos/seed/ps5/400/300", seller:"GameZone", rating:4.6, created_at:"2026-03-17", relevance_score:75, ai_reason:"Good match" },
  { id:"lst_005", title:"Toyota Land Cruiser 2024 GXR", price:320000, currency:"AED", category:"Vehicles", location:"Dubai, UAE", condition:"New", image:"https://picsum.photos/seed/landcruiser/400/300", seller:"Premier Motors", rating:4.8, created_at:"2026-03-16", relevance_score:70, ai_reason:"Good match" },
  { id:"lst_006", title:"Rolex Submariner - Stainless Steel", price:45000, currency:"AED", category:"Watches", location:"Dubai, UAE", condition:"Like New", image:"https://picsum.photos/seed/rolex/400/300", seller:"Luxury Timepieces", rating:4.9, created_at:"2026-03-15", relevance_score:65, ai_reason:"Possible match" },
  { id:"lst_007", title:"شقة فاخرة في دبي مارينا - 2 غرف", price:120000, currency:"AED", category:"Real Estate", location:"Dubai Marina, UAE", condition:"Like New", image:"https://picsum.photos/seed/apartment/400/300", seller:"Prime Properties", rating:4.7, created_at:"2026-03-14", relevance_score:60, ai_reason:"Possible match" },
  { id:"lst_008", title:"DJI Mavic 3 Pro Drone - Full Kit", price:5800, currency:"AED", category:"Electronics", location:"Abu Dhabi, UAE", condition:"New", image:"https://picsum.photos/seed/drone/400/300", seller:"Fly Tech UAE", rating:4.8, created_at:"2026-03-13", relevance_score:58, ai_reason:"Possible match" },
  { id:"lst_009", title:"iPhone 14 Pro 128GB Space Black", price:3200, currency:"AED", category:"Electronics", location:"Dubai, UAE", condition:"Good", image:"https://picsum.photos/seed/iphone14/400/300", seller:"Mobile World", rating:4.5, created_at:"2026-03-12", relevance_score:55, ai_reason:"Possible match" },
  { id:"lst_010", title:"Nike Air Jordan 1 Retro High OG", price:850, currency:"AED", category:"Clothing", location:"Dubai, UAE", condition:"New", image:"https://picsum.photos/seed/jordan/400/300", seller:"Sneaker Lab", rating:4.6, created_at:"2026-03-11", relevance_score:50, ai_reason:"Possible match" },
  { id:"lst_011", title:"iPad Pro 12.9 M2 256GB WiFi", price:4600, currency:"AED", category:"Electronics", location:"Abu Dhabi, UAE", condition:"New", image:"https://picsum.photos/seed/ipad/400/300", seller:"iStore Abu Dhabi", rating:4.9, created_at:"2026-03-10", relevance_score:48, ai_reason:"Possible match" },
  { id:"lst_012", title:"Sony PlayStation VR2 Headset", price:1800, currency:"AED", category:"Electronics", location:"Sharjah, UAE", condition:"Like New", image:"https://picsum.photos/seed/psvr/400/300", seller:"GameZone UAE", rating:4.4, created_at:"2026-03-09", relevance_score:45, ai_reason:"Possible match" },
];

const TRENDING_SEARCHES = [
  { query: "iPhone 15 Pro", count: 1240 }, { query: "Toyota Land Cruiser", count: 980 },
  { query: "شقة دبي", count: 875 }, { query: "PS5", count: 760 },
  { query: "MacBook Pro M3", count: 710 }, { query: "Rolex", count: 620 },
  { query: "سيارة للبيع", count: 590 }, { query: "DJI Drone", count: 480 },
];

function mockSearch(q: string, filters: Record<string, unknown>) {
  const ql = q.toLowerCase();
  let results = MOCK_LISTINGS.filter(l =>
    l.title.toLowerCase().includes(ql) ||
    l.category.toLowerCase().includes(ql) ||
    l.location.toLowerCase().includes(ql) ||
    l.title.includes(q) // Arabic support
  );
  if (results.length === 0) results = MOCK_LISTINGS; // fallback: show all
  if (filters.category) results = results.filter(l => l.category === filters.category);
  if (filters.price_max) results = results.filter(l => l.price <= Number(filters.price_max));
  if (filters.location) results = results.filter(l => l.location.toLowerCase().includes(String(filters.location).toLowerCase()));
  // Build intent
  const intent = {
    keywords: q.split(" "),
    category: ql.includes("iphone") || ql.includes("samsung") || ql.includes("laptop") || ql.includes("ps5") ? "Electronics"
      : ql.includes("car") || ql.includes("toyota") || ql.includes("سيارة") ? "Vehicles"
      : ql.includes("villa") || ql.includes("apartment") || ql.includes("شقة") ? "Real Estate" : undefined,
    location: ql.includes("dubai") ? "Dubai" : ql.includes("abu dhabi") ? "Abu Dhabi" : undefined,
    summary: `AI found ${results.length} listings matching "${q}" in the GCC marketplace`,
    suggestions: [`Used ${q}`, `${q} Dubai`, `Best price ${q}`].slice(0, 3),
  };
  return { results, intent, total: results.length, ai_powered: true };
}

// ── Types ────────────────────────────────────────────────────────────────────

/** Shape of a listing returned by the Go backend (/listings or /listings/search) */
interface BackendListing {
  id: string;
  title: string;
  price: number | null;
  currency: string;
  condition: string;
  city: string;
  country: string;
  created_at: string;
  images?: { url: string }[];
  category?: { name_en?: string; slug?: string } | null;
  seller?: { name?: string; rating?: number } | null;
}

interface SearchResult {
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

function backendListingToResult(l: BackendListing): SearchResult {
  return {
    id: l.id,
    title: l.title,
    price: l.price ?? 0,
    currency: l.currency ?? "AED",
    category: l.category?.name_en ?? l.category?.slug ?? "",
    location: [l.city, l.country].filter(Boolean).join(", "),
    condition: l.condition ?? "",
    image: l.images?.[0]?.url ?? `https://picsum.photos/seed/${l.id}/400/300`,
    seller: l.seller?.name ?? "",
    rating: l.seller?.rating ?? 0,
    created_at: l.created_at,
    relevance_score: 80,
    ai_reason: "Matched your search",
  };
}

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

interface LiveResult {
  session_id: string;
  title: string;
  viewers: number;
  thumbnail_url?: string;
  is_hot?: boolean;
  is_premium?: boolean;
  urgency?: string;
}

interface SearchState {
  results: SearchResult[];
  intent: SearchIntent | null;
  total: number;
  ai_powered: boolean;
  loading: boolean;
  error: string | null;
  live_results: LiveResult[];
}

// ── Debounce hook ────────────────────────────────────────────────────────────
function useDebounce<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(t);
  }, [value, delay]);
  return debounced;
}

// ── ResultCard ────────────────────────────────────────────────────────────────
function ResultCard({ result, onClick }: { result: SearchResult; onClick: () => void }) {
  return (
    <div
      onClick={onClick}
      className="bg-white rounded-2xl overflow-hidden shadow-sm border border-gray-100 hover:shadow-md hover:border-[#0071CE]/30 transition-all cursor-pointer group"
    >
      <div className="relative">
        <img
          src={result.image}
          alt={result.title}
          className="w-full h-44 object-cover group-hover:scale-[1.02] transition-transform duration-300"
        />
        {result.relevance_score >= 30 && (
          <div className="absolute top-2 left-2 flex items-center gap-1 bg-[#0071CE] text-white text-xs font-semibold px-2 py-0.5 rounded-full">
            <Sparkles className="w-3 h-3" /> Best Match
          </div>
        )}
        <div className="absolute top-2 right-2 bg-white/90 backdrop-blur-sm text-xs text-gray-600 px-2 py-0.5 rounded-full">
          {result.condition}
        </div>
      </div>

      <div className="p-3">
        <p className="text-sm font-semibold text-gray-800 line-clamp-2 leading-snug">{result.title}</p>
        <p className="text-[#0071CE] font-bold mt-1.5">
          {result.currency} {result.price.toLocaleString()}
        </p>

        <div className="flex items-center gap-3 mt-2 text-xs text-gray-400">
          <span className="flex items-center gap-0.5">
            <MapPin className="w-3 h-3" /> {result.location.split(",")[0]}
          </span>
          <span className="flex items-center gap-0.5">
            <Tag className="w-3 h-3" /> {result.category}
          </span>
          <span className="flex items-center gap-0.5 ml-auto">
            <Star className="w-3 h-3 fill-[#FFC220] text-[#FFC220]" /> {result.rating}
          </span>
        </div>
      </div>
    </div>
  );
}

// ── Main SearchPage ───────────────────────────────────────────────────────────
function SearchContent() {
  const router = useRouter();
  const urlSearchParams = useSearchParams();
  const [query, setQuery] = useState(urlSearchParams.get("q") || "");
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [suggestCategories, setSuggestCategories] = useState<{ slug: string; name_en: string; icon?: string }[]>([]);
  const [suggestLive, setSuggestLive] = useState<{ session_id: string; title: string; viewers: number }[]>([]);
  const [suggestTrending, setSuggestTrending] = useState<{ query: string; count: number }[]>([]);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [trending, setTrending] = useState<{ query: string; count: number }[]>([]);
  const [state, setState] = useState<SearchState>({
    results: [], intent: null, total: 0, ai_powered: false, loading: false, error: null, live_results: [],
  });
  const [showFilters, setShowFilters] = useState(false);
  const [filters, setFilters] = useState({ category: "", price_max: "", location: "" });
  const inputRef = useRef<HTMLInputElement>(null);
  const debouncedQuery = useDebounce(query, 300);

  // Load trending searches
  useEffect(() => {
    setTrending(TRENDING_SEARCHES);
  }, []);

  // Auto-suggest as user types — Amazon-style dropdown with listings, categories, live, trending.
  useEffect(() => {
    if (debouncedQuery.length < 2) {
      setSuggestions([]);
      setSuggestCategories([]);
      setSuggestLive([]);
      setSuggestTrending([]);
      return;
    }
    let cancelled = false;
    interface SuggestPayload {
      listings?: string[];
      categories?: { slug: string; name_en: string; icon?: string }[];
      live?: { session_id: string; title: string; viewers: number }[];
      trending?: { query: string; count: number }[];
    }
    api
      .get<{ data: SuggestPayload }>(`/listings/suggestions?q=${encodeURIComponent(debouncedQuery)}`)
      .then((res) => {
        if (cancelled) return;
        const d = res.data?.data ?? {};
        const items = Array.isArray(d.listings) ? d.listings : [];
        setSuggestions(
          items.length > 0
            ? items
            : [`${debouncedQuery} for sale`, `${debouncedQuery} Dubai`, `Buy ${debouncedQuery}`]
        );
        setSuggestCategories(Array.isArray(d.categories) ? d.categories : []);
        setSuggestLive(Array.isArray(d.live) ? d.live : []);
        setSuggestTrending(Array.isArray(d.trending) ? d.trending : []);
      })
      .catch(() => {
        if (!cancelled) {
          setSuggestions([`${debouncedQuery} for sale`, `${debouncedQuery} Dubai`, `Buy ${debouncedQuery}`]);
          setSuggestCategories([]);
          setSuggestLive([]);
          setSuggestTrending([]);
        }
      });
    return () => { cancelled = true; };
  }, [debouncedQuery]);

  // Main search
  const doSearch = useCallback(async (q: string) => {
    if (!q.trim()) return;
    setShowSuggestions(false);
    setState((s) => ({ ...s, loading: true, error: null }));

    // Category filter: convert display name to slug (e.g. "Real Estate" → "real-estate")
    const categorySlug = filters.category
      ? filters.category.toLowerCase().replace(/\s+/g, "-")
      : "";

    // ── Primary: backend /listings/search ──────────────────────────────────
    // This is always attempted first; results are authoritative (including empty).
    try {
      const params = new URLSearchParams({ q, per_page: "20" });
      if (categorySlug) params.set("category", categorySlug);
      if (filters.price_max) params.set("max_price", filters.price_max);
      if (filters.location) params.set("city", filters.location);
      const { data } = await api.get(`/listings/search?${params.toString()}`);
      const sr = data?.data as { results?: BackendListing[]; total?: number; live_results?: LiveResult[] } | undefined;
      if (sr) {
        const results = (sr.results ?? []).map(backendListingToResult);
        const liveResults = sr.live_results ?? [];

        // ── Optional: try AI for intent/suggestions only ──────────────────
        let intent: SearchIntent | null = null;
        try {
          const aiRes = await api.post("/ai/search", {
            query: q,
            filters: { category: filters.category, price_max: filters.price_max ? Number(filters.price_max) : undefined, location: filters.location },
          });
          const aiData = aiRes.data?.data as { intent?: SearchIntent } | undefined;
          if (aiData?.intent) intent = aiData.intent;
        } catch {
          // AI is optional — ignore errors
        }

        setState({ results, intent, total: sr.total ?? results.length, ai_powered: intent !== null, loading: false, error: null, live_results: liveResults });
        return;
      }
    } catch {
      // Backend search failed — fall through to mock
    }

    // ── Final fallback: mock data (only when backend throws an error) ──────
    const activeFilters: Record<string, unknown> = {};
    if (filters.category) activeFilters.category = filters.category;
    if (filters.price_max) activeFilters.price_max = Number(filters.price_max);
    if (filters.location) activeFilters.location = filters.location;
    const { results, intent, total, ai_powered } = mockSearch(q, activeFilters);
    setState({ results, intent, total, ai_powered, loading: false, error: null, live_results: [] });
  }, [filters]);

  // Search on Enter or query change
  const handleSubmit = (e?: React.FormEvent) => {
    e?.preventDefault();
    if (query.trim()) {
      router.push(`/search?q=${encodeURIComponent(query)}`);
      doSearch(query);
    }
  };

  // Initial search from URL
  useEffect(() => {
    const q = urlSearchParams.get("q");
    if (q) { setQuery(q); doSearch(q); }
  }, []);

  const CATEGORIES = ["Electronics", "Vehicles", "Real Estate", "Clothing", "Furniture", "Watches"];

  return (
    <div className="min-h-screen bg-[#F5F5F5]">
      {/* ── Hero Search Bar ── */}
      <div className="bg-[#0071CE] px-4 pt-8 pb-12">
        <div className="max-w-3xl mx-auto">
          <div className="flex items-center gap-2 mb-4">
            <Sparkles className="w-5 h-5 text-[#FFC220]" />
            <h1 className="text-white font-bold text-lg">AI-Powered Search</h1>
          </div>
          <form onSubmit={handleSubmit} className="relative">
            <div className="flex gap-2">
              <div className="flex-1 relative">
                <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
                <input
                  ref={inputRef}
                  value={query}
                  onChange={(e) => { setQuery(e.target.value); setShowSuggestions(true); }}
                  onFocus={() => setShowSuggestions(true)}
                  onBlur={() => setTimeout(() => setShowSuggestions(false), 150)}
                  placeholder="Search in Arabic or English… e.g. iPhone 15, سيارة رخيصة"
                  className="w-full pl-11 pr-10 py-4 rounded-2xl bg-white text-gray-800 placeholder-gray-400 text-sm focus:outline-none focus:ring-2 focus:ring-[#FFC220] shadow-lg"
                  autoFocus
                />
                {query && (
                  <button type="button" onClick={() => { setQuery(""); setSuggestions([]); inputRef.current?.focus(); }}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600">
                    <X className="w-4 h-4" />
                  </button>
                )}

                {/* Autocomplete dropdown — listings / categories / live / trending */}
                {showSuggestions && (suggestions.length > 0 || suggestCategories.length > 0 || suggestLive.length > 0 || suggestTrending.length > 0) && (
                  <div className="absolute top-full left-0 right-0 mt-1 bg-white rounded-xl shadow-xl border border-gray-100 z-50 overflow-hidden max-h-[70vh] overflow-y-auto">
                    {suggestions.length > 0 && (
                      <div>
                        <div className="px-4 pt-2 pb-1 text-[10px] font-semibold uppercase tracking-wide text-gray-400">Listings</div>
                        {suggestions.map((s, i) => (
                          <button key={`l-${i}`} type="button"
                            onClick={() => { setQuery(s); setShowSuggestions(false); doSearch(s); }}
                            className="w-full text-left flex items-center gap-3 px-4 py-2.5 hover:bg-gray-50 text-sm text-gray-700">
                            <Search className="w-4 h-4 text-gray-300 shrink-0" />
                            <span className="truncate">{s}</span>
                          </button>
                        ))}
                      </div>
                    )}
                    {suggestCategories.length > 0 && (
                      <div className="border-t border-gray-50">
                        <div className="px-4 pt-2 pb-1 text-[10px] font-semibold uppercase tracking-wide text-gray-400">Categories</div>
                        {suggestCategories.map((cat) => (
                          <button key={cat.slug} type="button"
                            onMouseDown={(e) => e.preventDefault()}
                            onClick={() => { setShowSuggestions(false); router.push(`/category/${cat.slug}`); }}
                            className="w-full text-left flex items-center gap-3 px-4 py-2.5 hover:bg-gray-50 text-sm">
                            <Tag className="w-4 h-4 text-[#0071CE] shrink-0" />
                            <span className="truncate text-gray-700">{cat.name_en}</span>
                            {cat.icon && <span className="ml-auto text-xs text-gray-300">{cat.icon}</span>}
                          </button>
                        ))}
                      </div>
                    )}
                    {suggestLive.length > 0 && (
                      <div className="border-t border-gray-50">
                        <div className="px-4 pt-2 pb-1 text-[10px] font-semibold uppercase tracking-wide text-rose-500 flex items-center gap-1">
                          <span className="w-1.5 h-1.5 rounded-full bg-rose-500 animate-pulse" /> Live now
                        </div>
                        {suggestLive.map((lr) => (
                          <button key={lr.session_id} type="button"
                            onMouseDown={(e) => e.preventDefault()}
                            onClick={() => { setShowSuggestions(false); router.push(`/auctions/live/${lr.session_id}`); }}
                            className="w-full text-left flex items-center gap-3 px-4 py-2.5 hover:bg-rose-50 text-sm">
                            <span className="inline-flex items-center gap-1 text-[10px] font-bold text-white bg-rose-600 px-1.5 py-0.5 rounded">LIVE</span>
                            <span className="truncate text-gray-700 flex-1">{lr.title}</span>
                            <span className="text-xs text-gray-400">👁 {lr.viewers.toLocaleString()}</span>
                          </button>
                        ))}
                      </div>
                    )}
                    {suggestTrending.length > 0 && (
                      <div className="border-t border-gray-50">
                        <div className="px-4 pt-2 pb-1 text-[10px] font-semibold uppercase tracking-wide text-gray-400">Trending</div>
                        {suggestTrending.map((t) => (
                          <button key={t.query} type="button"
                            onClick={() => { setQuery(t.query); setShowSuggestions(false); doSearch(t.query); }}
                            className="w-full text-left flex items-center gap-3 px-4 py-2.5 hover:bg-gray-50 text-sm text-gray-700">
                            <TrendingUp className="w-4 h-4 text-orange-400 shrink-0" />
                            <span className="truncate flex-1">{t.query}</span>
                            <span className="text-xs text-gray-300">{t.count.toLocaleString()}</span>
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>

              <button type="submit"
                className="bg-[#FFC220] hover:bg-yellow-400 text-gray-900 font-bold px-6 py-4 rounded-2xl transition-colors shrink-0 shadow-lg">
                Search
              </button>
            </div>
          </form>

          {/* Filter toggle */}
          <button onClick={() => setShowFilters(!showFilters)}
            className="mt-3 flex items-center gap-2 text-white/80 hover:text-white text-sm transition-colors">
            <Filter className="w-4 h-4" />
            {showFilters ? "Hide filters" : "Add filters"}
            <ChevronDown className={`w-4 h-4 transition-transform ${showFilters ? "rotate-180" : ""}`} />
          </button>

          {/* Filters panel */}
          {showFilters && (
            <div className="mt-3 bg-white/10 backdrop-blur-sm rounded-2xl p-4 grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div>
                <label className="block text-white/80 text-xs font-semibold mb-1.5 uppercase tracking-wide">Category</label>
                <select value={filters.category}
                  onChange={(e) => setFilters({ ...filters, category: e.target.value })}
                  className="w-full bg-white/20 text-white rounded-xl px-3 py-2.5 text-sm border border-white/20 focus:outline-none focus:ring-1 focus:ring-white/50">
                  <option value="">All Categories</option>
                  {CATEGORIES.map((c) => <option key={c} value={c}>{c}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-white/80 text-xs font-semibold mb-1.5 uppercase tracking-wide">Max Price (AED)</label>
                <input value={filters.price_max}
                  onChange={(e) => setFilters({ ...filters, price_max: e.target.value })}
                  placeholder="e.g. 5000"
                  type="number"
                  className="w-full bg-white/20 text-white placeholder-white/60 rounded-xl px-3 py-2.5 text-sm border border-white/20 focus:outline-none focus:ring-1 focus:ring-white/50"
                />
              </div>
              <div>
                <label className="block text-white/80 text-xs font-semibold mb-1.5 uppercase tracking-wide">Location</label>
                <input value={filters.location}
                  onChange={(e) => setFilters({ ...filters, location: e.target.value })}
                  placeholder="e.g. Dubai"
                  className="w-full bg-white/20 text-white placeholder-white/60 rounded-xl px-3 py-2.5 text-sm border border-white/20 focus:outline-none focus:ring-1 focus:ring-white/50"
                />
              </div>
            </div>
          )}
        </div>
      </div>

      {/* ── Content area ── */}
      <div className="max-w-7xl mx-auto px-4 -mt-6 pb-12">

        {/* No query yet: show trending */}
        {!query && !state.loading && state.results.length === 0 && (
          <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-6">
            <div className="flex items-center gap-2 mb-4 text-gray-700 font-semibold">
              <TrendingUp className="w-5 h-5 text-[#0071CE]" />
              Trending searches
            </div>
            <div className="flex flex-wrap gap-2">
              {trending.map((t) => (
                <button key={t.query}
                  onClick={() => { setQuery(t.query); doSearch(t.query); router.push(`/search?q=${encodeURIComponent(t.query)}`); }}
                  className="flex items-center gap-1.5 bg-gray-50 hover:bg-[#0071CE]/10 border border-gray-100 hover:border-[#0071CE]/30 rounded-full px-4 py-2 text-sm text-gray-700 hover:text-[#0071CE] transition-colors">
                  <span>{t.query}</span>
                  <span className="text-xs text-gray-400">{t.count.toLocaleString()}</span>
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Loading skeleton */}
        {state.loading && (
          <div className="space-y-4">
            <div className="h-16 bg-white rounded-2xl animate-pulse" />
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
              {Array.from({ length: 8 }).map((_, i) => (
                <div key={i} className="bg-white rounded-2xl overflow-hidden animate-pulse">
                  <div className="h-44 bg-gray-100" />
                  <div className="p-3 space-y-2">
                    <div className="h-3 bg-gray-100 rounded w-3/4" />
                    <div className="h-4 bg-gray-100 rounded w-1/2" />
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Error */}
        {state.error && !state.loading && (
          <div className="bg-red-50 border border-red-200 rounded-2xl p-6 text-center text-red-600">
            {state.error}
          </div>
        )}

        {/* AI Intent summary */}
        {state.intent && !state.loading && (
          <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-4 mb-4">
            <div className="flex items-start gap-3">
              <div className="w-8 h-8 bg-[#0071CE]/10 rounded-full flex items-center justify-center shrink-0">
                <Sparkles className="w-4 h-4 text-[#0071CE]" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-semibold text-gray-800">{state.intent.summary}</p>
                <div className="flex flex-wrap gap-1.5 mt-2">
                  {state.intent.category && (
                    <span className="bg-[#0071CE]/10 text-[#0071CE] text-xs font-medium px-2.5 py-1 rounded-full flex items-center gap-1">
                      <Tag className="w-3 h-3" /> {state.intent.category}
                    </span>
                  )}
                  {state.intent.location && (
                    <span className="bg-green-50 text-green-700 text-xs font-medium px-2.5 py-1 rounded-full flex items-center gap-1">
                      <MapPin className="w-3 h-3" /> {state.intent.location}
                    </span>
                  )}
                  {state.intent.price_max && (
                    <span className="bg-yellow-50 text-yellow-700 text-xs font-medium px-2.5 py-1 rounded-full">
                      Up to AED {state.intent.price_max.toLocaleString()}
                    </span>
                  )}
                  {state.intent.condition && (
                    <span className="bg-purple-50 text-purple-700 text-xs font-medium px-2.5 py-1 rounded-full">
                      {state.intent.condition}
                    </span>
                  )}
                </div>
                {state.intent.suggestions.length > 0 && (
                  <div className="mt-3">
                    <p className="text-xs text-gray-400 mb-1">Related searches:</p>
                    <div className="flex flex-wrap gap-1.5">
                      {state.intent.suggestions.map((s) => (
                        <button key={s}
                          onClick={() => { setQuery(s); doSearch(s); router.push(`/search?q=${encodeURIComponent(s)}`); }}
                          className="text-xs text-[#0071CE] hover:underline bg-[#0071CE]/5 hover:bg-[#0071CE]/10 px-2.5 py-1 rounded-full transition-colors">
                          {s}
                        </button>
                      ))}
                    </div>
                  </div>
                )}
              </div>
              <div className="shrink-0 text-right">
                <p className="text-xs text-gray-400">{state.results.length} of {state.total} results</p>
                {state.ai_powered && (
                  <span className="inline-flex items-center gap-1 text-xs text-[#0071CE] font-medium mt-1">
                    <Sparkles className="w-3 h-3" /> AI Powered
                  </span>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Live sessions strip (Sprint 18) */}
        {state.live_results.length > 0 && !state.loading && (
          <div className="mb-5 bg-gradient-to-r from-rose-50 to-orange-50 border border-rose-100 rounded-2xl p-4">
            <div className="flex items-center gap-2 mb-3">
              <span className="inline-flex items-center gap-1 bg-rose-600 text-white text-xs font-bold px-2 py-0.5 rounded-full">
                <span className="w-1.5 h-1.5 bg-white rounded-full animate-pulse" /> LIVE
              </span>
              <span className="text-sm font-semibold text-gray-800">
                Matching live sessions ({state.live_results.length})
              </span>
            </div>
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
              {state.live_results.map((lr) => (
                <button
                  key={lr.session_id}
                  onClick={() => router.push(`/auctions/live/${lr.session_id}`)}
                  className="relative bg-white rounded-xl overflow-hidden shadow-sm border border-gray-100 hover:shadow-md hover:border-rose-300 transition-all text-left"
                >
                  <div className="relative aspect-video bg-gray-100">
                    {lr.thumbnail_url ? (
                      <img src={lr.thumbnail_url} alt={lr.title} className="w-full h-full object-cover" />
                    ) : (
                      <div className="w-full h-full flex items-center justify-center text-gray-300 text-xs">No preview</div>
                    )}
                    <span className="absolute top-1.5 left-1.5 bg-rose-600 text-white text-[10px] font-bold px-1.5 py-0.5 rounded">
                      LIVE
                    </span>
                    {lr.urgency === "very_hot" && (
                      <span className="absolute top-1.5 right-1.5 bg-yellow-400 text-gray-900 text-[10px] font-bold px-1.5 py-0.5 rounded">
                        🔥 HOT
                      </span>
                    )}
                    <span className="absolute bottom-1.5 right-1.5 bg-black/60 text-white text-[10px] px-1.5 py-0.5 rounded">
                      👁 {lr.viewers.toLocaleString()}
                    </span>
                  </div>
                  <div className="p-2">
                    <p className="text-xs font-semibold text-gray-800 line-clamp-2 leading-snug">
                      {lr.title}
                    </p>
                  </div>
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Results grid */}
        {state.results.length > 0 && !state.loading && (
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
            {state.results.map((result) => (
              <ResultCard key={result.id} result={result}
                onClick={() => router.push(`/listings/${result.id}`)} />
            ))}
          </div>
        )}

        {/* No results */}
        {!state.loading && !state.error && state.results.length === 0 && query && (
          <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-12 text-center">
            <Search className="w-12 h-12 text-gray-200 mx-auto mb-3" />
            <p className="text-gray-500 font-medium">No results found for "{query}"</p>
            <p className="text-gray-400 text-sm mt-1">Try different keywords or remove some filters</p>
          </div>
        )}
      </div>
    </div>
  );
}

export default function SearchPage() {
  return (
    <Suspense fallback={null}>
      <SearchContent />
    </Suspense>
  );
}
