import { useState } from "react";
import { useParams, useLocation } from "wouter";
import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";
import { ListingCard } from "@/components/listings/ListingCard";
import { LoadingGrid } from "@/components/ui/LoadingGrid";
import { useAuthStore } from "@/store/auth";
import { SellerReviews } from "@/components/reviews/SellerReviews";
import { ChatPanel } from "@/components/chat/ChatPanel";
import {
  Star,
  MessageCircle,
  ShieldCheck,
  Package,
  ChevronLeft,
  Calendar,
  UserX,
} from "lucide-react";

function formatMemberSince(dateStr: string) {
  try {
    return new Date(dateStr).toLocaleDateString("en-AE", { year: "numeric", month: "long" });
  } catch {
    return dateStr;
  }
}

export default function SellerPage() {
  const params = useParams<{ id: string }>();
  const sellerId = params.id;
  const [, navigate] = useLocation();
  const { isAuthenticated } = useAuthStore();
  const [chatOpen, setChatOpen] = useState(false);

  const {
    data: seller,
    isLoading: sellerLoading,
    error: sellerError,
  } = useQuery({
    queryKey: ["seller", sellerId],
    queryFn: () =>
      api.get(`/users/${sellerId}/profile`).then((r) => {
        const d = r.data.data;
        if (!d) return d;
        // Normalize backend field names to UI expectations
        return {
          ...d,
          avatar: d.avatar ?? d.avatar_url ?? null,
          verified: d.verified ?? d.is_verified ?? false,
          city: d.city ?? (typeof d.location === "string" ? d.location : null),
          listings_sold: d.listings_sold ?? d.sold_count ?? 0,
          member_since: d.member_since ?? d.created_at ?? null,
        };
      }),
    retry: 1,
  });

  const { data: listings, isLoading: listingsLoading } = useQuery({
    queryKey: ["seller-listings", sellerId],
    queryFn: () =>
      api.get(`/listings?seller_id=${sellerId}&per_page=24`).then((r) => r.data.data ?? []),
    enabled: !sellerError,
    retry: 1,
  });

  const handleMessage = () => {
    if (!isAuthenticated) {
      navigate("/login");
      return;
    }
    setChatOpen(true);
  };

  if (sellerLoading) {
    return (
      <div className="max-w-6xl mx-auto px-4 py-10">
        <div className="h-40 bg-gray-100 rounded-2xl animate-pulse mb-6" />
        <LoadingGrid count={6} />
      </div>
    );
  }

  if (sellerError || !seller) {
    return (
      <div className="max-w-6xl mx-auto px-4 py-8">
        <button
          onClick={() => navigate("/listings")}
          className="flex items-center gap-1.5 text-gray-500 hover:text-[#0071CE] text-sm mb-6 transition-colors"
        >
          <ChevronLeft size={16} /> Back to listings
        </button>
        <div className="text-center py-20 text-gray-400">
          <UserX size={48} className="mx-auto mb-4 opacity-40" />
          <p className="text-lg font-semibold">Seller not found</p>
          <p className="text-sm mt-1">This seller may no longer be active.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-6xl mx-auto px-4 py-8">
      <button
        onClick={() => navigate("/listings")}
        className="flex items-center gap-1.5 text-gray-500 hover:text-[#0071CE] text-sm mb-6 transition-colors"
      >
        <ChevronLeft size={16} /> Back to listings
      </button>

      <div className="bg-white rounded-2xl shadow-sm p-6 mb-6">
        <div className="flex flex-col sm:flex-row items-start sm:items-center gap-5">
          <div className="w-20 h-20 rounded-full bg-[#0071CE] flex items-center justify-center text-white text-2xl font-bold shrink-0 overflow-hidden">
            {seller.avatar ? (
              <img
                src={seller.avatar as string}
                alt={seller.name as string}
                className="w-full h-full object-cover"
              />
            ) : (
              <span>{typeof seller.name === "string" ? seller.name[0] : "S"}</span>
            )}
          </div>

          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <h1 className="text-2xl font-bold text-gray-900">{seller.name as string}</h1>
              {(seller.verified || seller.is_verified) && (
                <span className="flex items-center gap-1 bg-green-50 text-green-700 text-xs font-semibold px-2 py-1 rounded-full">
                  <ShieldCheck size={12} /> Verified
                </span>
              )}
            </div>

            {(seller.city || seller.location) && (
              <p className="text-sm text-gray-500 mt-0.5">
                📍 {(seller.city ?? seller.location) as string}
              </p>
            )}

            {seller.bio && (
              <p className="text-sm text-gray-600 mt-2 leading-relaxed max-w-2xl">
                {seller.bio as string}
              </p>
            )}
          </div>

          <button
            onClick={handleMessage}
            className="flex items-center gap-2 bg-[#0071CE] hover:bg-[#005BA1] text-white font-semibold px-5 py-2.5 rounded-xl transition-colors text-sm shrink-0"
          >
            <MessageCircle size={15} /> Message Seller
          </button>
        </div>

        <div className="mt-5 pt-5 border-t grid grid-cols-2 sm:grid-cols-4 gap-4 text-center">
          <div>
            <p className="text-2xl font-bold text-gray-900">
              {typeof seller.rating === "number" ? seller.rating.toFixed(1) : "—"}
            </p>
            <p className="text-xs text-gray-500 flex items-center justify-center gap-1 mt-0.5">
              <Star size={11} fill="#FFC220" className="text-[#FFC220]" /> Rating
            </p>
          </div>
          <div>
            <p className="text-2xl font-bold text-gray-900">
              {typeof seller.review_count === "number"
                ? seller.review_count.toLocaleString()
                : "0"}
            </p>
            <p className="text-xs text-gray-500 mt-0.5">Reviews</p>
          </div>
          <div>
            <p className="text-2xl font-bold text-gray-900">
              {typeof seller.listings_sold === "number"
                ? seller.listings_sold.toLocaleString()
                : typeof seller.sold_count === "number"
                ? seller.sold_count.toLocaleString()
                : "0"}
            </p>
            <p className="text-xs text-gray-500 flex items-center justify-center gap-1 mt-0.5">
              <Package size={11} /> Sold
            </p>
          </div>
          <div>
            <p className="text-sm font-semibold text-gray-900 mt-1">
              {seller.member_since
                ? formatMemberSince(seller.member_since as string)
                : seller.created_at
                ? formatMemberSince(seller.created_at as string)
                : "—"}
            </p>
            <p className="text-xs text-gray-500 flex items-center justify-center gap-1 mt-0.5">
              <Calendar size={11} /> Member Since
            </p>
          </div>
        </div>
      </div>

      <h2 className="text-lg font-bold text-gray-900 mb-4">
        Active Listings
        {Array.isArray(listings) && listings.length > 0 && (
          <span className="ml-2 text-sm text-gray-500 font-normal">({listings.length})</span>
        )}
      </h2>

      {listingsLoading ? (
        <LoadingGrid count={6} />
      ) : !Array.isArray(listings) || listings.length === 0 ? (
        <div className="text-center py-16 text-gray-400">
          <p className="text-4xl mb-3">📦</p>
          <p className="font-semibold">No active listings</p>
          <p className="text-sm mt-1">This seller has no active listings at the moment.</p>
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
          {(listings as any[]).map((listing) => (
            <ListingCard key={listing.id as string} listing={listing} />
          ))}
        </div>
      )}

      <SellerReviews
        sellerId={sellerId}
        sellerName={seller.name as string}
        rating={typeof seller.rating === "number" ? seller.rating : undefined}
        reviewCount={typeof seller.review_count === "number" ? seller.review_count : undefined}
      />

      {chatOpen && (
        <ChatPanel
          sellerId={sellerId}
          sellerName={seller.name as string}
          onClose={() => setChatOpen(false)}
        />
      )}
    </div>
  );
}
