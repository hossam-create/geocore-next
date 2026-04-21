'use client'
import Link from 'next/link';
import { useRef } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { ListingCard } from "@/components/listings/ListingCard";

interface ProductCarouselProps {
  title: string;
  icon?: string;
  listings: any[];
  viewAllHref?: string;
  badge?: string;
  badgeColor?: string;
  isLoading?: boolean;
}

function SkeletonCard() {
  return (
    <div className="bg-white rounded-2xl overflow-hidden border border-gray-100 animate-pulse shrink-0" style={{ width: 220 }}>
      <div className="bg-gray-200 h-44 w-full" />
      <div className="p-3 space-y-2">
        <div className="h-3 bg-gray-200 rounded w-16" />
        <div className="h-4 bg-gray-200 rounded w-full" />
        <div className="h-4 bg-gray-200 rounded w-3/4" />
        <div className="h-5 bg-gray-200 rounded w-24" />
        <div className="h-9 bg-gray-100 rounded-xl w-full mt-2" />
      </div>
    </div>
  );
}

export function ProductCarousel({
  title,
  icon,
  listings,
  viewAllHref = "/listings",
  badge,
  badgeColor = "#0071CE",
  isLoading,
}: ProductCarouselProps) {
  const scrollRef = useRef<HTMLDivElement>(null);

  const scroll = (dir: "left" | "right") => {
    const el = scrollRef.current;
    if (!el) return;
    const amount = el.clientWidth * 0.75;
    el.scrollBy({ left: dir === "right" ? amount : -amount, behavior: "smooth" });
  };

  return (
    <section>
      {/* Header row */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className="w-1 h-6 rounded-full" style={{ backgroundColor: badgeColor }} />
          <h2 className="text-xl font-extrabold text-gray-900 tracking-tight">
            {icon && <span className="mr-1.5">{icon}</span>}
            {title}
          </h2>
          {badge && (
            <span
              className="text-[10px] font-black px-2 py-0.5 rounded-full text-white uppercase tracking-wider"
              style={{ backgroundColor: badgeColor }}
            >
              {badge}
            </span>
          )}
        </div>
        <Link
          href={viewAllHref}
          className="text-sm font-semibold flex items-center gap-1 transition-colors hover:underline"
          style={{ color: badgeColor }}
        >
          See all <ChevronRight size={14} />
        </Link>
      </div>

      {/* Carousel container */}
      <div className="relative group">
        {/* Left arrow */}
        <button
          onClick={() => scroll("left")}
          className="absolute left-0 top-1/2 -translate-y-1/2 -translate-x-3 z-10 w-9 h-9 rounded-full bg-white shadow-lg border border-gray-200 flex items-center justify-center text-gray-600 hover:text-[#0071CE] hover:border-[#0071CE] transition-all opacity-0 group-hover:opacity-100 duration-200"
        >
          <ChevronLeft size={18} />
        </button>

        {/* Scrollable track */}
        <div
          ref={scrollRef}
          className="flex gap-4 overflow-x-auto scrollbar-none pb-2"
          style={{ scrollSnapType: "x mandatory" }}
        >
          {isLoading
            ? Array.from({ length: 6 }).map((_, i) => <SkeletonCard key={i} />)
            : listings.map((listing) => (
                <div
                  key={listing.id}
                  className="shrink-0"
                  style={{ width: 220, scrollSnapAlign: "start" }}
                >
                  <ListingCard listing={listing} />
                </div>
              ))}
        </div>

        {/* Right arrow */}
        <button
          onClick={() => scroll("right")}
          className="absolute right-0 top-1/2 -translate-y-1/2 translate-x-3 z-10 w-9 h-9 rounded-full bg-white shadow-lg border border-gray-200 flex items-center justify-center text-gray-600 hover:text-[#0071CE] hover:border-[#0071CE] transition-all opacity-0 group-hover:opacity-100 duration-200"
        >
          <ChevronRight size={18} />
        </button>
      </div>
    </section>
  );
}
