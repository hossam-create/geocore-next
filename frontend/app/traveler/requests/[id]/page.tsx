"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import axios from "axios";
import { ArrowLeft, Package, Plane, CheckCircle, XCircle, Star } from "lucide-react";

interface MatchResult {
  trip: {
    id: string;
    traveler_id: string;
    origin_country: string;
    origin_city: string;
    dest_country: string;
    dest_city: string;
    departure_date: string;
    arrival_date: string;
    available_weight: number;
    price_per_kg: number;
    base_price: number;
    currency: string;
  };
  match_score: number;
  estimated_cost: number;
  estimated_delivery: string;
  can_deliver: boolean;
}

interface DeliveryRequest {
  id: string;
  item_name: string;
  item_description: string;
  item_price: number;
  item_weight: number | null;
  pickup_country: string;
  pickup_city: string;
  delivery_country: string;
  delivery_city: string;
  reward: number;
  currency: string;
  status: string;
  match_score: number | null;
  trip_id: string | null;
}

export default function DeliveryRequestDetailPage() {
  const { id } = useParams<{ id: string }>();
  const qc = useQueryClient();
  const [finding, setFinding] = useState(false);
  const [matches, setMatches] = useState<MatchResult[]>([]);

  const { data: dr, isLoading } = useQuery<DeliveryRequest>({
    queryKey: ["delivery-request", id],
    queryFn: async () => {
      const { data } = await axios.get(`/api/v1/delivery-requests/${id}`);
      return data.data;
    },
  });

  const findTravelers = async () => {
    setFinding(true);
    try {
      const { data } = await axios.post(`/api/v1/delivery-requests/${id}/find-travelers`);
      setMatches(data.data?.matches ?? []);
    } catch {
      setMatches([]);
    } finally {
      setFinding(false);
    }
  };

  const matchMutation = useMutation({
    mutationFn: (tripId: string) => axios.post(`/api/v1/delivery-requests/${id}/match`, { trip_id: tripId }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["delivery-request", id] });
      setMatches([]);
    },
  });

  const acceptMutation = useMutation({
    mutationFn: () => axios.post(`/api/v1/delivery-requests/${id}/accept`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["delivery-request", id] }),
  });

  const rejectMutation = useMutation({
    mutationFn: () => axios.post(`/api/v1/delivery-requests/${id}/reject`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["delivery-request", id] }),
  });

  const confirmMutation = useMutation({
    mutationFn: () => axios.post(`/api/v1/delivery-requests/${id}/confirm-delivery`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["delivery-request", id] }),
  });

  if (isLoading) return <div className="text-center py-16 text-gray-400">Loading...</div>;
  if (!dr) return <div className="text-center py-16 text-gray-400">Not found</div>;

  const statusColor: Record<string, string> = {
    pending: "bg-orange-100 text-orange-700",
    matched: "bg-blue-100 text-blue-700",
    accepted: "bg-indigo-100 text-indigo-700",
    delivered: "bg-emerald-100 text-emerald-700",
    cancelled: "bg-red-100 text-red-600",
  };

  return (
    <div className="max-w-3xl mx-auto px-4 py-8">
      <Link href="/traveler" className="flex items-center gap-2 text-gray-500 hover:text-gray-700 mb-6 text-sm">
        <ArrowLeft className="w-4 h-4" /> Back
      </Link>

      {/* Header */}
      <div className="bg-white rounded-xl border p-6 mb-6">
        <div className="flex items-start justify-between mb-4">
          <div className="flex items-center gap-3">
            <Package className="w-6 h-6 text-orange-500" />
            <h1 className="text-xl font-bold">{dr.item_name}</h1>
          </div>
          <span className={`text-sm px-3 py-1 rounded-full font-medium ${statusColor[dr.status] ?? "bg-gray-100"}`}>
            {dr.status}
          </span>
        </div>

        {dr.item_description && <p className="text-gray-600 text-sm mb-4">{dr.item_description}</p>}

        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="text-gray-500">Route:</span>
            <p className="font-medium">{dr.pickup_city}, {dr.pickup_country} &rarr; {dr.delivery_city}, {dr.delivery_country}</p>
          </div>
          <div>
            <span className="text-gray-500">Item Value:</span>
            <p className="font-medium">{dr.currency} {dr.item_price}</p>
          </div>
          <div>
            <span className="text-gray-500">Weight:</span>
            <p className="font-medium">{dr.item_weight ? `${dr.item_weight} kg` : "N/A"}</p>
          </div>
          <div>
            <span className="text-gray-500">Reward:</span>
            <p className="font-medium text-green-600">{dr.currency} {dr.reward}</p>
          </div>
        </div>

        {dr.match_score && (
          <div className="mt-4 flex items-center gap-2 text-sm">
            <Star className="w-4 h-4 text-yellow-500" />
            <span>Match score: <strong>{dr.match_score.toFixed(1)}</strong>/200</span>
          </div>
        )}
      </div>

      {/* Actions based on status */}
      {dr.status === "pending" && (
        <div className="mb-6">
          <button onClick={findTravelers} disabled={finding} className="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 disabled:opacity-50 transition text-sm">
            {finding ? "Searching..." : "Find Travelers"}
          </button>
        </div>
      )}

      {dr.status === "matched" && (
        <div className="flex gap-3 mb-6">
          <button onClick={() => acceptMutation.mutate()} className="flex items-center gap-2 bg-green-600 text-white px-5 py-2 rounded-lg hover:bg-green-700 transition text-sm">
            <CheckCircle className="w-4 h-4" /> Accept Match
          </button>
          <button onClick={() => rejectMutation.mutate()} className="flex items-center gap-2 bg-red-50 text-red-600 border border-red-200 px-5 py-2 rounded-lg hover:bg-red-100 transition text-sm">
            <XCircle className="w-4 h-4" /> Reject
          </button>
        </div>
      )}

      {(dr.status === "accepted" || dr.status === "picked_up" || dr.status === "in_transit") && (
        <div className="mb-6">
          <button onClick={() => confirmMutation.mutate()} className="flex items-center gap-2 bg-emerald-600 text-white px-5 py-2 rounded-lg hover:bg-emerald-700 transition text-sm">
            <CheckCircle className="w-4 h-4" /> Confirm Delivery
          </button>
        </div>
      )}

      {/* Matches */}
      {matches.length > 0 && (
        <div>
          <h2 className="text-lg font-semibold mb-3">Compatible Travelers ({matches.length})</h2>
          <div className="space-y-3">
            {matches.map((m) => (
              <div key={m.trip.id} className="bg-white rounded-xl border p-5">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <Plane className="w-5 h-5 text-blue-500" />
                    <span className="font-medium">
                      {m.trip.origin_city} &rarr; {m.trip.dest_city}
                    </span>
                  </div>
                  <span className="text-sm font-medium bg-blue-50 text-blue-700 px-2 py-1 rounded-full">
                    Score: {m.match_score.toFixed(0)}
                  </span>
                </div>
                <div className="grid grid-cols-3 gap-2 text-sm text-gray-500 mb-3">
                  <span>Departs: {new Date(m.trip.departure_date).toLocaleDateString()}</span>
                  <span>Capacity: {m.trip.available_weight} kg</span>
                  <span>Cost: {m.trip.currency} {m.estimated_cost.toFixed(2)}</span>
                </div>
                <button
                  onClick={() => matchMutation.mutate(m.trip.id)}
                  disabled={!m.can_deliver || matchMutation.isPending}
                  className="bg-blue-600 text-white px-4 py-1.5 rounded-lg text-sm hover:bg-blue-700 disabled:opacity-50 transition"
                >
                  {m.can_deliver ? "Select This Traveler" : "Insufficient Capacity"}
                </button>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
