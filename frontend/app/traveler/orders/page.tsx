"use client";

import { useState } from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import axios from "axios";
import { useAuthStore } from "@/store/auth";
import {
  Package, MapPin, DollarSign, Weight, Search,
  CheckCircle, ChevronRight, Clock, Truck, Star,
} from "lucide-react";
import { formatDistanceToNow, parseISO } from "date-fns";

interface DeliveryRequest {
  id: string;
  item_name: string;
  item_description?: string;
  item_price: number;
  item_weight: number | null;
  pickup_country: string;
  pickup_city: string;
  delivery_country: string;
  delivery_city: string;
  reward: number;
  currency: string;
  status: string;
  created_at: string;
  buyer_name?: string;
  buyer_rating?: number;
}

const MOCK: DeliveryRequest[] = [
  { id: "r1", item_name: "Italian Leather Bag",      item_description: "Authentic Gucci tote from Milan boutique", item_price: 1200, item_weight: 1.2, pickup_city: "Milan",       pickup_country: "Italy",        delivery_city: "Dubai",    delivery_country: "UAE",   reward: 180,  currency: "AED", status: "pending",  created_at: new Date(Date.now() - 86400000).toISOString(),     buyer_name: "Sara M.",    buyer_rating: 4.9 },
  { id: "r2", item_name: "Swiss Chocolate Gift Set", item_description: "Lindt premium assortment 500g",            item_price: 180,  item_weight: 0.6, pickup_city: "Zurich",      pickup_country: "Switzerland",  delivery_city: "Riyadh",   delivery_country: "KSA",   reward: 55,   currency: "AED", status: "pending",  created_at: new Date(Date.now() - 2 * 86400000).toISOString(), buyer_name: "Ahmed K.",   buyer_rating: 4.7 },
  { id: "r3", item_name: "Sneakers Nike Dunk Low",   item_description: "Brand new, unworn, US size 10",            item_price: 450,  item_weight: 1.4, pickup_city: "New York",    pickup_country: "USA",          delivery_city: "Abu Dhabi",delivery_country: "UAE",   reward: 120,  currency: "AED", status: "pending",  created_at: new Date(Date.now() - 3 * 86400000).toISOString(), buyer_name: "Omar F.",    buyer_rating: 4.8 },
  { id: "r4", item_name: "Camera Lens EF 50mm",      item_description: "Canon EF 50mm f/1.4 USM",                  item_price: 1500, item_weight: 0.9, pickup_city: "London",      pickup_country: "UK",           delivery_city: "Dubai",    delivery_country: "UAE",   reward: 200,  currency: "AED", status: "accepted", created_at: new Date(Date.now() - 4 * 86400000).toISOString(), buyer_name: "Layla H.",   buyer_rating: 5.0 },
  { id: "r5", item_name: "Book Set — Arabic Edition",item_description: "Set of 5 rare Arabic novels",              item_price: 280,  item_weight: 2.1, pickup_city: "Paris",       pickup_country: "France",       delivery_city: "Cairo",    delivery_country: "Egypt", reward: 70,   currency: "AED", status: "pending",  created_at: new Date(Date.now() - 5 * 86400000).toISOString(), buyer_name: "Noura A.",   buyer_rating: 4.6 },
];

const STATUS_CFG: Record<string, { label: string; cls: string; icon: React.ElementType }> = {
  pending:    { label: "Open",       cls: "bg-emerald-50 text-emerald-700 border-emerald-200", icon: Clock },
  accepted:   { label: "Accepted",   cls: "bg-blue-50 text-blue-700 border-blue-200",          icon: CheckCircle },
  in_transit: { label: "In Transit", cls: "bg-amber-50 text-amber-700 border-amber-200",       icon: Truck },
  delivered:  { label: "Delivered",  cls: "bg-gray-100 text-gray-600 border-gray-200",         icon: CheckCircle },
};

type SortKey = "newest" | "reward_high" | "reward_low" | "weight_low";

function timeAgo(d: string) {
  try { return formatDistanceToNow(parseISO(d), { addSuffix: true }); }
  catch { return d; }
}

export default function TravelerOrdersPage() {
  const qc = useQueryClient();
  const { isAuthenticated } = useAuthStore();
  const [pickupFilter, setPickupFilter] = useState("");
  const [destFilter, setDestFilter] = useState("");
  const [sort, setSort] = useState<SortKey>("newest");
  const [accepting, setAccepting] = useState<string | null>(null);
  const [notice, setNotice] = useState<{ id: string; msg: string } | null>(null);

  const { data: requests = MOCK, isLoading } = useQuery<DeliveryRequest[]>({
    queryKey: ["available-delivery-requests", pickupFilter, destFilter],
    queryFn: async () => {
      try {
        const params = new URLSearchParams({ status: "pending" });
        if (pickupFilter) params.set("pickup_country", pickupFilter);
        if (destFilter) params.set("delivery_country", destFilter);
        const { data } = await axios.get(`/api/v1/delivery-requests?${params}`);
        const d = data.data ?? [];
        return Array.isArray(d) && d.length ? d : MOCK;
      } catch { return MOCK; }
    },
    enabled: isAuthenticated,
  });

  const acceptMutation = useMutation({
    mutationFn: (id: string) => axios.post(`/api/v1/delivery-requests/${id}/accept`, {}),
    onMutate: (id) => setAccepting(id),
    onSuccess: (_, id) => {
      setNotice({ id, msg: "Request accepted! Contact the buyer to arrange pickup." });
      setAccepting(null);
      qc.invalidateQueries({ queryKey: ["available-delivery-requests"] });
      qc.invalidateQueries({ queryKey: ["my-delivery-requests"] });
    },
    onError: () => { setAccepting(null); },
  });

  let sorted = [...requests];
  if (sort === "reward_high")  sorted.sort((a, b) => b.reward - a.reward);
  if (sort === "reward_low")   sorted.sort((a, b) => a.reward - b.reward);
  if (sort === "weight_low")   sorted.sort((a, b) => (a.item_weight ?? 99) - (b.item_weight ?? 99));

  const pendingCount = requests.filter((r) => r.status === "pending").length;

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div>
          <h1 className="text-xl font-bold text-gray-900">Delivery Orders</h1>
          <p className="text-sm text-gray-400">{pendingCount} open requests available</p>
        </div>
        <Link
          href="/traveler/trips/new"
          className="flex items-center gap-1.5 px-4 py-2 bg-[#0071CE] text-white rounded-xl text-sm font-semibold hover:bg-[#005ba3] transition-colors"
        >
          Post a Trip First
        </Link>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative">
          <MapPin className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400" />
          <input
            value={pickupFilter}
            onChange={(e) => setPickupFilter(e.target.value)}
            placeholder="Pickup country..."
            className="pl-8 pr-3 py-1.5 text-sm border border-gray-200 rounded-xl bg-white outline-none focus:ring-2 focus:ring-[#0071CE]/20 focus:border-[#0071CE] w-44"
          />
        </div>
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400" />
          <input
            value={destFilter}
            onChange={(e) => setDestFilter(e.target.value)}
            placeholder="Delivery country..."
            className="pl-8 pr-3 py-1.5 text-sm border border-gray-200 rounded-xl bg-white outline-none focus:ring-2 focus:ring-[#0071CE]/20 focus:border-[#0071CE] w-44"
          />
        </div>
        <select
          value={sort}
          onChange={(e) => setSort(e.target.value as SortKey)}
          className="text-sm border border-gray-200 rounded-xl px-3 py-1.5 bg-white outline-none focus:ring-2 focus:ring-[#0071CE]/20"
        >
          <option value="newest">Newest First</option>
          <option value="reward_high">Highest Reward</option>
          <option value="reward_low">Lowest Reward</option>
          <option value="weight_low">Lightest First</option>
        </select>
      </div>

      {/* Request cards */}
      {isLoading ? (
        <div className="grid sm:grid-cols-2 gap-4">
          {[1, 2, 3, 4].map((i) => <div key={i} className="h-48 animate-pulse rounded-2xl bg-gray-100" />)}
        </div>
      ) : sorted.length === 0 ? (
        <div className="py-16 text-center bg-white rounded-2xl border border-gray-100">
          <Package className="w-10 h-10 mx-auto mb-3 text-gray-200" />
          <p className="text-sm text-gray-400">No delivery requests match your filters.</p>
        </div>
      ) : (
        <div className="grid sm:grid-cols-2 gap-4">
          {sorted.map((req) => {
            const cfg = STATUS_CFG[req.status] ?? STATUS_CFG.pending;
            const StatusIcon = cfg.icon;
            const isAccepted = req.status === "accepted";
            const thisNotice = notice?.id === req.id ? notice.msg : null;
            return (
              <div key={req.id} className="bg-white rounded-2xl border border-gray-100 p-5 hover:shadow-sm transition-shadow flex flex-col gap-3">
                {/* Top row */}
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2 flex-wrap">
                      <p className="text-sm font-semibold text-gray-900 truncate">{req.item_name}</p>
                      <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${cfg.cls} shrink-0`}>
                        <span className="inline-flex items-center gap-0.5"><StatusIcon className="w-2.5 h-2.5" /> {cfg.label}</span>
                      </span>
                    </div>
                    {req.item_description && (
                      <p className="text-xs text-gray-400 mt-0.5 line-clamp-1">{req.item_description}</p>
                    )}
                  </div>
                  {/* Reward badge */}
                  <div className="text-right shrink-0">
                    <p className="text-base font-bold text-emerald-600">+{req.currency} {req.reward}</p>
                    <p className="text-[10px] text-gray-400">reward</p>
                  </div>
                </div>

                {/* Route */}
                <div className="flex items-center gap-2 text-xs text-gray-600 bg-gray-50 rounded-xl px-3 py-2">
                  <MapPin className="w-3.5 h-3.5 text-gray-400 shrink-0" />
                  <span className="font-medium">{req.pickup_city}, {req.pickup_country}</span>
                  <ChevronRight className="w-3 h-3 text-gray-300 shrink-0" />
                  <span className="font-medium">{req.delivery_city}, {req.delivery_country}</span>
                </div>

                {/* Meta row */}
                <div className="flex items-center gap-3 text-xs text-gray-400 flex-wrap">
                  <span className="flex items-center gap-1">
                    <DollarSign className="w-3 h-3" /> Item: {req.currency} {req.item_price}
                  </span>
                  {req.item_weight && (
                    <span className="flex items-center gap-1">
                      <Weight className="w-3 h-3" /> {req.item_weight} kg
                    </span>
                  )}
                  {req.buyer_name && (
                    <span className="flex items-center gap-1">
                      <Star className="w-3 h-3 text-yellow-400" />
                      {req.buyer_name} ({req.buyer_rating?.toFixed(1) ?? "—"})
                    </span>
                  )}
                  <span className="ml-auto">{timeAgo(req.created_at)}</span>
                </div>

                {/* Notice */}
                {thisNotice && (
                  <p className="text-xs font-medium text-[#0071CE] bg-blue-50 rounded-lg px-3 py-2">{thisNotice}</p>
                )}

                {/* Action */}
                {!isAccepted ? (
                  <button
                    onClick={() => acceptMutation.mutate(req.id)}
                    disabled={accepting === req.id}
                    className="w-full py-2.5 rounded-xl bg-[#0071CE] text-white text-sm font-semibold hover:bg-[#005ba3] transition-colors disabled:opacity-60 flex items-center justify-center gap-2"
                  >
                    {accepting === req.id ? (
                      <span className="animate-pulse">Accepting…</span>
                    ) : (
                      <><CheckCircle className="w-4 h-4" /> Accept Delivery</>
                    )}
                  </button>
                ) : (
                  <Link
                    href={`/traveler/requests/${req.id}`}
                    className="w-full py-2.5 rounded-xl border border-gray-200 bg-white text-sm font-semibold text-gray-700 hover:bg-gray-50 transition-colors flex items-center justify-center gap-2"
                  >
                    <ChevronRight className="w-4 h-4" /> View Details
                  </Link>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
