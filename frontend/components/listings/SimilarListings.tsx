'use client'
import { useRouter } from 'next/navigation';
import { useEffect, useState } from "react";
import { Sparkles, MapPin, Tag, Star } from "lucide-react";
import { getSimilarListings, trackListing, ScoredListing, Listing } from "@/lib/recommendations";

const TAG_STYLES: Record<string, string> = {
  "Top Pick": "bg-[#0071CE] text-white",
  "Near You": "bg-emerald-500 text-white",
  "Similar Category": "bg-violet-500 text-white",
  "Price Match": "bg-amber-500 text-white",
  "Trending": "bg-rose-500 text-white",
  "New Arrival": "bg-teal-500 text-white",
};

function SimilarCard({ item, onClick }: { item: ScoredListing; onClick: () => void }) {
  return (
    <div
      onClick={onClick}
      className="bg-white rounded-2xl overflow-hidden shadow-sm border border-gray-100 hover:shadow-md hover:border-[#0071CE]/30 transition-all cursor-pointer group"
    >
      <div className="relative">
        <img
          src={item.image}
          alt={item.title}
          className="w-full h-40 object-cover group-hover:scale-[1.02] transition-transform duration-300"
        />
        <div className={`absolute top-2 left-2 text-xs font-semibold px-2 py-0.5 rounded-full ${TAG_STYLES[item.reason_tag] || "bg-gray-600 text-white"}`}>
          {item.reason_tag}
        </div>
        <div className="absolute top-2 right-2 bg-white/90 backdrop-blur-sm text-xs text-gray-600 px-2 py-0.5 rounded-full">
          {item.condition}
        </div>
      </div>
      <div className="p-3">
        <p className="text-sm font-semibold text-gray-800 line-clamp-2 leading-snug">{item.title}</p>
        <p className="text-[#0071CE] font-bold mt-1">
          {item.currency} {item.price.toLocaleString()}
        </p>
        <div className="flex items-center gap-2 mt-1.5 text-xs text-gray-400">
          <span className="flex items-center gap-0.5">
            <MapPin className="w-3 h-3" /> {item.location.split(",")[0]}
          </span>
          <span className="flex items-center gap-0.5">
            <Tag className="w-3 h-3" /> {item.category}
          </span>
          <span className="flex items-center gap-0.5 ml-auto">
            <Star className="w-3 h-3 fill-[#FFC220] text-[#FFC220]" /> {item.rating}
          </span>
        </div>
        <p className="text-[10px] text-gray-400 mt-1.5 line-clamp-1 italic">{item.ai_reason}</p>
      </div>
    </div>
  );
}

interface SimilarListingsProps {
  listing: Listing;
}

export function SimilarListings({ listing }: SimilarListingsProps) {
  const router = useRouter();
  const [similar, setSimilar] = useState<ScoredListing[]>([]);

  useEffect(() => {
    // Track this listing view and get similar items
    trackListing(listing);
    setSimilar(getSimilarListings(listing, 4));
  }, [listing.id]);

  if (similar.length === 0) return null;

  return (
    <section className="mt-10">
      <div className="flex items-center gap-2 mb-4">
        <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-[#0071CE] to-violet-600 flex items-center justify-center">
          <Sparkles className="w-4 h-4 text-white" />
        </div>
        <div>
          <h2 className="text-lg font-bold text-gray-900">Similar Listings</h2>
          <p className="text-xs text-gray-400 flex items-center gap-1">
            <Sparkles className="w-3 h-3 text-[#0071CE]" />
            AI-matched listings you might like
          </p>
        </div>
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        {similar.map(item => (
          <SimilarCard
            key={item.id}
            item={item}
            onClick={() => router.push(`/listings/${item.id}`)}
          />
        ))}
      </div>

      <div className="mt-3 flex items-center gap-1.5 text-xs text-gray-400">
        <Sparkles className="w-3.5 h-3.5 text-[#0071CE]" />
        <span>Personalized by Mnbarh AI · Based on category, price & your browsing</span>
      </div>
    </section>
  );
}
