"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import axios from "axios";
import { Search, Plus, Clock, DollarSign, Tag } from "lucide-react";

interface ReverseAuctionRequest {
  id: string;
  buyer_id: string;
  title: string;
  description: string;
  category_id: string | null;
  max_budget: number | null;
  deadline: string;
  status: string;
  created_at: string;
}

export default function ReverseAuctionsPage() {
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("open");

  const { data: requests = [], isLoading } = useQuery<ReverseAuctionRequest[]>({
    queryKey: ["reverse-auctions", status, search],
    queryFn: async () => {
      const params = new URLSearchParams({ status });
      if (search) params.set("q", search);
      const { data } = await axios.get(`/api/v1/reverse-auctions?${params}`);
      return data.data ?? [];
    },
  });

  const isExpired = (deadline: string) => new Date(deadline) < new Date();

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Product Requests</h1>
          <p className="text-gray-500 text-sm">Browse buyer requests and submit your offer</p>
        </div>
        <Link
          href="/reverse-auctions/new"
          className="flex items-center gap-2 bg-purple-600 text-white px-4 py-2 rounded-lg hover:bg-purple-700 transition text-sm font-medium"
        >
          <Plus className="w-4 h-4" /> Request a Product
        </Link>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-3 mb-6">
        <div className="flex items-center gap-2 bg-white border rounded-lg px-3 py-2 flex-1 min-w-[200px]">
          <Search className="w-4 h-4 text-gray-400" />
          <input
            className="text-sm outline-none w-full"
            placeholder="Search requests..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <div className="flex gap-1">
          {["open", "fulfilled", "closed"].map((s) => (
            <button
              key={s}
              onClick={() => setStatus(s)}
              className={`text-xs px-3 py-1.5 rounded-full font-medium capitalize transition ${
                status === s
                  ? "bg-purple-600 text-white"
                  : "bg-gray-100 text-gray-600 hover:bg-gray-200"
              }`}
            >
              {s}
            </button>
          ))}
        </div>
      </div>

      {isLoading && <p className="text-center py-12 text-gray-400">Loading...</p>}

      {!isLoading && requests.length === 0 && (
        <div className="text-center py-16 text-gray-400">
          <Tag className="w-12 h-12 mx-auto mb-3 opacity-40" />
          <p>No requests found.</p>
        </div>
      )}

      <div className="space-y-3">
        {requests.map((req) => (
          <Link
            key={req.id}
            href={`/reverse-auctions/${req.id}`}
            className="block bg-white rounded-xl border p-5 hover:shadow-md transition"
          >
            <div className="flex items-start justify-between mb-2">
              <h3 className="font-semibold text-gray-800">{req.title}</h3>
              <span
                className={`text-xs px-2 py-1 rounded-full font-medium ${
                  req.status === "open"
                    ? "bg-green-100 text-green-700"
                    : req.status === "fulfilled"
                    ? "bg-blue-100 text-blue-700"
                    : "bg-gray-100 text-gray-600"
                }`}
              >
                {req.status}
              </span>
            </div>
            {req.description && (
              <p className="text-sm text-gray-500 line-clamp-2 mb-3">{req.description}</p>
            )}
            <div className="flex items-center gap-4 text-xs text-gray-400">
              {req.max_budget && (
                <span className="flex items-center gap-1">
                  <DollarSign className="w-3 h-3" /> Budget: {req.max_budget.toLocaleString()}
                </span>
              )}
              <span
                className={`flex items-center gap-1 ${
                  isExpired(req.deadline) ? "text-red-400" : ""
                }`}
              >
                <Clock className="w-3 h-3" />{" "}
                {isExpired(req.deadline)
                  ? "Expired"
                  : `Deadline: ${new Date(req.deadline).toLocaleDateString()}`}
              </span>
              <span>{new Date(req.created_at).toLocaleDateString()}</span>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
