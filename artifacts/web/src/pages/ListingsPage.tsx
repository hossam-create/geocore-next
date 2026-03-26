import { useState, useEffect } from "react";
import { useSearch, useLocation as useWouterLocation } from "wouter";
import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";
import { ListingCard } from "@/components/listings/ListingCard";
import { FiltersPanel } from "@/components/listings/FiltersPanel";
import { LoadingGrid } from "@/components/ui/LoadingGrid";
import { useLocation as useGeoLocation } from "@/hooks/useLocation";
import { SlidersHorizontal, X, MapPin, Loader2 } from "lucide-react";

const SORT_OPTIONS = [
  { label: "Newest First", value: "newest" },
  { label: "Oldest First", value: "oldest" },
  { label: "Price: Low to High", value: "price_asc" },
  { label: "Price: High to Low", value: "price_desc" },
];

const MOCK_LISTINGS = [
  { id: "1", title: "iPhone 15 Pro Max 256GB", price: 4200, currency: "AED", type: "fixed", city: "Dubai", condition: "new", category: { slug: "electronics" }, attributes: { brand: "Apple", storage: "256GB", ram: "8GB" } },
  { id: "2", title: "Toyota Camry 2023", price: 89000, current_bid: 75000, currency: "AED", type: "auction", city: "Abu Dhabi", auctionEndsAt: new Date(Date.now() + 7200000).toISOString(), category: { slug: "vehicles" }, attributes: { make: "Toyota", model: "Camry", year: 2023, mileage: 12000, color: "White" } },
  { id: "3", title: "MacBook Pro M3 Max", price: 9500, currency: "AED", type: "fixed", city: "Riyadh", condition: "new", category: { slug: "electronics" }, attributes: { brand: "Apple", storage: "1TB", ram: "36GB" } },
  { id: "4", title: "Chanel Classic Flap", price: 18000, current_bid: 14500, currency: "AED", type: "auction", city: "Kuwait City", auctionEndsAt: new Date(Date.now() + 3600000).toISOString(), category: { slug: "jewelry" }, attributes: { material: "Leather", brand: "Chanel" } },
  { id: "5", title: "Samsung 85\" QLED 8K TV", price: 12000, currency: "AED", type: "fixed", city: "Doha", condition: "new", category: { slug: "electronics" }, attributes: { brand: "Samsung", storage: "N/A" } },
  { id: "6", title: "Dubai JBR 2BR Apartment", price: 4200000, currency: "AED", type: "fixed", city: "Dubai", condition: "new", category: { slug: "real-estate" }, attributes: { bedrooms: 2, bathrooms: 2, area: 120, furnished: "yes", floor: 8 } },
  { id: "7", title: "Rolex Datejust 41mm", price: 38000, currency: "AED", type: "auction", city: "Manama", auctionEndsAt: new Date(Date.now() + 86400000).toISOString(), category: { slug: "jewelry" }, attributes: { material: "Stainless Steel", brand: "Rolex" } },
  { id: "8", title: "PlayStation 5 Slim Bundle", price: 2100, currency: "AED", type: "fixed", city: "Muscat", condition: "new", category: { slug: "gaming" }, attributes: { platform: "PlayStation 5", brand: "Sony", storage: "1TB" } },
  { id: "9", title: "DJI Mavic 3 Pro", price: 7200, currency: "AED", type: "fixed", city: "Dubai", condition: "new", category: { slug: "electronics" }, attributes: { brand: "DJI", storage: "N/A" } },
  { id: "10", title: "BMW 3 Series 2022", price: 180000, currency: "AED", type: "auction", city: "Dubai", auctionEndsAt: new Date(Date.now() + 172800000).toISOString(), category: { slug: "vehicles" }, attributes: { make: "BMW", model: "3 Series", year: 2022, mileage: 35000, color: "Black" } },
  { id: "11", title: "Louis Vuitton Neverfull MM", price: 5500, currency: "AED", type: "fixed", city: "Dubai", condition: "like-new", category: { slug: "clothing" }, attributes: { brand: "Louis Vuitton", size: "MM", color: "Brown" } },
  { id: "12", title: "Nintendo Switch OLED", price: 1200, currency: "AED", type: "fixed", city: "Dubai", condition: "new", category: { slug: "gaming" }, attributes: { platform: "Nintendo Switch", brand: "Nintendo", storage: "64GB" } },
];

interface Filters {
  q: string;
  category: string;
  condition: string;
  min_price: string;
  max_price: string;
  type: string;
  sort: string;
  city: string;
  nearMe: boolean;
  lat: string;
  lng: string;
  radius: number;
}

function buildUrlParams(filters: Filters): string {
  const p = new URLSearchParams();
  if (filters.q) p.set("q", filters.q);
  if (filters.category) p.set("category", filters.category);
  if (filters.condition) p.set("condition", filters.condition);
  if (filters.min_price) p.set("min_price", filters.min_price);
  if (filters.max_price) p.set("max_price", filters.max_price);
  if (filters.type) p.set("type", filters.type);
  if (filters.sort && filters.sort !== "newest") p.set("sort", filters.sort);
  if (filters.city) p.set("city", filters.city);
  if (filters.nearMe && filters.lat && filters.lng) {
    p.set("lat", filters.lat);
    p.set("lng", filters.lng);
    p.set("radius", String(filters.radius));
  }
  return p.toString();
}

function ActiveFilterChips({
  filters,
  onRemove,
}: {
  filters: Filters;
  onRemove: (key: string) => void;
}) {
  const chips: { key: string; label: string }[] = [];

  if (filters.nearMe && filters.lat) {
    chips.push({ key: "nearMe", label: `📍 Near Me (${filters.radius} km)` });
  }
  if (filters.city) {
    chips.push({ key: "city", label: `📍 ${filters.city}` });
  }
  if (filters.category) {
    chips.push({ key: "category", label: `Category: ${filters.category}` });
  }
  if (filters.condition) {
    chips.push({ key: "condition", label: `Condition: ${filters.condition}` });
  }
  if (filters.type) {
    chips.push({ key: "type", label: `Type: ${filters.type === "auction" ? "Auction" : "Buy Now"}` });
  }
  if (filters.min_price || filters.max_price) {
    const label =
      filters.min_price && filters.max_price
        ? `AED ${filters.min_price}–${filters.max_price}`
        : filters.min_price
        ? `AED ${filters.min_price}+`
        : `Up to AED ${filters.max_price}`;
    chips.push({ key: "price", label });
  }

  if (chips.length === 0) return null;

  return (
    <div className="flex flex-wrap gap-2 mb-4">
      {chips.map((chip) => (
        <span
          key={chip.key}
          className="flex items-center gap-1.5 bg-blue-50 text-[#0071CE] text-xs font-medium px-3 py-1.5 rounded-full"
        >
          {chip.label}
          <button
            onClick={() => onRemove(chip.key)}
            className="text-[#0071CE] hover:text-blue-800 transition-colors"
            aria-label={`Remove ${chip.label} filter`}
          >
            <X size={11} />
          </button>
        </span>
      ))}
      {chips.length > 1 && (
        <button
          onClick={() => onRemove("all")}
          className="text-xs text-gray-500 hover:text-gray-700 px-2 py-1 underline"
        >
          Clear all
        </button>
      )}
    </div>
  );
}

export default function ListingsPage() {
  const searchStr = useSearch();
  const [, navigate] = useWouterLocation();
  const params = new URLSearchParams(searchStr);

  const [filters, setFilters] = useState<Filters>({
    q: params.get("q") || "",
    category: params.get("category") || "",
    condition: params.get("condition") || "",
    min_price: params.get("min_price") || "",
    max_price: params.get("max_price") || "",
    type: params.get("type") || "",
    sort: params.get("sort") || "newest",
    city: params.get("city") || "",
    nearMe: !!(params.get("lat") && params.get("lng")),
    lat: params.get("lat") || "",
    lng: params.get("lng") || "",
    radius: Number(params.get("radius")) || 50,
  });

  const [attrFilters, setAttrFilters] = useState<Record<string, string>>({});

  const [showMobileFilters, setShowMobileFilters] = useState(false);
  const { detectLocation, loading: locationLoading } = useGeoLocation();

  useEffect(() => {
    setFilters((f) => ({
      ...f,
      q: params.get("q") || "",
      category: params.get("category") || "",
      city: params.get("city") || f.city,
    }));
    setAttrFilters({});
  }, [searchStr]);

  useEffect(() => {
    const urlParams = buildUrlParams(filters);
    const currentSearch = searchStr.startsWith("?") ? searchStr.slice(1) : searchStr;
    if (urlParams !== currentSearch) {
      navigate(`/listings${urlParams ? `?${urlParams}` : ""}`, { replace: true });
    }
  }, [filters]);

  const queryKey = ["listings", filters, attrFilters];
  const { data: listings, isLoading, error } = useQuery({
    queryKey,
    queryFn: () => {
      const p = new URLSearchParams();
      if (filters.q) p.set("search", filters.q);
      if (filters.category) p.set("category", filters.category);
      if (filters.condition) p.set("condition", filters.condition);
      if (filters.min_price) p.set("min_price", filters.min_price);
      if (filters.max_price) p.set("max_price", filters.max_price);
      if (filters.type) p.set("type", filters.type);
      if (filters.city) p.set("city", filters.city);
      if (filters.nearMe && filters.lat && filters.lng) {
        p.set("lat", filters.lat);
        p.set("lng", filters.lng);
        p.set("radius", String(filters.radius));
      }
      p.set("sort", filters.sort);
      p.set("per_page", "24");
      Object.entries(attrFilters).forEach(([key, val]) => {
        if (val) p.set(`attr_${key}`, val);
      });
      return api.get(`/listings?${p.toString()}`).then((r) => r.data.data);
    },
    retry: false,
  });

  const handleFilterChange = (key: string, value: string) => {
    setFilters((f) => ({ ...f, [key]: key === "radius" ? Number(value) : value }));
    if (key === "category") {
      setAttrFilters({});
    }
  };

  const handleAttrFilterChange = (key: string, value: string) => {
    setAttrFilters((prev) => ({ ...prev, [key]: value }));
  };

  const handleNearMe = (lat: number, lng: number, city: string) => {
    setFilters((f) => ({
      ...f,
      nearMe: true,
      lat: String(lat),
      lng: String(lng),
      city: f.city || city,
    }));
  };

  const handleNearMeHeader = async () => {
    if (filters.nearMe) {
      handleClearNearMe();
      return;
    }
    const loc = await detectLocation();
    if (loc) {
      handleNearMe(loc.lat, loc.lon, loc.city);
    }
  };

  const handleClearNearMe = () => {
    setFilters((f) => ({
      ...f,
      nearMe: false,
      lat: "",
      lng: "",
      radius: 50,
    }));
  };

  const handleRemoveChip = (key: string) => {
    if (key === "all") {
      setFilters((f) => ({
        ...f,
        category: "",
        condition: "",
        min_price: "",
        max_price: "",
        type: "",
        city: "",
        nearMe: false,
        lat: "",
        lng: "",
        radius: 50,
      }));
      setAttrFilters({});
    } else if (key === "nearMe") {
      handleClearNearMe();
    } else if (key === "price") {
      setFilters((f) => ({ ...f, min_price: "", max_price: "" }));
    } else if (key.startsWith("attr_")) {
      const attrKey = key.replace("attr_", "");
      setAttrFilters((prev) => {
        const next = { ...prev };
        delete next[attrKey];
        return next;
      });
    } else {
      setFilters((f) => ({ ...f, [key]: "" }));
    }
  };

  const displayListings = (listings?.length ? listings : MOCK_LISTINGS).filter((l: any) => {
    if (filters.q && !l.title.toLowerCase().includes(filters.q.toLowerCase())) return false;
    if (filters.category && l.category?.slug !== filters.category) return false;
    if (filters.city && l.city && !l.city.toLowerCase().includes(filters.city.toLowerCase())) return false;
    return true;
  });

  const pageTitle = filters.category
    ? `${filters.category.charAt(0).toUpperCase() + filters.category.slice(1)} Listings`
    : filters.q
    ? `Results for "${filters.q}"`
    : filters.nearMe
    ? "Listings Near You"
    : filters.city
    ? `Listings in ${filters.city}`
    : "All Listings";

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{pageTitle}</h1>
          <p className="text-sm text-gray-500 mt-1">
            {displayListings.length.toLocaleString()} items found
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handleNearMeHeader}
            disabled={locationLoading}
            className={`flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
              filters.nearMe
                ? "bg-[#0071CE] text-white"
                : "border border-gray-200 text-gray-700 hover:border-[#0071CE] hover:text-[#0071CE]"
            }`}
          >
            {locationLoading ? (
              <Loader2 size={13} className="animate-spin" />
            ) : (
              <MapPin size={13} />
            )}
            <span className="hidden sm:inline">Near Me</span>
          </button>
          <select
            value={filters.sort}
            onChange={(e) => handleFilterChange("sort", e.target.value)}
            className="border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-700 outline-none focus:ring-1 focus:ring-[#0071CE]"
          >
            {SORT_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </select>
          <button
            onClick={() => setShowMobileFilters(!showMobileFilters)}
            className="md:hidden flex items-center gap-1.5 border border-gray-200 rounded-lg px-3 py-2 text-sm"
          >
            <SlidersHorizontal size={15} /> Filters
          </button>
        </div>
      </div>

      <ActiveFilterChips filters={filters} onRemove={handleRemoveChip} />

      <div className="flex gap-6">
        <FiltersPanel
          category={filters.category}
          condition={filters.condition}
          minPrice={filters.min_price}
          maxPrice={filters.max_price}
          city={filters.city}
          nearMe={filters.nearMe}
          lat={filters.lat}
          lng={filters.lng}
          radius={filters.radius}
          attrFilters={attrFilters}
          onChange={handleFilterChange}
          onAttrFilterChange={handleAttrFilterChange}
          onNearMe={handleNearMe}
          onClearNearMe={handleClearNearMe}
        />

        <div className="flex-1 min-w-0">
          {isLoading ? (
            <LoadingGrid count={12} />
          ) : displayListings.length === 0 ? (
            <div className="text-center py-20 text-gray-400">
              <p className="text-4xl mb-3">🔍</p>
              <p className="font-semibold text-lg">No listings found</p>
              <p className="text-sm mt-1">Try adjusting your filters</p>
            </div>
          ) : (
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
              {displayListings.map((listing: any) => (
                <ListingCard key={listing.id} listing={listing} />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
