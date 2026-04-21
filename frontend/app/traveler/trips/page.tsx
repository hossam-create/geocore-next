"use client";

import { useState } from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import axios from "axios";
import { useAuthStore } from "@/store/auth";
import {
  Plane, Plus, Calendar, Weight, DollarSign,
  ChevronRight, Search, XCircle, CheckCircle,
} from "lucide-react";
import { formatDistanceToNow, parseISO, format } from "date-fns";

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
  currency: string;
  status: string;
  matched_requests?: number;
}

const MOCK: Trip[] = [
  { id: "t1", origin_city: "Dubai",  origin_country: "UAE",    dest_city: "London",   dest_country: "UK",      departure_date: new Date(Date.now() + 7 * 86400000).toISOString(),  arrival_date: new Date(Date.now() + 9 * 86400000).toISOString(),  available_weight: 8,  price_per_kg: 25, currency: "AED", status: "active",    matched_requests: 2 },
  { id: "t2", origin_city: "Cairo",  origin_country: "Egypt",  dest_city: "Istanbul", dest_country: "Turkey",  departure_date: new Date(Date.now() + 14 * 86400000).toISOString(), arrival_date: new Date(Date.now() + 16 * 86400000).toISOString(), available_weight: 5,  price_per_kg: 20, currency: "AED", status: "active",    matched_requests: 0 },
  { id: "t3", origin_city: "Riyadh", origin_country: "KSA",    dest_city: "Paris",    dest_country: "France",  departure_date: new Date(Date.now() - 10 * 86400000).toISOString(), arrival_date: new Date(Date.now() - 8 * 86400000).toISOString(),  available_weight: 0,  price_per_kg: 30, currency: "AED", status: "completed", matched_requests: 3 },
  { id: "t4", origin_city: "Abu Dhabi", origin_country: "UAE", dest_city: "New York", dest_country: "USA",     departure_date: new Date(Date.now() + 3 * 86400000).toISOString(),  arrival_date: new Date(Date.now() + 5 * 86400000).toISOString(),  available_weight: 12, price_per_kg: 35, currency: "AED", status: "active",    matched_requests: 1 },
];

const STATUS_STYLES: Record<string, { label: string; cls: string }> = {
  active:     { label: "Active",      cls: "bg-emerald-50 text-emerald-700 border-emerald-200" },
  matched:    { label: "Matched",     cls: "bg-blue-50 text-blue-700 border-blue-200" },
  in_transit: { label: "In Transit",  cls: "bg-amber-50 text-amber-700 border-amber-200" },
  completed:  { label: "Completed",   cls: "bg-gray-100 text-gray-600 border-gray-200" },
  cancelled:  { label: "Cancelled",   cls: "bg-red-50 text-red-600 border-red-200" },
};

type FilterStatus = "all" | "active" | "completed" | "cancelled";

function fmtDate(d: string) {
  try { return format(parseISO(d), "MMM d, yyyy"); }
  catch { return d; }
}

export default function TravelerTripsPage() {
  const qc = useQueryClient();
  const { isAuthenticated } = useAuthStore();
  const [filter, setFilter] = useState<FilterStatus>("all");
  const [search, setSearch] = useState("");

  const { data: trips = MOCK, isLoading } = useQuery<Trip[]>({
    queryKey: ["my-trips-full"],
    queryFn: async () => {
      try {
        const { data } = await axios.get("/api/v1/trips?mine=true&limit=100");
        const d = data.data ?? [];
        return Array.isArray(d) && d.length ? d : MOCK;
      } catch { return MOCK; }
    },
    enabled: isAuthenticated,
  });

  const cancelMutation = useMutation({
    mutationFn: (id: string) => axios.patch(`/api/v1/trips/${id}/cancel`, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["my-trips-full"] }),
  });

  const filtered = trips.filter((t) => {
    const matchStatus = filter === "all" || t.status === filter;
    const q = search.toLowerCase();
    const matchSearch = !search || t.origin_city.toLowerCase().includes(q) || t.dest_city.toLowerCase().includes(q);
    return matchStatus && matchSearch;
  });

  const counts: Record<string, number> = { all: trips.length };
  trips.forEach((t) => { counts[t.status] = (counts[t.status] ?? 0) + 1; });

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div>
          <h1 className="text-xl font-bold text-gray-900">My Trips</h1>
          <p className="text-sm text-gray-400">{trips.length} total trips posted</p>
        </div>
        <Link
          href="/traveler/trips/new"
          className="flex items-center gap-1.5 px-4 py-2 bg-[#0071CE] text-white rounded-xl text-sm font-semibold hover:bg-[#005ba3] transition-colors"
        >
          <Plus className="w-4 h-4" /> Post New Trip
        </Link>
      </div>

      {/* Filters + Search */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex gap-1.5 flex-wrap">
          {(["all", "active", "completed", "cancelled"] as FilterStatus[]).map((f) => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={`px-3 py-1.5 rounded-full text-xs font-semibold capitalize transition-colors ${
                filter === f
                  ? "bg-[#0071CE] text-white"
                  : "bg-white text-gray-500 border border-gray-200 hover:bg-gray-50"
              }`}
            >
              {f === "all" ? `All (${counts.all})` : `${f.charAt(0).toUpperCase() + f.slice(1)} (${counts[f] ?? 0})`}
            </button>
          ))}
        </div>
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by city..."
            className="pl-8 pr-3 py-1.5 text-sm border border-gray-200 rounded-xl bg-white outline-none focus:ring-2 focus:ring-[#0071CE]/20 focus:border-[#0071CE] w-52"
          />
        </div>
      </div>

      {/* Trips list */}
      {isLoading ? (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => <div key={i} className="h-24 animate-pulse rounded-2xl bg-gray-100" />)}
        </div>
      ) : filtered.length === 0 ? (
        <div className="py-16 text-center bg-white rounded-2xl border border-gray-100">
          <Plane className="w-10 h-10 mx-auto mb-3 text-gray-200" />
          <p className="text-sm text-gray-400">
            {search ? `No trips matching "${search}"` : "No trips in this filter."}
          </p>
          <Link href="/traveler/trips/new" className="mt-2 inline-flex items-center gap-1 text-xs text-[#0071CE] font-semibold hover:underline">
            <Plus className="w-3 h-3" /> Post your first trip
          </Link>
        </div>
      ) : (
        <div className="space-y-3">
          {filtered.map((trip) => {
            const ss = STATUS_STYLES[trip.status] ?? STATUS_STYLES.active;
            const isUpcoming = new Date(trip.departure_date).getTime() > Date.now();
            return (
              <div key={trip.id} className="bg-white rounded-2xl border border-gray-100 px-5 py-4 hover:shadow-sm transition-shadow group">
                <div className="flex items-center gap-4">
                  {/* Icon */}
                  <div className="w-10 h-10 rounded-xl bg-blue-50 flex items-center justify-center shrink-0">
                    <Plane className="w-5 h-5 text-[#0071CE]" />
                  </div>

                  {/* Route + info */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <p className="text-sm font-semibold text-gray-900">
                        {trip.origin_city}, {trip.origin_country}
                        <span className="text-gray-400 mx-1.5">→</span>
                        {trip.dest_city}, {trip.dest_country}
                      </p>
                      <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${ss.cls}`}>
                        {ss.label}
                      </span>
                      {trip.matched_requests !== undefined && trip.matched_requests > 0 && (
                        <span className="text-[10px] font-bold px-2 py-0.5 rounded-full bg-violet-50 text-violet-700 border border-violet-200">
                          {trip.matched_requests} match{trip.matched_requests > 1 ? "es" : ""}
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-4 mt-1.5 flex-wrap">
                      <span className="text-xs text-gray-400 flex items-center gap-1">
                        <Calendar className="w-3 h-3" />
                        {fmtDate(trip.departure_date)}
                        {isUpcoming && (
                          <span className="text-[#0071CE] font-medium ml-1">
                            ({formatDistanceToNow(parseISO(trip.departure_date), { addSuffix: true })})
                          </span>
                        )}
                      </span>
                      <span className="text-xs text-gray-400 flex items-center gap-1">
                        <Weight className="w-3 h-3" /> {trip.available_weight} kg available
                      </span>
                      <span className="text-xs text-gray-400 flex items-center gap-1">
                        <DollarSign className="w-3 h-3" /> {trip.currency} {trip.price_per_kg}/kg
                      </span>
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-2 shrink-0">
                    {trip.status === "active" && (
                      <button
                        onClick={() => { if (confirm("Cancel this trip?")) cancelMutation.mutate(trip.id); }}
                        disabled={cancelMutation.isPending}
                        className="p-1.5 rounded-lg text-gray-400 hover:text-red-500 hover:bg-red-50 transition-colors"
                        title="Cancel trip"
                      >
                        <XCircle className="w-4 h-4" />
                      </button>
                    )}
                    <Link
                      href={`/traveler/requests/new?trip_id=${trip.id}`}
                      className="hidden group-hover:flex items-center gap-1 px-3 py-1.5 border border-gray-200 rounded-lg text-xs font-medium text-gray-600 hover:bg-gray-50 transition-colors"
                    >
                      <CheckCircle className="w-3.5 h-3.5" /> Accept Requests
                    </Link>
                    <ChevronRight className="w-4 h-4 text-gray-300" />
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
