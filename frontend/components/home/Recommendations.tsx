'use client'
import { useRouter } from 'next/navigation';
import { useState, useEffect } from "react";
import { Sparkles, MapPin, Tag, Star, ChevronRight, RefreshCw } from "lucide-react";
import { getRecommendations, loadPreferences, ScoredListing } from "@/lib/recommendations";

const TAG_STYLES: Record<string, string> = {
  "Top Pick": "bg-[#0071CE] text-white",
  "Near You": "bg-emerald-500 text-white",
  "Similar Category": "bg-violet-500 text-white",
  "Price Match": "bg-amber-500 text-white",
  "Trending": "bg-rose-500 text-white",
  "New Arrival": "bg-teal-500 text-white",
};

function RecommendCard({ item, onClick }: { item: ScoredListing; onClick: () => void }) {
  return (
    <div
      onClick={onClick}
      className="bg-white rounded-2xl overflow-hidden shadow-sm border border-gray-100 hover:shadow-md hover:border-[#0071CE]/30 transition-all cursor-pointer group flex-shrink-0 w-52 sm:w-56"
    >
      <div className="relative">
        <img
          src={item.image}
          alt={item.title}
          className="w-full h-36 object-cover group-hover:scale-[1.02] transition-transform duration-300"
        />
        <div className={`absolute top-2 left-2 text-xs font-semibold px-2 py-0.5 rounded-full ${TAG_STYLES[item.reason_tag] || "bg-gray-600 text-white"}`}>
          {item.reason_tag}
        </div>
      </div>

      <div className="p-3">
        <p className="text-sm font-semibold text-gray-800 line-clamp-2 leading-snug">{item.title}</p>
        <p className="text-[#0071CE] font-bold mt-1 text-sm">
          {item.currency} {item.price.toLocaleString()}
        </p>
        <div className="flex items-center gap-2 mt-1.5 text-xs text-gray-400">
          <span className="flex items-center gap-0.5">
            <MapPin className="w-3 h-3" /> {item.location.split(",")[0]}
          </span>
          <span className="flex items-center gap-0.5 ml-auto">
            <Star className="w-3 h-3 fill-[#FFC220] text-[#FFC220]" /> {item.rating}
          </span>
        </div>
        <p className="text-[10px] text-gray-400 mt-1.5 line-clamp-2 italic">{item.ai_reason}</p>
      </div>
    </div>
  );
}

export function Recommendations() {
  const router = useRouter();
  const [recs, setRecs] = useState<ScoredListing[]>([]);
  const [refreshKey, setRefreshKey] = useState(0);
  const [hasPrefs, setHasPrefs] = useState(false);

  useEffect(() => {
    const prefs = loadPreferences();
    const hasBehavior = Object.keys(prefs.viewedCategories).length > 0;
    setHasPrefs(hasBehavior);
    setRecs(getRecommendations({ limit: 10 }));
  }, [refreshKey]);

  if (recs.length === 0) return null;

  return (
    <section>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-[#0071CE] to-violet-600 flex items-center justify-center">
            <Sparkles className="w-4 h-4 text-white" />
          </div>
          <div>
            <h2 className="text-lg font-bold text-gray-900">
              {hasPrefs ? "Recommended for You" : "Top Picks in the GCC"}
            </h2>
            <p className="text-xs text-gray-400 flex items-center gap-1">
              <Sparkles className="w-3 h-3 text-[#0071CE]" />
              {hasPrefs
                ? "Personalized based on your browsing activity"
                : "Curated by AI — browse more to personalize"}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setRefreshKey(k => k + 1)}
            className="p-1.5 rounded-lg hover:bg-gray-100 text-gray-400 hover:text-[#0071CE] transition-colors"
            title="Refresh recommendations"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
          <button
            onClick={() => router.push("/listings")}
            className="flex items-center gap-1 text-sm text-[#0071CE] font-semibold hover:underline"
          >
            See all <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Category preference pills */}
      {hasPrefs && (() => {
        const prefs = loadPreferences();
        const topCats = Object.entries(prefs.viewedCategories)
          .sort((a, b) => b[1] - a[1])
          .slice(0, 3);
        return topCats.length > 0 ? (
          <div className="flex gap-2 mb-3 flex-wrap">
            <span className="text-xs text-gray-400 self-center">Your interests:</span>
            {topCats.map(([cat]) => (
              <button
                key={cat}
                onClick={() => router.push(`/listings?category=${encodeURIComponent(cat)}`)}
                className="text-xs bg-[#0071CE]/10 text-[#0071CE] px-3 py-1 rounded-full font-medium hover:bg-[#0071CE]/20 transition-colors flex items-center gap-1"
              >
                <Tag className="w-3 h-3" /> {cat}
              </button>
            ))}
          </div>
        ) : null;
      })()}

      {/* Horizontal scroll row */}
      <div className="flex gap-3 overflow-x-auto pb-2 scrollbar-hide -mx-1 px-1">
        {recs.map(item => (
          <RecommendCard
            key={item.id}
            item={item}
            onClick={() => router.push(`/listings/${item.id}`)}
          />
        ))}
      </div>

      {/* AI powered footer */}
      <div className="mt-3 flex items-center gap-1.5 text-xs text-gray-400">
        <Sparkles className="w-3.5 h-3.5 text-[#0071CE]" />
        <span>AI-powered recommendations · Updates as you browse</span>
      </div>
    </section>
  );
}
