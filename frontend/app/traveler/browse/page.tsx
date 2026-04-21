"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import axios from "axios";
import { Package, ArrowLeft, Search, MapPin } from "lucide-react";

interface DeliveryRequest {
  id: string;
  item_name: string;
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
}

export default function BrowseRequestsPage() {
  const [pickupCountry, setPickupCountry] = useState("");
  const [deliveryCountry, setDeliveryCountry] = useState("");

  const { data: requests = [], isLoading } = useQuery<DeliveryRequest[]>({
    queryKey: ["browse-requests", pickupCountry, deliveryCountry],
    queryFn: async () => {
      const params = new URLSearchParams({ status: "pending" });
      if (pickupCountry) params.set("pickup_country", pickupCountry);
      if (deliveryCountry) params.set("delivery_country", deliveryCountry);
      const { data } = await axios.get(`/api/v1/delivery-requests?${params}`);
      return data.data ?? [];
    },
  });

  return (
    <div className="max-w-5xl mx-auto px-4 py-8">
      <Link href="/traveler" className="flex items-center gap-2 text-gray-500 hover:text-gray-700 mb-6 text-sm">
        <ArrowLeft className="w-4 h-4" /> Back to Dashboard
      </Link>

      <h1 className="text-2xl font-bold mb-2">Browse Delivery Requests</h1>
      <p className="text-gray-500 mb-6">Find items to deliver on your next trip</p>

      {/* Filters */}
      <div className="flex flex-wrap gap-3 mb-6">
        <div className="flex items-center gap-2 bg-white border rounded-lg px-3 py-2">
          <MapPin className="w-4 h-4 text-gray-400" />
          <input className="text-sm outline-none" placeholder="Pickup country" value={pickupCountry} onChange={(e) => setPickupCountry(e.target.value)} />
        </div>
        <div className="flex items-center gap-2 bg-white border rounded-lg px-3 py-2">
          <MapPin className="w-4 h-4 text-gray-400" />
          <input className="text-sm outline-none" placeholder="Delivery country" value={deliveryCountry} onChange={(e) => setDeliveryCountry(e.target.value)} />
        </div>
      </div>

      {isLoading && <p className="text-gray-400 py-8 text-center">Loading...</p>}

      {!isLoading && requests.length === 0 && (
        <div className="text-center py-16 text-gray-400">
          <Search className="w-12 h-12 mx-auto mb-3 opacity-40" />
          <p>No pending delivery requests found.</p>
        </div>
      )}

      <div className="grid gap-4 sm:grid-cols-2">
        {requests.map((req) => (
          <Link
            key={req.id}
            href={`/traveler/requests/${req.id}`}
            className="bg-white rounded-xl border p-5 hover:shadow-md transition"
          >
            <div className="flex items-start justify-between mb-3">
              <div className="flex items-center gap-2">
                <Package className="w-5 h-5 text-orange-500" />
                <span className="font-semibold">{req.item_name}</span>
              </div>
              <span className="text-green-600 font-medium text-sm">
                {req.currency} {req.reward}
              </span>
            </div>
            <p className="text-sm text-gray-500 mb-2">
              {req.pickup_city}, {req.pickup_country} &rarr; {req.delivery_city}, {req.delivery_country}
            </p>
            <div className="flex items-center gap-4 text-xs text-gray-400">
              <span>Item value: {req.currency} {req.item_price}</span>
              {req.item_weight && <span>{req.item_weight} kg</span>}
              <span>{new Date(req.created_at).toLocaleDateString()}</span>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
