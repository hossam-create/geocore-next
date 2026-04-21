"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import axios from "axios";
import { ArrowLeft, Tag, DollarSign, Clock, Check, X, Send, User, MessageSquare } from "lucide-react";
import { useAuthStore } from "@/store/auth";

interface Offer {
  id: string;
  request_id: string;
  seller_id: string;
  price: number;
  description: string;
  delivery_days: number | null;
  counter_price: number | null;
  message: string | null;
  expires_at: string | null;
  responded_at: string | null;
  status: string;
  created_at: string;
}

interface ReverseAuctionRequest {
  id: string;
  buyer_id: string;
  title: string;
  description: string;
  category_id: string | null;
  max_budget: number | null;
  deadline: string;
  status: string;
  images: string;
  created_at: string;
  offers: Offer[];
}

export default function ReverseAuctionDetailPage() {
  const { id } = useParams<{ id: string }>();
  const qc = useQueryClient();
  const { user } = useAuthStore();

  const [offerPrice, setOfferPrice] = useState("");
  const [offerDesc, setOfferDesc] = useState("");
  const [offerDays, setOfferDays] = useState("");
  const [msg, setMsg] = useState("");
  const [counterOfferId, setCounterOfferId] = useState<string | null>(null);
  const [counterPrice, setCounterPrice] = useState("");
  const [counterMsg, setCounterMsg] = useState("");

  const { data: request, isLoading } = useQuery<ReverseAuctionRequest>({
    queryKey: ["reverse-auction", id],
    queryFn: async () => {
      const { data } = await axios.get(`/api/v1/reverse-auctions/${id}`);
      return data.data;
    },
  });

  const submitOffer = useMutation({
    mutationFn: async () => {
      const body: Record<string, unknown> = {
        price: parseFloat(offerPrice),
        description: offerDesc,
      };
      if (offerDays) body.delivery_days = parseInt(offerDays);
      return axios.post(`/api/v1/reverse-auctions/${id}/offers`, body);
    },
    onSuccess: () => {
      setMsg("Offer submitted!");
      setOfferPrice("");
      setOfferDesc("");
      setOfferDays("");
      qc.invalidateQueries({ queryKey: ["reverse-auction", id] });
    },
    onError: (err: any) => {
      setMsg(err?.response?.data?.message || "Failed to submit offer");
    },
  });

  const acceptOffer = useMutation({
    mutationFn: (offerId: string) =>
      axios.put(`/api/v1/reverse-auctions/${id}/offers/${offerId}/accept`),
    onSuccess: () => {
      setMsg("Offer accepted!");
      qc.invalidateQueries({ queryKey: ["reverse-auction", id] });
    },
    onError: (err: any) => setMsg(err?.response?.data?.message || "Failed"),
  });

  const rejectOffer = useMutation({
    mutationFn: (offerId: string) =>
      axios.put(`/api/v1/reverse-auctions/${id}/offers/${offerId}/reject`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["reverse-auction", id] });
    },
    onError: (err: any) => setMsg(err?.response?.data?.message || "Failed"),
  });

  const withdrawOffer = useMutation({
    mutationFn: (offerId: string) =>
      axios.delete(`/api/v1/reverse-auctions/${id}/offers/${offerId}`),
    onSuccess: () => {
      setMsg("Offer withdrawn");
      qc.invalidateQueries({ queryKey: ["reverse-auction", id] });
    },
    onError: (err: any) => setMsg(err?.response?.data?.message || "Failed"),
  });

  const counterOffer = useMutation({
    mutationFn: ({ offerId, price, message }: { offerId: string; price: number; message: string }) =>
      axios.put(`/api/v1/reverse-auctions/${id}/offers/${offerId}/counter`, { counter_price: price, message }),
    onSuccess: () => {
      setMsg("Counter offer sent!");
      setCounterOfferId(null);
      setCounterPrice("");
      setCounterMsg("");
      qc.invalidateQueries({ queryKey: ["reverse-auction", id] });
    },
    onError: (err: any) => setMsg(err?.response?.data?.message || "Failed"),
  });

  const respondToCounter = useMutation({
    mutationFn: ({ offerId, accept }: { offerId: string; accept: boolean }) =>
      axios.put(`/api/v1/reverse-auctions/${id}/offers/${offerId}/respond`, { accept }),
    onSuccess: () => {
      setMsg("Response submitted!");
      qc.invalidateQueries({ queryKey: ["reverse-auction", id] });
    },
    onError: (err: any) => setMsg(err?.response?.data?.message || "Failed"),
  });

  if (isLoading) return <div className="text-center py-16 text-gray-400">Loading...</div>;
  if (!request) return <div className="text-center py-16 text-gray-400">Not found</div>;

  const isOwner = user?.id === request.buyer_id;
  const isOpen = request.status === "open";
  const isExpired = new Date(request.deadline) < new Date();
  const offers = request.offers ?? [];
  const myOffer = offers.find((o) => o.seller_id === user?.id);

  const statusColor: Record<string, string> = {
    open: "bg-green-100 text-green-700",
    fulfilled: "bg-blue-100 text-blue-700",
    closed: "bg-gray-100 text-gray-600",
    expired: "bg-red-100 text-red-600",
  };

  return (
    <div className="max-w-3xl mx-auto px-4 py-8">
      <Link
        href="/reverse-auctions"
        className="flex items-center gap-2 text-gray-500 hover:text-gray-700 mb-6 text-sm"
      >
        <ArrowLeft className="w-4 h-4" /> Back to Requests
      </Link>

      {/* Request Details */}
      <div className="bg-white rounded-xl border p-6 mb-6">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <Tag className="w-5 h-5 text-purple-600" />
            <h1 className="text-xl font-bold">{request.title}</h1>
          </div>
          <span
            className={`text-sm px-3 py-1 rounded-full font-medium ${
              statusColor[request.status] ?? "bg-gray-100"
            }`}
          >
            {request.status}
          </span>
        </div>

        {request.description && (
          <p className="text-gray-600 text-sm mb-4 whitespace-pre-line">{request.description}</p>
        )}

        <div className="grid grid-cols-2 sm:grid-cols-3 gap-4 text-sm">
          {request.max_budget && (
            <div>
              <span className="text-gray-500 flex items-center gap-1">
                <DollarSign className="w-3 h-3" /> Max Budget
              </span>
              <p className="font-semibold">{request.max_budget.toLocaleString()}</p>
            </div>
          )}
          <div>
            <span className={`text-gray-500 flex items-center gap-1 ${isExpired ? "text-red-500" : ""}`}>
              <Clock className="w-3 h-3" /> Deadline
            </span>
            <p className={`font-semibold ${isExpired ? "text-red-600" : ""}`}>
              {new Date(request.deadline).toLocaleString()}
            </p>
          </div>
          <div>
            <span className="text-gray-500">Offers</span>
            <p className="font-semibold">{offers.length}</p>
          </div>
        </div>
      </div>

      {msg && (
        <div
          className={`rounded-lg p-3 mb-4 text-sm ${
            msg.includes("success") || msg.includes("accepted") || msg.includes("submitted")
              ? "bg-green-50 text-green-700 border border-green-200"
              : "bg-red-50 text-red-600 border border-red-200"
          }`}
        >
          {msg}
        </div>
      )}

      {/* Offers List */}
      <div className="mb-6">
        <h2 className="font-semibold text-lg mb-3">
          Offers ({offers.length})
        </h2>
        {offers.length === 0 ? (
          <div className="text-center py-8 text-gray-400 bg-gray-50 rounded-xl">
            <p>No offers yet. Be the first to make one!</p>
          </div>
        ) : (
          <div className="space-y-3">
            {offers
              .sort((a, b) => a.price - b.price)
              .map((offer) => {
                const offerStatusColor: Record<string, string> = {
                  pending: "bg-yellow-100 text-yellow-700",
                  accepted: "bg-green-100 text-green-700",
                  rejected: "bg-red-100 text-red-600",
                  withdrawn: "bg-gray-100 text-gray-500",
                  countered: "bg-blue-100 text-blue-700",
                  expired: "bg-orange-100 text-orange-600",
                };

                return (
                  <div key={offer.id} className="bg-white rounded-xl border p-4">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 bg-purple-100 rounded-full flex items-center justify-center">
                          <User className="w-4 h-4 text-purple-600" />
                        </div>
                        <div>
                          <p className="font-bold text-lg">{offer.price.toLocaleString()}</p>
                          {offer.delivery_days && (
                            <p className="text-xs text-gray-400">
                              {offer.delivery_days} days delivery
                            </p>
                          )}
                        </div>
                      </div>
                      <span
                        className={`text-xs px-2 py-1 rounded-full font-medium ${
                          offerStatusColor[offer.status] ?? "bg-gray-100"
                        }`}
                      >
                        {offer.status}
                      </span>
                    </div>
                    {offer.description && (
                      <p className="text-sm text-gray-600 mb-2">{offer.description}</p>
                    )}

                    {/* Counter offer info */}
                    {offer.status === "countered" && offer.counter_price && (
                      <div className="bg-blue-50 border border-blue-200 rounded-lg p-2 mb-2 text-xs text-blue-700">
                        <MessageSquare className="w-3 h-3 inline mr-1" />
                        Counter: <strong>{offer.counter_price.toLocaleString()}</strong>
                        {offer.message && <span className="ml-1">— {offer.message}</span>}
                        {offer.expires_at && (
                          <span className="ml-2 text-blue-500">Expires: {new Date(offer.expires_at).toLocaleString()}</span>
                        )}
                      </div>
                    )}

                    {/* Owner actions: accept, reject, or counter */}
                    {isOwner && offer.status === "pending" && isOpen && (
                      <div className="flex flex-wrap gap-2 mt-3">
                        <button
                          onClick={() => acceptOffer.mutate(offer.id)}
                          disabled={acceptOffer.isPending}
                          className="flex items-center gap-1 bg-green-600 text-white text-xs px-3 py-1.5 rounded-lg hover:bg-green-700 disabled:opacity-50 transition"
                        >
                          <Check className="w-3 h-3" /> Accept
                        </button>
                        <button
                          onClick={() => setCounterOfferId(counterOfferId === offer.id ? null : offer.id)}
                          className="flex items-center gap-1 bg-blue-100 text-blue-700 text-xs px-3 py-1.5 rounded-lg hover:bg-blue-200 transition"
                        >
                          <MessageSquare className="w-3 h-3" /> Counter
                        </button>
                        <button
                          onClick={() => rejectOffer.mutate(offer.id)}
                          disabled={rejectOffer.isPending}
                          className="flex items-center gap-1 bg-gray-100 text-gray-700 text-xs px-3 py-1.5 rounded-lg hover:bg-gray-200 disabled:opacity-50 transition"
                        >
                          <X className="w-3 h-3" /> Reject
                        </button>
                      </div>
                    )}

                    {/* Counter offer form */}
                    {counterOfferId === offer.id && (
                      <div className="mt-3 bg-blue-50 rounded-lg p-3 space-y-2">
                        <div className="grid grid-cols-2 gap-2">
                          <input
                            type="number"
                            step="0.01"
                            className="border rounded-lg px-2 py-1.5 text-xs"
                            value={counterPrice}
                            onChange={(e) => setCounterPrice(e.target.value)}
                            placeholder="Your counter price"
                          />
                          <input
                            className="border rounded-lg px-2 py-1.5 text-xs"
                            value={counterMsg}
                            onChange={(e) => setCounterMsg(e.target.value)}
                            placeholder="Message (optional)"
                          />
                        </div>
                        <button
                          onClick={() => counterOffer.mutate({ offerId: offer.id, price: parseFloat(counterPrice), message: counterMsg })}
                          disabled={counterOffer.isPending || !counterPrice}
                          className="bg-blue-600 text-white text-xs px-3 py-1.5 rounded-lg hover:bg-blue-700 disabled:opacity-50 transition"
                        >
                          {counterOffer.isPending ? "Sending..." : "Send Counter"}
                        </button>
                      </div>
                    )}

                    {/* Owner can also accept countered offers */}
                    {isOwner && offer.status === "countered" && isOpen && (
                      <div className="flex gap-2 mt-3">
                        <button
                          onClick={() => acceptOffer.mutate(offer.id)}
                          disabled={acceptOffer.isPending}
                          className="flex items-center gap-1 bg-green-600 text-white text-xs px-3 py-1.5 rounded-lg hover:bg-green-700 disabled:opacity-50 transition"
                        >
                          <Check className="w-3 h-3" /> Accept at counter price
                        </button>
                      </div>
                    )}

                    {/* Seller sees countered offer and can accept/decline */}
                    {offer.seller_id === user?.id && offer.status === "countered" && (
                      <div className="flex gap-2 mt-3">
                        <button
                          onClick={() => respondToCounter.mutate({ offerId: offer.id, accept: true })}
                          disabled={respondToCounter.isPending}
                          className="flex items-center gap-1 bg-green-600 text-white text-xs px-3 py-1.5 rounded-lg hover:bg-green-700 disabled:opacity-50 transition"
                        >
                          <Check className="w-3 h-3" /> Accept counter
                        </button>
                        <button
                          onClick={() => respondToCounter.mutate({ offerId: offer.id, accept: false })}
                          disabled={respondToCounter.isPending}
                          className="flex items-center gap-1 bg-gray-100 text-gray-700 text-xs px-3 py-1.5 rounded-lg hover:bg-gray-200 disabled:opacity-50 transition"
                        >
                          <X className="w-3 h-3" /> Decline
                        </button>
                      </div>
                    )}

                    {/* Seller can withdraw pending or countered offers */}
                    {offer.seller_id === user?.id && (offer.status === "pending" || offer.status === "countered") && (
                      <button
                        onClick={() => withdrawOffer.mutate(offer.id)}
                        disabled={withdrawOffer.isPending}
                        className="flex items-center gap-1 text-xs text-red-500 hover:text-red-700 mt-2 transition"
                      >
                        <X className="w-3 h-3" /> Withdraw my offer
                      </button>
                    )}
                  </div>
                );
              })}
          </div>
        )}
      </div>

      {/* Submit Offer Form (for sellers, not the owner) */}
      {!isOwner && isOpen && !isExpired && !myOffer && (
        <div className="bg-white rounded-xl border p-5">
          <h3 className="font-semibold mb-3 flex items-center gap-2">
            <Send className="w-4 h-4 text-purple-600" /> Submit Your Offer
          </h3>
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">Price *</label>
                <input
                  type="number"
                  step="0.01"
                  className="w-full border rounded-lg px-3 py-2 text-sm"
                  value={offerPrice}
                  onChange={(e) => setOfferPrice(e.target.value)}
                  placeholder="Your price"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">
                  Delivery (days)
                </label>
                <input
                  type="number"
                  className="w-full border rounded-lg px-3 py-2 text-sm"
                  value={offerDays}
                  onChange={(e) => setOfferDays(e.target.value)}
                  placeholder="e.g. 3"
                />
              </div>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Description</label>
              <textarea
                className="w-full border rounded-lg px-3 py-2 text-sm"
                rows={2}
                value={offerDesc}
                onChange={(e) => setOfferDesc(e.target.value)}
                placeholder="Describe your offer..."
              />
            </div>
            <button
              onClick={() => submitOffer.mutate()}
              disabled={submitOffer.isPending || !offerPrice}
              className="w-full bg-purple-600 text-white py-2.5 rounded-lg font-medium hover:bg-purple-700 disabled:opacity-50 transition text-sm"
            >
              {submitOffer.isPending ? "Submitting..." : "Submit Offer"}
            </button>
          </div>
        </div>
      )}

      {myOffer && myOffer.status !== "withdrawn" && (
        <div className="bg-purple-50 border border-purple-200 rounded-xl p-4 text-sm text-purple-700">
          You already submitted an offer of <strong>{myOffer.price.toLocaleString()}</strong> — status: <strong>{myOffer.status}</strong>
        </div>
      )}
    </div>
  );
}
