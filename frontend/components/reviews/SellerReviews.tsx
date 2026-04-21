'use client'
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { Star, Loader2, ThumbsUp, AlertCircle } from "lucide-react";
import { useAuthStore } from "@/store/auth";

interface Review {
  id: string;
  reviewer_id: string;
  reviewer_name: string;
  rating: number;
  comment: string;
  created_at: string;
}

interface SellerReviewsProps {
  sellerId: string;
  sellerName: string;
  rating?: number;
  reviewCount?: number;
}

function StarRating({ value, onChange }: { value: number; onChange?: (v: number) => void }) {
  const [hovered, setHovered] = useState(0);
  const display = onChange ? (hovered || value) : value;
  return (
    <div className="flex gap-0.5">
      {[1, 2, 3, 4, 5].map((s) => (
        <Star
          key={s}
          size={onChange ? 22 : 13}
          className={`transition-colors ${
            s <= display ? "text-[#FFC220]" : "text-gray-300"
          } ${onChange ? "cursor-pointer" : ""}`}
          fill={s <= display ? "#FFC220" : "none"}
          onMouseEnter={() => onChange && setHovered(s)}
          onMouseLeave={() => onChange && setHovered(0)}
          onClick={() => onChange && onChange(s)}
        />
      ))}
    </div>
  );
}

function ReviewForm({ sellerId, onSuccess }: { sellerId: string; onSuccess: () => void }) {
  const [rating, setRating] = useState(0);
  const [comment, setComment] = useState("");

  const { mutate, isPending, error } = useMutation({
    mutationFn: () =>
      api.post(`/users/${sellerId}/reviews`, { rating, comment }).then((r) => r.data),
    onSuccess: () => {
      setRating(0);
      setComment("");
      onSuccess();
    },
  });

  return (
    <div className="bg-blue-50 rounded-xl p-4 border border-blue-100">
      <h4 className="font-semibold text-sm text-gray-800 mb-3">Leave a Review</h4>
      <div className="mb-3">
        <p className="text-xs text-gray-500 mb-1">Your rating</p>
        <StarRating value={rating} onChange={setRating} />
      </div>
      <textarea
        value={comment}
        onChange={(e) => setComment(e.target.value)}
        placeholder="Share your experience with this seller..."
        rows={3}
        className="w-full text-sm border border-blue-200 rounded-xl px-3 py-2 focus:outline-none focus:border-[#0071CE] bg-white resize-none"
      />
      {error && (
        <p className="text-xs text-red-500 mt-1 flex items-center gap-1">
          <AlertCircle size={11} />
          {(() => {
            const axiosErr = error as { response?: { data?: { error?: string } }; message?: string };
            return axiosErr?.response?.data?.error ?? axiosErr?.message ?? "Failed to submit review.";
          })()}
        </p>
      )}
      <button
        onClick={() => mutate()}
        disabled={rating === 0 || !comment.trim() || isPending}
        className="mt-3 w-full bg-[#0071CE] text-white text-sm font-semibold py-2 rounded-xl hover:bg-[#005BA1] transition-colors disabled:opacity-40 flex items-center justify-center gap-2"
      >
        {isPending ? <Loader2 size={14} className="animate-spin" /> : <ThumbsUp size={14} />}
        Submit Review
      </button>
    </div>
  );
}

export function SellerReviews({ sellerId, sellerName, rating, reviewCount }: SellerReviewsProps) {
  const { isAuthenticated } = useAuthStore();
  const [showForm, setShowForm] = useState(false);
  const qc = useQueryClient();

  const { data: reviews, isLoading, isError } = useQuery<Review[]>({
    queryKey: ["seller-reviews", sellerId],
    queryFn: () =>
      api.get(`/users/${sellerId}/reviews`).then((r) => {
        const d = r.data.data ?? r.data;
        return Array.isArray(d) ? d : [];
      }),
    staleTime: 60_000,
    retry: 1,
  });

  const displayReviews = reviews ?? [];

  const avgRating = rating ?? (displayReviews.length
    ? displayReviews.reduce((s, r) => s + r.rating, 0) / displayReviews.length
    : 0);

  const totalCount = reviewCount ?? displayReviews.length;

  return (
    <div className="mt-8">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-bold text-gray-900">
          Reviews
          {totalCount > 0 && (
            <span className="ml-2 text-sm text-gray-500 font-normal">
              ({totalCount})
            </span>
          )}
        </h2>
        {isAuthenticated && !showForm && (
          <button
            onClick={() => setShowForm(true)}
            className="text-sm font-semibold text-[#0071CE] hover:underline"
          >
            Write a review
          </button>
        )}
      </div>

      {avgRating > 0 && (
        <div className="flex items-center gap-3 mb-5 bg-gray-50 rounded-xl p-4">
          <p className="text-4xl font-extrabold text-gray-900">{avgRating.toFixed(1)}</p>
          <div>
            <StarRating value={Math.round(avgRating)} />
            <p className="text-xs text-gray-500 mt-1">
              Based on {totalCount} review{totalCount !== 1 ? "s" : ""}
            </p>
          </div>
        </div>
      )}

      {showForm && (
        <div className="mb-5">
          <ReviewForm
            sellerId={sellerId}
            onSuccess={() => {
              setShowForm(false);
              qc.invalidateQueries({ queryKey: ["seller-reviews", sellerId] });
              qc.invalidateQueries({ queryKey: ["seller", sellerId] });
            }}
          />
        </div>
      )}

      {isLoading ? (
        <div className="flex justify-center py-8">
          <Loader2 size={24} className="animate-spin text-gray-300" />
        </div>
      ) : isError ? (
        <div className="text-center py-6 text-gray-400">
          <AlertCircle size={28} className="mx-auto mb-2 opacity-40" />
          <p className="text-sm">Unable to load reviews at this time.</p>
        </div>
      ) : displayReviews.length === 0 ? (
        <div className="text-center py-8 text-gray-400">
          <Star size={36} className="mx-auto mb-2 opacity-30" />
          <p className="text-sm font-semibold">No reviews yet</p>
          <p className="text-xs mt-1">Be the first to review {sellerName}</p>
        </div>
      ) : (
        <div className="space-y-4">
          {displayReviews.map((review) => (
            <div key={review.id} className="bg-white rounded-xl p-4 border border-gray-100 shadow-sm">
              <div className="flex items-start justify-between mb-2">
                <div className="flex items-center gap-2">
                  <div className="w-8 h-8 rounded-full bg-[#0071CE] text-white text-xs font-bold flex items-center justify-center shrink-0">
                    {review.reviewer_name[0]?.toUpperCase()}
                  </div>
                  <div>
                    <p className="text-sm font-semibold text-gray-800">{review.reviewer_name}</p>
                    <StarRating value={review.rating} />
                  </div>
                </div>
                <p className="text-xs text-gray-400">
                  {new Date(review.created_at).toLocaleDateString("en-AE", {
                    year: "numeric",
                    month: "short",
                    day: "numeric",
                  })}
                </p>
              </div>
              {review.comment && (
                <p className="text-sm text-gray-600 leading-relaxed">{review.comment}</p>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
