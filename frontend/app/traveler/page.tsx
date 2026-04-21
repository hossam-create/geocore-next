"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import axios from "axios";
import { useAuthStore } from "@/store/auth";
import {
  Plane, Package, Plus, TrendingUp, Clock, CheckCircle,
  ArrowUpRight, DollarSign, MapPin, Calendar, Weight,
  ChevronRight, Star, AlertTriangle,
} from "lucide-react";

interface Trip {
  id: string;
  origin_country: string;
  origin_city: string;
  dest_country: string;
  dest_city: string;
  departure_date: string;
  arrival_date: string;
  available_weight: number;
  price_per_kg: number;
  status: string;
}

interface DeliveryRequest {
  id: string;
  item_name: string;
  item_price: number;
  pickup_country: string;
  pickup_city: string;
  delivery_country: string;
  delivery_city: string;
  reward: number;
  status: string;
  created_at: string;
}

const STATUS_COLORS: Record<string, string> = {
  active:     "bg-emerald-50 text-emerald-700 border border-emerald-200",
  matched:    "bg-blue-50 text-blue-700 border border-blue-200",
  in_transit: "bg-amber-50 text-amber-700 border border-amber-200",
  completed:  "bg-gray-100 text-gray-600 border border-gray-200",
  cancelled:  "bg-red-50 text-red-600 border border-red-200",
  pending:    "bg-orange-50 text-orange-700 border border-orange-200",
  accepted:   "bg-indigo-50 text-indigo-700 border border-indigo-200",
  delivered:  "bg-emerald-50 text-emerald-700 border border-emerald-200",
};

const MOCK_TRIPS: Trip[] = [
  { id: "t1", origin_city: "Dubai", origin_country: "UAE", dest_city: "London", dest_country: "UK", departure_date: new Date(Date.now() + 7 * 86400000).toISOString(), arrival_date: new Date(Date.now() + 9 * 86400000).toISOString(), available_weight: 8, price_per_kg: 25, status: "active" },
  { id: "t2", origin_city: "Cairo", origin_country: "Egypt", dest_city: "Istanbul", dest_country: "Turkey", departure_date: new Date(Date.now() + 14 * 86400000).toISOString(), arrival_date: new Date(Date.now() + 16 * 86400000).toISOString(), available_weight: 5, price_per_kg: 20, status: "active" },
  { id: "t3", origin_city: "Riyadh", origin_country: "KSA", dest_city: "Paris", dest_country: "France", departure_date: new Date(Date.now() - 10 * 86400000).toISOString(), arrival_date: new Date(Date.now() - 8 * 86400000).toISOString(), available_weight: 0, price_per_kg: 30, status: "completed" },
];

const MOCK_REQUESTS: DeliveryRequest[] = [
  { id: "r1", item_name: "Italian Leather Bag", item_price: 350, pickup_city: "Milan", pickup_country: "Italy", delivery_city: "Dubai", delivery_country: "UAE", reward: 45, status: "pending", created_at: new Date(Date.now() - 86400000).toISOString() },
  { id: "r2", item_name: "Swiss Chocolate Gift Set", item_price: 120, pickup_city: "Zurich", pickup_country: "Switzerland", delivery_city: "Riyadh", delivery_country: "KSA", reward: 30, status: "accepted", created_at: new Date(Date.now() - 2 * 86400000).toISOString() },
];

export default function TravelerDashboardPage() {
  const { user, isAuthenticated } = useAuthStore();
  const [tab, setTab] = useState<"trips" | "requests">("trips");

  const { data: trips = MOCK_TRIPS } = useQuery<Trip[]>({
    queryKey: ["my-trips"],
    queryFn: async () => {
      try {
        const { data } = await axios.get("/api/v1/trips?mine=true");
        const d = data.data ?? [];
        return Array.isArray(d) && d.length ? d : MOCK_TRIPS;
      } catch { return MOCK_TRIPS; }
    },
  });

  const { data: requests = MOCK_REQUESTS } = useQuery<DeliveryRequest[]>({
    queryKey: ["my-delivery-requests"],
    queryFn: async () => {
      try {
        const { data } = await axios.get("/api/v1/delivery-requests?mine=true");
        const d = data.data ?? [];
        return Array.isArray(d) && d.length ? d : MOCK_REQUESTS;
      } catch { return MOCK_REQUESTS; }
    },
  });

  const activeTrips     = trips.filter((t) => t.status === "active");
  const completedTrips  = trips.filter((t) => t.status === "completed");
  const pendingRequests = requests.filter((r) => r.status === "pending");
  const totalEarnings   = requests.filter((r) => r.status === "delivered").reduce((s, r) => s + r.reward, 0);

  return (
    <div className="space-y-6">

      {/* Welcome */}
      <p className="text-sm text-gray-500">
        Welcome back, <span className="font-semibold text-gray-700">{user?.name?.split(" ")[0] ?? "Traveler"}</span>. Earn by carrying items on your travel routes.
      </p>

      {/* ── Alert: pending requests ── */}
      {pendingRequests.length > 0 && (
        <div className="flex items-center gap-3 bg-blue-50 border border-blue-200/70 rounded-xl px-4 py-3">
          <AlertTriangle className="w-4 h-4 text-blue-500 shrink-0" />
          <p className="text-sm text-blue-800 font-medium flex-1">
            <strong>{pendingRequests.length}</strong> new delivery {pendingRequests.length === 1 ? "request" : "requests"} match your upcoming routes.
          </p>
          <Link href="/traveler/orders" className="text-xs font-semibold text-blue-700 hover:text-blue-900 whitespace-nowrap">
            View requests →
          </Link>
        </div>
      )}

      {/* ── KPI Cards ── */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {[
          { label: "Active Trips", value: String(activeTrips.length), icon: Plane, accent: "bg-blue-500", sub: `${completedTrips.length} completed` },
          { label: "Pending Deliveries", value: String(pendingRequests.length), icon: Package, accent: "bg-amber-500", sub: "awaiting match" },
          { label: "Total Earnings", value: `$${totalEarnings.toFixed(0)}`, icon: DollarSign, accent: "bg-emerald-500", sub: "from deliveries" },
          { label: "Avg. Reward", value: requests.length ? `$${(requests.reduce((s, r) => s + r.reward, 0) / requests.length).toFixed(0)}` : "$0", icon: Star, accent: "bg-violet-500", sub: "per delivery" },
        ].map(({ label, value, icon: Icon, accent, sub }) => (
          <div key={label} className={`relative bg-white rounded-2xl border border-gray-100 p-5 overflow-hidden`}>
            <div className={`absolute top-0 left-0 right-0 h-[3px] ${accent}`} />
            <div className="flex items-start justify-between mb-3">
              <p className="text-xs font-semibold text-gray-400 uppercase tracking-wider">{label}</p>
              <div className="w-8 h-8 rounded-xl bg-gray-50 flex items-center justify-center">
                <Icon className="w-4 h-4 text-gray-400" />
              </div>
            </div>
            <p className="text-2xl font-bold text-gray-900 tabular-nums">{value}</p>
            <p className="text-xs text-gray-400 mt-1">{sub}</p>
          </div>
        ))}
      </div>

      {/* ── Tab Switcher ── */}
      <div className="flex gap-1 bg-gray-100 rounded-xl p-1 w-fit">
        {(["trips", "requests"] as const).map((t) => (
          <button key={t} onClick={() => setTab(t)}
            className={`px-4 py-2 rounded-lg text-sm font-semibold capitalize transition-all ${tab === t ? "bg-white shadow-sm text-[#0071CE]" : "text-gray-500 hover:text-gray-700"}`}>
            {t === "trips" ? `My Trips (${trips.length})` : `Delivery Requests (${requests.length})`}
          </button>
        ))}
      </div>

      {/* ── Trips Tab ── */}
      {tab === "trips" && (
        <div className="space-y-3">
          {trips.length === 0 && (
            <div className="bg-white rounded-2xl border border-gray-100 py-16 text-center">
              <Plane className="w-10 h-10 mx-auto mb-3 text-gray-200" />
              <p className="text-sm text-gray-500 font-medium">No trips posted yet</p>
              <p className="text-xs text-gray-400 mt-1">Post your first trip to start earning</p>
              <Link href="/traveler/trips/new" className="inline-flex items-center gap-1 mt-3 text-xs text-[#0071CE] font-semibold hover:underline">
                <Plus className="w-3 h-3" /> Create a trip
              </Link>
            </div>
          )}
          {trips.map((trip) => (
            <Link key={trip.id} href={`/traveler/trips/${trip.id}`}
              className="flex items-center gap-4 bg-white rounded-2xl border border-gray-100 px-5 py-4 hover:shadow-sm transition-shadow group">
              {/* Route icon */}
              <div className="w-10 h-10 rounded-xl bg-blue-50 flex items-center justify-center shrink-0">
                <Plane className="w-5 h-5 text-[#0071CE]" />
              </div>

              {/* Route info */}
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <p className="text-sm font-semibold text-gray-900">
                    {trip.origin_city} <span className="text-gray-400 mx-1">→</span> {trip.dest_city}
                  </p>
                  <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${STATUS_COLORS[trip.status] ?? "bg-gray-100 text-gray-500"}`}>
                    {trip.status.replace(/_/g, " ")}
                  </span>
                </div>
                <div className="flex items-center gap-3 mt-1">
                  <span className="text-xs text-gray-400 flex items-center gap-1">
                    <Calendar className="w-3 h-3" />
                    {new Date(trip.departure_date).toLocaleDateString("en-US", { month: "short", day: "numeric" })}
                  </span>
                  <span className="text-xs text-gray-400 flex items-center gap-1">
                    <Weight className="w-3 h-3" />
                    {trip.available_weight} kg available
                  </span>
                  <span className="text-xs text-gray-400 flex items-center gap-1">
                    <DollarSign className="w-3 h-3" />
                    ${trip.price_per_kg}/kg
                  </span>
                </div>
              </div>

              <ChevronRight className="w-4 h-4 text-gray-300 group-hover:text-gray-500 transition-colors shrink-0" />
            </Link>
          ))}
        </div>
      )}

      {/* ── Requests Tab ── */}
      {tab === "requests" && (
        <div className="space-y-3">
          {requests.length === 0 && (
            <div className="bg-white rounded-2xl border border-gray-100 py-16 text-center">
              <Package className="w-10 h-10 mx-auto mb-3 text-gray-200" />
              <p className="text-sm text-gray-500 font-medium">No delivery requests yet</p>
              <Link href="/traveler/browse" className="inline-flex items-center gap-1 mt-3 text-xs text-[#0071CE] font-semibold hover:underline">
                Browse available requests →
              </Link>
            </div>
          )}
          {requests.map((req) => (
            <Link key={req.id} href={`/traveler/requests/${req.id}`}
              className="flex items-center gap-4 bg-white rounded-2xl border border-gray-100 px-5 py-4 hover:shadow-sm transition-shadow group">
              {/* Icon */}
              <div className="w-10 h-10 rounded-xl bg-orange-50 flex items-center justify-center shrink-0">
                <Package className="w-5 h-5 text-orange-500" />
              </div>

              {/* Info */}
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <p className="text-sm font-semibold text-gray-900 truncate">{req.item_name}</p>
                  <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full shrink-0 ${STATUS_COLORS[req.status] ?? "bg-gray-100 text-gray-500"}`}>
                    {req.status}
                  </span>
                </div>
                <div className="flex items-center gap-3 mt-1">
                  <span className="text-xs text-gray-400 flex items-center gap-1">
                    <MapPin className="w-3 h-3" />
                    {req.pickup_city} → {req.delivery_city}
                  </span>
                  <span className="text-xs text-gray-400">Item value: ${req.item_price}</span>
                </div>
              </div>

              {/* Reward */}
              <div className="text-right shrink-0">
                <p className="text-sm font-bold text-emerald-600">+${req.reward}</p>
                <p className="text-[10px] text-gray-400">reward</p>
              </div>

              <ChevronRight className="w-4 h-4 text-gray-300 group-hover:text-gray-500 transition-colors shrink-0" />
            </Link>
          ))}
        </div>
      )}

      {/* ── Browse CTA ── */}
      <div className="bg-gradient-to-r from-[#0071CE]/5 to-blue-50 rounded-2xl border border-blue-100 p-5 flex items-center justify-between gap-4">
        <div>
          <p className="text-sm font-semibold text-gray-900">Ready for your next trip?</p>
          <p className="text-xs text-gray-500 mt-0.5">Browse delivery requests that match your route and earn while you travel.</p>
        </div>
        <Link href="/traveler/orders"
          className="flex items-center gap-2 px-4 py-2.5 bg-[#0071CE] text-white rounded-xl text-sm font-semibold hover:bg-[#005ba3] transition-colors whitespace-nowrap">
          <TrendingUp className="w-4 h-4" /> Browse Orders
        </Link>
      </div>
    </div>
  );
}
