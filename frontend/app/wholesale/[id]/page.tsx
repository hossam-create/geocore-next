'use client'

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { formatPrice } from "@/lib/utils";
import { useAuthStore } from "@/store/auth";
import {
  Package, CheckCircle, Truck, Clock, Shield, ChevronLeft,
  Minus, Plus, ShoppingCart, AlertCircle, Store
} from "lucide-react";
import { useTranslations } from "next-intl";

interface PriceTier {
  min_quantity: number;
  max_quantity: number;
  unit_price_cents: number;
}

interface WholesaleListing {
  id: string;
  seller_id: string;
  title: string;
  description: string;
  category_slug: string;
  images: string[];
  unit_price_cents: number;
  currency: string;
  tier_pricing: PriceTier[];
  moq: number;
  max_order_quantity: number;
  available_units: number;
  units_per_lot: number;
  shipping_per_unit_cents: number;
  free_shipping_moq: number;
  lead_time_days: number;
  status: string;
  is_verified: boolean;
  views_count: number;
  orders_count: number;
}

export default function WholesaleDetailPage() {
  const t = useTranslations("wholesale");
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const qc = useQueryClient();
  const [quantity, setQuantity] = useState(0);
  const [orderNotes, setOrderNotes] = useState("");
  const [orderSuccess, setOrderSuccess] = useState(false);

  const { data: listing, isLoading } = useQuery({
    queryKey: ["wholesale", "listing", id],
    queryFn: async () => {
      const res = await api.get(`/wholesale/listings/${id}`);
      return res.data?.data as WholesaleListing;
    },
    enabled: !!id,
  });

  const placeOrder = useMutation({
    mutationFn: async () => {
      const res = await api.post("/wholesale/orders", {
        listing_id: id,
        quantity,
        notes: orderNotes || undefined,
      });
      return res.data?.data;
    },
    onSuccess: () => {
      setOrderSuccess(true);
      qc.invalidateQueries({ queryKey: ["wholesale"] });
    },
  });

  if (isLoading) {
    return (
      <div className="max-w-5xl mx-auto px-4 py-10">
        <div className="h-96 animate-pulse rounded-2xl bg-gray-100" />
      </div>
    );
  }

  if (!listing) {
    return (
      <div className="max-w-5xl mx-auto px-4 py-20 text-center">
        <Package size={48} className="text-gray-300 mx-auto mb-4" />
        <h1 className="text-xl font-bold text-gray-700">Listing not found</h1>
        <button onClick={() => router.push("/wholesale")} className="text-emerald-600 text-sm mt-3 hover:underline">
          ← Back to Wholesale
        </button>
      </div>
    );
  }

  const basePrice = listing.unit_price_cents / 100;

  // Find the applicable tier price for current quantity
  const getApplicablePrice = (qty: number) => {
    if (!listing.tier_pricing?.length) return basePrice;
    let price = basePrice;
    for (const tier of listing.tier_pricing) {
      if (qty >= tier.min_quantity && (tier.max_quantity === 0 || qty <= tier.max_quantity)) {
        price = tier.unit_price_cents / 100;
      }
    }
    return price;
  };

  const unitPrice = getApplicablePrice(quantity || listing.moq);
  const totalPrice = unitPrice * (quantity || listing.moq);
  const shippingPerUnit = listing.shipping_per_unit_cents / 100;
  const totalShipping = (listing.free_shipping_moq > 0 && (quantity || listing.moq) >= listing.free_shipping_moq)
    ? 0
    : shippingPerUnit * (quantity || listing.moq);
  const grandTotal = totalPrice + totalShipping;

  const discount = basePrice > 0 ? Math.round((1 - unitPrice / basePrice) * 100) : 0;

  if (orderSuccess) {
    return (
      <div className="max-w-5xl mx-auto px-4 py-20 text-center">
        <CheckCircle size={64} className="text-emerald-500 mx-auto mb-4" />
        <h1 className="text-2xl font-bold text-gray-900 mb-2">Order Placed!</h1>
        <p className="text-gray-500 text-sm mb-6">Your wholesale order has been sent to the seller for confirmation.</p>
        <div className="flex gap-3 justify-center">
          <button onClick={() => router.push("/orders")} className="bg-emerald-600 text-white font-bold px-6 py-3 rounded-xl hover:bg-emerald-700 transition-colors">
            View Orders
          </button>
          <button onClick={() => router.push("/wholesale")} className="border border-gray-200 text-gray-700 font-semibold px-6 py-3 rounded-xl hover:bg-gray-50 transition-colors">
            Continue Browsing
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-5xl mx-auto px-4 py-6">
      <button onClick={() => router.push("/wholesale")} className="flex items-center gap-1.5 text-gray-500 hover:text-emerald-600 text-sm mb-6 transition-colors">
        <ChevronLeft size={16} /> Back to Wholesale
      </button>

      <div className="grid grid-cols-1 lg:grid-cols-5 gap-6">
        {/* Left: Images + Details */}
        <div className="lg:col-span-3 space-y-5">
          {/* Main Image */}
          <div className="bg-white rounded-2xl border border-gray-100 overflow-hidden">
            <div className="h-80 bg-gray-50 flex items-center justify-center">
              {listing.images?.[0] ? (
                <img src={listing.images[0]} alt={listing.title} className="w-full h-full object-cover" />
              ) : (
                <Package size={64} className="text-gray-200" />
              )}
            </div>
            {listing.images?.length > 1 && (
              <div className="flex gap-2 p-3 overflow-x-auto">
                {listing.images.slice(1).map((img, i) => (
                  <img key={i} src={img} alt="" className="w-16 h-16 rounded-lg object-cover border border-gray-100" />
                ))}
              </div>
            )}
          </div>

          {/* Description */}
          <div className="bg-white rounded-2xl border border-gray-100 p-5">
            <h2 className="font-semibold text-gray-800 mb-3">Description</h2>
            <p className="text-sm text-gray-600 whitespace-pre-wrap">{listing.description || "No description provided."}</p>
          </div>

          {/* Tier Pricing Table */}
          {listing.tier_pricing?.length > 0 && (
            <div className="bg-white rounded-2xl border border-gray-100 p-5">
              <h2 className="font-semibold text-gray-800 mb-3">{t("volumePricing")}</h2>
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-gray-400 text-xs uppercase">
                    <th className="text-left pb-2">Quantity</th>
                    <th className="text-right pb-2">Unit Price</th>
                    <th className="text-right pb-2">Savings</th>
                  </tr>
                </thead>
                <tbody>
                  <tr className="border-t border-gray-50">
                    <td className="py-2 text-gray-600">1+ units</td>
                    <td className="py-2 text-right font-medium">{formatPrice(basePrice, listing.currency)}</td>
                    <td className="py-2 text-right text-gray-400">—</td>
                  </tr>
                  {listing.tier_pricing.map((tier, i) => {
                    const tierPrice = tier.unit_price_cents / 100;
                    const tierDiscount = Math.round((1 - tierPrice / basePrice) * 100);
                    return (
                      <tr key={i} className="border-t border-gray-50">
                        <td className="py-2 text-gray-600">{tier.min_quantity}+ units</td>
                        <td className="py-2 text-right font-semibold text-emerald-700">{formatPrice(tierPrice, listing.currency)}</td>
                        <td className="py-2 text-right text-emerald-600 font-medium">{tierDiscount}% off</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Right: Order Panel */}
        <div className="lg:col-span-2 space-y-4">
          {/* Title + Badges */}
          <div className="bg-white rounded-2xl border border-gray-100 p-5">
            <div className="flex items-center gap-2 mb-2">
              {listing.is_verified && (
                <span className="bg-emerald-50 text-emerald-700 text-[10px] font-bold px-2 py-0.5 rounded-full flex items-center gap-1">
                  <CheckCircle size={10} /> {t("verifiedSellers")}
                </span>
              )}
              <span className="bg-amber-50 text-amber-700 text-[10px] font-bold px-2 py-0.5 rounded-full">
                {t("moq")}: {listing.moq}
              </span>
            </div>
            <h1 className="text-xl font-bold text-gray-900 mb-1">{listing.title}</h1>
            <div className="flex items-baseline gap-2">
              <span className="text-2xl font-extrabold text-emerald-700">{formatPrice(unitPrice, listing.currency)}</span>
              <span className="text-xs text-gray-400">/ unit</span>
              {discount > 0 && (
                <span className="text-xs text-red-500 font-medium">-{discount}%</span>
              )}
            </div>
            {basePrice !== unitPrice && (
              <p className="text-xs text-gray-400 mt-1">Base price: {formatPrice(basePrice, listing.currency)}/unit</p>
            )}

            <div className="grid grid-cols-2 gap-3 mt-4">
              <div className="bg-gray-50 rounded-lg p-2.5 text-center">
                <p className="text-[10px] text-gray-400 uppercase">Available</p>
                <p className="text-sm font-bold text-gray-700">{listing.available_units > 0 ? listing.available_units : "Unlimited"}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-2.5 text-center">
                <p className="text-[10px] text-gray-400 uppercase">Per Lot</p>
                <p className="text-sm font-bold text-gray-700">{listing.units_per_lot} units</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-2.5 text-center">
                <p className="text-[10px] text-gray-400 uppercase">Lead Time</p>
                <p className="text-sm font-bold text-gray-700">{listing.lead_time_days} days</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-2.5 text-center">
                <p className="text-[10px] text-gray-400 uppercase">Orders</p>
                <p className="text-sm font-bold text-gray-700">{listing.orders_count}</p>
              </div>
            </div>
          </div>

          {/* Quantity Selector */}
          <div className="bg-white rounded-2xl border border-gray-100 p-5">
            <h3 className="font-semibold text-gray-800 text-sm mb-3">Select Quantity</h3>
            <div className="flex items-center gap-3">
              <button onClick={() => setQuantity(Math.max(listing.moq, quantity - listing.units_per_lot))} className="w-10 h-10 rounded-lg border border-gray-200 flex items-center justify-center hover:bg-gray-50">
                <Minus size={16} />
              </button>
              <input
                type="number"
                value={quantity || listing.moq}
                onChange={(e) => {
                  const v = parseInt(e.target.value) || listing.moq;
                  setQuantity(Math.max(listing.moq, v));
                }}
                min={listing.moq}
                className="w-24 text-center border border-gray-200 rounded-lg py-2 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-emerald-500"
              />
              <button onClick={() => setQuantity(quantity + listing.units_per_lot)} className="w-10 h-10 rounded-lg border border-gray-200 flex items-center justify-center hover:bg-gray-50">
                <Plus size={16} />
              </button>
              <span className="text-xs text-gray-400">Min: {listing.moq}</span>
            </div>

            {quantity < listing.moq && (
              <p className="flex items-center gap-1 text-xs text-amber-600 mt-2">
                <AlertCircle size={12} /> Minimum order is {listing.moq} units
              </p>
            )}
          </div>

          {/* Order Summary */}
          <div className="bg-white rounded-2xl border border-gray-100 p-5">
            <h3 className="font-semibold text-gray-800 text-sm mb-3">{t("orderSummary")}</h3>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-gray-500">Unit Price</span>
                <span className="font-medium">{formatPrice(unitPrice, listing.currency)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-500">Quantity</span>
                <span className="font-medium">{quantity || listing.moq}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-500">Subtotal</span>
                <span className="font-medium">{formatPrice(totalPrice, listing.currency)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-500">Shipping</span>
                <span className="font-medium">
                  {totalShipping === 0 ? (
                    <span className="text-emerald-600">FREE</span>
                  ) : (
                    formatPrice(totalShipping, listing.currency)
                  )}
                </span>
              </div>
              <div className="border-t border-gray-100 pt-2 flex justify-between">
                <span className="font-semibold text-gray-800">Total</span>
                <span className="text-lg font-extrabold text-emerald-700">{formatPrice(grandTotal, listing.currency)}</span>
              </div>
            </div>

            {listing.free_shipping_moq > 0 && (quantity || listing.moq) < listing.free_shipping_moq && (
              <p className="text-xs text-emerald-600 mt-2">
                <Truck size={12} className="inline mr-1" />
                Order {listing.free_shipping_moq}+ units for free shipping!
              </p>
            )}

            <textarea
              value={orderNotes}
              onChange={(e) => setOrderNotes(e.target.value)}
              placeholder="Order notes (optional)"
              className="w-full mt-3 p-3 border border-gray-200 rounded-xl text-sm resize-none h-16 focus:outline-none focus:ring-2 focus:ring-emerald-500"
            />

            {isAuthenticated ? (
              <button
                onClick={() => placeOrder.mutate()}
                disabled={quantity < listing.moq || placeOrder.isPending}
                className="w-full mt-4 bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 text-white font-bold py-3.5 rounded-xl transition-colors flex items-center justify-center gap-2 text-sm"
              >
                <ShoppingCart size={16} />
                {placeOrder.isPending ? "Placing Order..." : `Place Order — ${formatPrice(grandTotal, listing.currency)}`}
              </button>
            ) : (
              <button
                onClick={() => router.push("/login?next=/wholesale/" + id)}
                className="w-full mt-4 bg-emerald-600 hover:bg-emerald-700 text-white font-bold py-3.5 rounded-xl transition-colors flex items-center justify-center gap-2 text-sm"
              >
                Login to Order
              </button>
            )}

            <div className="flex items-center gap-2 mt-3 text-xs text-gray-400">
              <Shield size={12} />
              <span>Payment held in escrow until delivery confirmed</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
