import { useState } from "react";
import { getCategorySchema } from "@/lib/categoryFields";
import { MapPin, Loader2 } from "lucide-react";
import { useLocation } from "@/hooks/useLocation";

const CATEGORIES = [
  { label: "All Categories", value: "" },
  { label: "🚗 Vehicles", value: "vehicles" },
  { label: "🏠 Real Estate", value: "real-estate" },
  { label: "📱 Electronics", value: "electronics" },
  { label: "👕 Clothing", value: "clothing" },
  { label: "🛋️ Furniture", value: "furniture" },
  { label: "💎 Jewelry", value: "jewelry" },
  { label: "🔧 Tools", value: "tools" },
  { label: "🎮 Gaming", value: "gaming" },
];

const CONDITIONS = [
  { label: "Any Condition", value: "" },
  { label: "New", value: "new" },
  { label: "Like New", value: "like-new" },
  { label: "Good", value: "good" },
  { label: "Fair", value: "fair" },
];

export const CITIES = [
  { label: "All Cities", value: "" },
  { label: "Dubai", value: "Dubai" },
  { label: "Abu Dhabi", value: "Abu Dhabi" },
  { label: "Sharjah", value: "Sharjah" },
  { label: "Riyadh", value: "Riyadh" },
  { label: "Jeddah", value: "Jeddah" },
  { label: "Kuwait City", value: "Kuwait City" },
  { label: "Doha", value: "Doha" },
  { label: "Manama", value: "Manama" },
  { label: "Muscat", value: "Muscat" },
  { label: "Amman", value: "Amman" },
  { label: "Beirut", value: "Beirut" },
];

interface FiltersPanelProps {
  category: string;
  condition: string;
  minPrice: string;
  maxPrice: string;
  attrFilters?: Record<string, string>;
  city: string;
  nearMe: boolean;
  lat: string;
  lng: string;
  radius: number;
  onChange: (key: string, value: string) => void;
  onAttrFilterChange?: (key: string, value: string) => void;
  onNearMe: (lat: number, lng: number, city: string) => void;
  onClearNearMe: () => void;
}

export function FiltersPanel({
  category,
  condition,
  minPrice,
  maxPrice,
  attrFilters = {},
  city,
  nearMe,
  radius,
  onChange,
  onAttrFilterChange,
  onNearMe,
  onClearNearMe,
}: FiltersPanelProps) {
  const schema = getCategorySchema(category);
  const filterableFields = schema?.fields.filter((f) => !!f.filterType) ?? [];
  const { detectLocation, loading } = useLocation();

  const handleNearMe = async () => {
    if (nearMe) {
      onClearNearMe();
      return;
    }
    const loc = await detectLocation();
    if (loc) {
      onNearMe(loc.lat, loc.lon, loc.city);
    }
  };

  return (
    <aside className="w-56 shrink-0 hidden md:block">
      <div className="bg-white rounded-xl shadow-sm p-4 sticky top-20 space-y-5">
        <div>
          <h3 className="font-semibold text-gray-800 mb-2 text-sm">Location</h3>
          <button
            onClick={handleNearMe}
            disabled={loading}
            className={`w-full flex items-center justify-center gap-1.5 text-sm px-3 py-2 rounded-lg font-medium transition-colors mb-2 ${
              nearMe
                ? "bg-[#0071CE] text-white"
                : "border border-[#0071CE] text-[#0071CE] hover:bg-blue-50"
            }`}
          >
            {loading ? (
              <Loader2 size={13} className="animate-spin" />
            ) : (
              <MapPin size={13} />
            )}
            {nearMe ? "Near Me ✓" : "Near Me"}
          </button>

          {nearMe && (
            <div className="mb-2">
              <label className="text-xs text-gray-500 mb-1 block">Radius: {radius} km</label>
              <input
                type="range"
                min={10}
                max={100}
                step={10}
                value={radius}
                onChange={(e) => onChange("radius", e.target.value)}
                className="w-full accent-[#0071CE]"
              />
              <div className="flex justify-between text-xs text-gray-400">
                <span>10 km</span>
                <span>100 km</span>
              </div>
            </div>
          )}

          <div>
            <label className="text-xs text-gray-500 mb-1 block">City / Emirate</label>
            <select
              value={city}
              onChange={(e) => onChange("city", e.target.value)}
              className="w-full border rounded-lg px-2 py-1.5 text-sm text-gray-700 outline-none focus:ring-1 focus:ring-[#0071CE]"
            >
              {CITIES.map((c) => (
                <option key={c.value} value={c.value}>
                  {c.label}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div>
          <h3 className="font-semibold text-gray-800 mb-2 text-sm">Category</h3>
          <div className="space-y-1">
            {CATEGORIES.map((c) => (
              <button
                key={c.value}
                onClick={() => onChange("category", c.value)}
                className={`w-full text-left text-sm px-2 py-1.5 rounded-lg transition-colors ${
                  category === c.value
                    ? "bg-[#0071CE] text-white font-medium"
                    : "text-gray-600 hover:bg-gray-100"
                }`}
              >
                {c.label}
              </button>
            ))}
          </div>
        </div>

        {filterableFields.length > 0 && (
          <div>
            <h3 className="font-semibold text-gray-800 mb-2 text-sm">
              {schema!.slug.charAt(0).toUpperCase() + schema!.slug.slice(1).replace("-", " ")} Filters
            </h3>
            <div className="space-y-3">
              {filterableFields.map((field) => {
                if (field.filterType === "select" && field.filterOptions) {
                  const current = attrFilters[field.name] ?? "";
                  return (
                    <div key={field.name}>
                      <p className="text-xs text-gray-500 mb-1">{field.label}</p>
                      <div className="flex flex-wrap gap-1">
                        <button
                          onClick={() => onAttrFilterChange?.(field.name, "")}
                          className={`text-xs px-2 py-1 rounded-full border transition-colors ${
                            current === ""
                              ? "bg-[#0071CE] text-white border-[#0071CE]"
                              : "text-gray-600 border-gray-200 hover:border-[#0071CE] hover:text-[#0071CE]"
                          }`}
                        >
                          Any
                        </button>
                        {field.filterOptions.map((opt) => (
                          <button
                            key={opt.value}
                            onClick={() => onAttrFilterChange?.(field.name, opt.value)}
                            className={`text-xs px-2 py-1 rounded-full border transition-colors ${
                              current === opt.value
                                ? "bg-[#0071CE] text-white border-[#0071CE]"
                                : "text-gray-600 border-gray-200 hover:border-[#0071CE] hover:text-[#0071CE]"
                            }`}
                          >
                            {opt.label}
                          </button>
                        ))}
                      </div>
                    </div>
                  );
                }

                if (field.filterType === "number_range") {
                  const minKey = `min_${field.name}`;
                  const maxKey = `max_${field.name}`;
                  return (
                    <div key={field.name}>
                      <p className="text-xs text-gray-500 mb-1">
                        {field.label}{field.unit ? ` (${field.unit})` : ""}
                      </p>
                      <div className="flex items-center gap-1">
                        <input
                          type="number"
                          placeholder="Min"
                          value={attrFilters[minKey] ?? ""}
                          onChange={(e) => onAttrFilterChange?.(minKey, e.target.value)}
                          className="w-full border rounded-lg px-2 py-1 text-xs outline-none focus:ring-1 focus:ring-[#0071CE]"
                        />
                        <span className="text-gray-400 text-xs">—</span>
                        <input
                          type="number"
                          placeholder="Max"
                          value={attrFilters[maxKey] ?? ""}
                          onChange={(e) => onAttrFilterChange?.(maxKey, e.target.value)}
                          className="w-full border rounded-lg px-2 py-1 text-xs outline-none focus:ring-1 focus:ring-[#0071CE]"
                        />
                      </div>
                    </div>
                  );
                }

                if (field.filterType === "text") {
                  return (
                    <div key={field.name}>
                      <p className="text-xs text-gray-500 mb-1">{field.label}</p>
                      <input
                        type="text"
                        placeholder={`Filter by ${field.label.toLowerCase()}`}
                        value={attrFilters[field.name] ?? ""}
                        onChange={(e) => onAttrFilterChange?.(field.name, e.target.value)}
                        className="w-full border rounded-lg px-2 py-1.5 text-xs outline-none focus:ring-1 focus:ring-[#0071CE]"
                      />
                    </div>
                  );
                }

                return null;
              })}
            </div>
          </div>
        )}

        <div>
          <h3 className="font-semibold text-gray-800 mb-2 text-sm">Condition</h3>
          <div className="space-y-1">
            {CONDITIONS.map((c) => (
              <button
                key={c.value}
                onClick={() => onChange("condition", c.value)}
                className={`w-full text-left text-sm px-2 py-1.5 rounded-lg transition-colors ${
                  condition === c.value
                    ? "bg-[#0071CE] text-white font-medium"
                    : "text-gray-600 hover:bg-gray-100"
                }`}
              >
                {c.label}
              </button>
            ))}
          </div>
        </div>

        <div>
          <h3 className="font-semibold text-gray-800 mb-2 text-sm">Price Range (AED)</h3>
          <div className="flex items-center gap-2">
            <input
              type="number"
              placeholder="Min"
              value={minPrice}
              onChange={(e) => onChange("min_price", e.target.value)}
              className="w-full border rounded-lg px-2 py-1.5 text-sm outline-none focus:ring-1 focus:ring-[#0071CE]"
            />
            <span className="text-gray-400">—</span>
            <input
              type="number"
              placeholder="Max"
              value={maxPrice}
              onChange={(e) => onChange("max_price", e.target.value)}
              className="w-full border rounded-lg px-2 py-1.5 text-sm outline-none focus:ring-1 focus:ring-[#0071CE]"
            />
          </div>
        </div>

        <div>
          <h3 className="font-semibold text-gray-800 mb-2 text-sm">Listing Type</h3>
          <div className="space-y-1">
            {[
              { label: "All Types", value: "" },
              { label: "🔨 Auction", value: "auction" },
              { label: "⚡ Buy Now", value: "fixed" },
            ].map((t) => (
              <button
                key={t.value}
                onClick={() => onChange("type", t.value)}
                className="w-full text-left text-sm px-2 py-1.5 rounded-lg text-gray-600 hover:bg-gray-100 transition-colors"
              >
                {t.label}
              </button>
            ))}
          </div>
        </div>
      </div>
    </aside>
  );
}
