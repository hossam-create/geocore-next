'use client'
import { useRouter } from 'next/navigation';
import { useState } from "react";
import { Search, SlidersHorizontal } from "lucide-react";
import { useCategoryFields } from "@/hooks/useCategoryFields";

const CATEGORIES = [
  { label: "All Categories", slug: "" },
  { label: "Vehicles", slug: "vehicles" },
  { label: "Real Estate", slug: "real-estate" },
  { label: "Electronics", slug: "electronics" },
  { label: "Fashion", slug: "clothing" },
  { label: "Furniture", slug: "furniture" },
  { label: "Jewelry", slug: "jewelry" },
  { label: "Tools", slug: "tools" },
  { label: "Gaming", slug: "gaming" },
  { label: "Books", slug: "books" },
  { label: "Sports", slug: "sports" },
];
const CONDITIONS = ["Any Condition", "New", "Like New", "Good", "Fair", "Used"];
const LISTING_TYPES = ["All Types", "Buy Now", "Auction"];
const CURRENCIES = ["AED", "SAR", "KWD", "QAR", "BHD", "OMR"];
const LOCATIONS = [
  "All Locations", "Dubai", "Abu Dhabi", "Riyadh", "Jeddah",
  "Kuwait City", "Doha", "Manama", "Muscat",
];
const SORT_OPTIONS = [
  { value: "newest", label: "Newest First" },
  { value: "price_asc", label: "Price: Low to High" },
  { value: "price_desc", label: "Price: High to Low" },
  { value: "most_bids", label: "Most Bids" },
  { value: "ending_soon", label: "Ending Soon" },
];

export default function AdvancedSearchPage() {
  const router = useRouter();
  const [form, setForm] = useState({
    q: "",
    category: "",
    condition: "Any Condition",
    type: "All Types",
    minPrice: "",
    maxPrice: "",
    currency: "AED",
    location: "All Locations",
    sort: "newest",
    seller: "",
    freeDelivery: false,
    featuredOnly: false,
  });
  const [cfValues, setCfValues] = useState<Record<string, string>>({});

  const set = (key: string, val: any) => setForm((f) => ({ ...f, [key]: val }));
  const setCf = (name: string, value: string) => setCfValues((prev) => ({ ...prev, [name]: value }));

  const { data: categoryFields } = useCategoryFields(form.category || undefined);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    const params = new URLSearchParams();
    if (form.q) params.set("q", form.q);
    if (form.category) params.set("category", form.category);
    // Custom field filters
    for (const [key, val] of Object.entries(cfValues)) {
      if (val) params.set(`cf[${key}]`, val);
    }
    if (form.condition !== "Any Condition") params.set("condition", form.condition.toLowerCase().replace(/\s+/g, "-"));
    if (form.type !== "All Types") params.set("type", form.type === "Auction" ? "auction" : "fixed");
    if (form.minPrice) params.set("min_price", form.minPrice);
    if (form.maxPrice) params.set("max_price", form.maxPrice);
    if (form.location !== "All Locations") params.set("city", form.location);
    if (form.sort !== "newest") params.set("sort", form.sort);
    if (form.featuredOnly) params.set("is_featured", "true");
    router.push(`/listings?${params.toString()}`);
  };

  return (
    <div className="min-h-screen bg-[#f2f8fd]">
      <div className="max-w-4xl mx-auto px-4 py-10">
        {/* Title */}
        <div className="flex items-center gap-3 mb-8">
          <div className="w-10 h-10 rounded-xl bg-[#0071CE] flex items-center justify-center">
            <SlidersHorizontal size={20} className="text-white" />
          </div>
          <div>
            <h1 className="text-2xl font-extrabold text-gray-900">Advanced Search</h1>
            <p className="text-sm text-gray-500">Use multiple filters to find exactly what you're looking for</p>
          </div>
        </div>

        <form onSubmit={handleSearch} className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden">

          {/* ── Keyword ── */}
          <div className="p-6 border-b border-gray-100">
            <label className="block text-sm font-bold text-gray-700 mb-2">Keywords</label>
            <div className="flex gap-3">
              <div className="relative flex-1">
                <Search size={16} className="absolute left-3.5 top-1/2 -translate-y-1/2 text-gray-400" />
                <input
                  type="text"
                  value={form.q}
                  onChange={(e) => set("q", e.target.value)}
                  placeholder="Enter keywords, model, brand…"
                  className="w-full pl-10 pr-4 py-2.5 border border-gray-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-[#0071CE]/30 focus:border-[#0071CE]"
                />
              </div>
            </div>
            <div className="flex gap-4 mt-3 text-xs text-gray-500">
              <label className="flex items-center gap-1.5 cursor-pointer">
                <input type="radio" name="keyword_in" defaultChecked className="accent-[#0071CE]" />
                Title and description
              </label>
              <label className="flex items-center gap-1.5 cursor-pointer">
                <input type="radio" name="keyword_in" className="accent-[#0071CE]" />
                Title only
              </label>
            </div>
          </div>

          {/* ── Category + Condition + Type ── */}
          <div className="p-6 border-b border-gray-100 grid grid-cols-1 sm:grid-cols-3 gap-5">
            <div>
              <label className="block text-sm font-bold text-gray-700 mb-2">Category</label>
              <select
                value={form.category}
                onChange={(e) => { set("category", e.target.value); setCfValues({}); }}
                className="w-full border border-gray-200 rounded-xl py-2.5 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-[#0071CE]/30 focus:border-[#0071CE] bg-white"
              >
                {CATEGORIES.map((c) => <option key={c.slug} value={c.slug}>{c.label}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm font-bold text-gray-700 mb-2">Condition</label>
              <select
                value={form.condition}
                onChange={(e) => set("condition", e.target.value)}
                className="w-full border border-gray-200 rounded-xl py-2.5 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-[#0071CE]/30 focus:border-[#0071CE] bg-white"
              >
                {CONDITIONS.map((c) => <option key={c}>{c}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm font-bold text-gray-700 mb-2">Listing Type</label>
              <select
                value={form.type}
                onChange={(e) => set("type", e.target.value)}
                className="w-full border border-gray-200 rounded-xl py-2.5 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-[#0071CE]/30 focus:border-[#0071CE] bg-white"
              >
                {LISTING_TYPES.map((t) => <option key={t}>{t}</option>)}
              </select>
            </div>
          </div>

          {/* ── Category Custom Fields ── */}
          {categoryFields && categoryFields.length > 0 && (
            <div className="p-6 border-b border-gray-100">
              <label className="block text-sm font-bold text-gray-700 mb-3">Category Filters</label>
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                {categoryFields.map((field: any) => {
                  const val = cfValues[field.name] ?? "";
                  let options: { value: string; label: string }[] = [];
                  try { options = typeof field.options === "string" ? JSON.parse(field.options) : (field.options || []); } catch { /* empty */ }

                  if (field.field_type === "select") {
                    return (
                      <div key={field.id}>
                        <label className="block text-xs font-medium text-gray-600 mb-1">{field.label_en || field.label}</label>
                        <select
                          className="w-full border border-gray-200 rounded-xl py-2.5 px-3 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[#0071CE]/30"
                          value={val}
                          onChange={(e) => setCf(field.name, e.target.value)}
                        >
                          <option value="">Any</option>
                          {options.map((o: any) => <option key={o.value} value={o.value}>{o.label}</option>)}
                        </select>
                      </div>
                    );
                  }
                  if (field.field_type === "number" || field.field_type === "range") {
                    return (
                      <div key={field.id}>
                        <label className="block text-xs font-medium text-gray-600 mb-1">
                          {field.label_en || field.label} {field.unit && <span className="text-gray-400">({field.unit})</span>}
                        </label>
                        <div className="flex gap-2">
                          <input
                            type="number"
                            className="flex-1 border border-gray-200 rounded-xl py-2 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-[#0071CE]/30"
                            placeholder="Min"
                            value={cfValues[field.name + "_min"] ?? ""}
                            onChange={(e) => setCf(field.name + "_min", e.target.value)}
                          />
                          <input
                            type="number"
                            className="flex-1 border border-gray-200 rounded-xl py-2 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-[#0071CE]/30"
                            placeholder="Max"
                            value={cfValues[field.name + "_max"] ?? ""}
                            onChange={(e) => setCf(field.name + "_max", e.target.value)}
                          />
                        </div>
                      </div>
                    );
                  }
                  if (field.field_type === "boolean") {
                    return (
                      <div key={field.id} className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          checked={val === "true"}
                          onChange={(e) => setCf(field.name, e.target.checked ? "true" : "")}
                          className="w-4 h-4 accent-[#0071CE]"
                        />
                        <span className="text-sm text-gray-700">{field.label_en || field.label}</span>
                      </div>
                    );
                  }
                  return (
                    <div key={field.id}>
                      <label className="block text-xs font-medium text-gray-600 mb-1">{field.label_en || field.label}</label>
                      <input
                        type="text"
                        className="w-full border border-gray-200 rounded-xl py-2.5 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-[#0071CE]/30"
                        value={val}
                        onChange={(e) => setCf(field.name, e.target.value)}
                        placeholder={field.placeholder || field.label_en || field.label}
                      />
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {/* ── Price range + Location ── */}
          <div className="p-6 border-b border-gray-100 grid grid-cols-1 sm:grid-cols-2 gap-5">
            <div>
              <label className="block text-sm font-bold text-gray-700 mb-2">Price Range</label>
              <div className="flex gap-2 items-center">
                <select
                  value={form.currency}
                  onChange={(e) => set("currency", e.target.value)}
                  className="border border-gray-200 rounded-xl py-2.5 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-[#0071CE]/30 focus:border-[#0071CE] bg-white w-24"
                >
                  {CURRENCIES.map((c) => <option key={c}>{c}</option>)}
                </select>
                <input
                  type="number"
                  value={form.minPrice}
                  onChange={(e) => set("minPrice", e.target.value)}
                  placeholder="Min"
                  className="flex-1 border border-gray-200 rounded-xl py-2.5 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-[#0071CE]/30 focus:border-[#0071CE]"
                />
                <span className="text-gray-400 font-medium">—</span>
                <input
                  type="number"
                  value={form.maxPrice}
                  onChange={(e) => set("maxPrice", e.target.value)}
                  placeholder="Max"
                  className="flex-1 border border-gray-200 rounded-xl py-2.5 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-[#0071CE]/30 focus:border-[#0071CE]"
                />
              </div>
            </div>
            <div>
              <label className="block text-sm font-bold text-gray-700 mb-2">Location</label>
              <select
                value={form.location}
                onChange={(e) => set("location", e.target.value)}
                className="w-full border border-gray-200 rounded-xl py-2.5 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-[#0071CE]/30 focus:border-[#0071CE] bg-white"
              >
                {LOCATIONS.map((l) => <option key={l}>{l}</option>)}
              </select>
            </div>
          </div>

          {/* ── Sort + Seller + Filters ── */}
          <div className="p-6 border-b border-gray-100 grid grid-cols-1 sm:grid-cols-2 gap-5">
            <div>
              <label className="block text-sm font-bold text-gray-700 mb-2">Sort Results By</label>
              <select
                value={form.sort}
                onChange={(e) => set("sort", e.target.value)}
                className="w-full border border-gray-200 rounded-xl py-2.5 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-[#0071CE]/30 focus:border-[#0071CE] bg-white"
              >
                {SORT_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm font-bold text-gray-700 mb-2">Seller Username</label>
              <input
                type="text"
                value={form.seller}
                onChange={(e) => set("seller", e.target.value)}
                placeholder="Search by seller name…"
                className="w-full border border-gray-200 rounded-xl py-2.5 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-[#0071CE]/30 focus:border-[#0071CE]"
              />
            </div>
          </div>

          {/* ── Extra toggles ── */}
          <div className="p-6 border-b border-gray-100 flex flex-wrap gap-6">
            <label className="flex items-center gap-2.5 cursor-pointer group">
              <div
                onClick={() => set("freeDelivery", !form.freeDelivery)}
                className={`w-10 h-5.5 rounded-full relative transition-colors ${form.freeDelivery ? "bg-[#0071CE]" : "bg-gray-200"}`}
              >
                <div className={`absolute top-0.5 w-4.5 h-4.5 bg-white rounded-full shadow transition-all ${form.freeDelivery ? "left-5" : "left-0.5"}`} />
              </div>
              <span className="text-sm text-gray-700 font-medium">Free delivery only</span>
            </label>
            <label className="flex items-center gap-2.5 cursor-pointer">
              <div
                onClick={() => set("featuredOnly", !form.featuredOnly)}
                className={`w-10 h-5.5 rounded-full relative transition-colors ${form.featuredOnly ? "bg-[#0071CE]" : "bg-gray-200"}`}
              >
                <div className={`absolute top-0.5 w-4.5 h-4.5 bg-white rounded-full shadow transition-all ${form.featuredOnly ? "left-5" : "left-0.5"}`} />
              </div>
              <span className="text-sm text-gray-700 font-medium">Featured listings only</span>
            </label>
          </div>

          {/* ── Submit ── */}
          <div className="p-6 flex items-center justify-between">
            <button
              type="button"
              onClick={() => { setForm({ q: "", category: "", condition: "Any Condition", type: "All Types", minPrice: "", maxPrice: "", currency: "AED", location: "All Locations", sort: "newest", seller: "", freeDelivery: false, featuredOnly: false }); setCfValues({}); }}
              className="text-sm text-gray-500 hover:text-gray-700 transition-colors font-medium"
            >
              Clear all filters
            </button>
            <button
              type="submit"
              className="flex items-center gap-2 bg-[#0071CE] hover:bg-[#005ea8] text-white px-8 py-3 rounded-xl font-bold text-sm transition-colors shadow-md hover:shadow-lg"
            >
              <Search size={16} />
              Search Now
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
