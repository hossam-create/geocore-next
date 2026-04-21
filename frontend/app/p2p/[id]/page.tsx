"use client";

import { useState, useRef, useEffect } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import axios from "axios";
import { ArrowLeft, ArrowLeftRight, Shield, Send, CheckCircle, XCircle } from "lucide-react";

interface ExchangeRequest {
  id: string;
  user_id: string;
  from_currency: string;
  to_currency: string;
  from_amount: number;
  to_amount: number;
  desired_rate: number;
  use_escrow: boolean;
  notes: string;
  status: string;
  matched_user_id: string | null;
  matched_at: string | null;
  completed_at: string | null;
  created_at: string;
}

interface Message {
  id: string;
  sender_id: string;
  body: string;
  created_at: string;
}

export default function ExchangeDetailPage() {
  const { id } = useParams<{ id: string }>();
  const qc = useQueryClient();
  const [msg, setMsg] = useState("");
  const chatEnd = useRef<HTMLDivElement>(null);

  const { data: er, isLoading } = useQuery<ExchangeRequest>({
    queryKey: ["p2p-request", id],
    queryFn: async () => {
      const { data } = await axios.get(`/api/v1/p2p/requests/${id}`);
      return data.data;
    },
  });

  const { data: messages = [] } = useQuery<Message[]>({
    queryKey: ["p2p-messages", id],
    queryFn: async () => {
      const { data } = await axios.get(`/api/v1/p2p/requests/${id}/messages`);
      return data.data ?? [];
    },
    refetchInterval: 5000,
    enabled: !!er && er.status !== "open",
  });

  useEffect(() => {
    chatEnd.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const acceptMut = useMutation({
    mutationFn: () => axios.post(`/api/v1/p2p/requests/${id}/accept`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["p2p-request", id] }),
  });

  const completeMut = useMutation({
    mutationFn: () => axios.post(`/api/v1/p2p/requests/${id}/complete`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["p2p-request", id] }),
  });

  const cancelMut = useMutation({
    mutationFn: () => axios.post(`/api/v1/p2p/requests/${id}/cancel`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["p2p-request", id] }),
  });

  const sendMut = useMutation({
    mutationFn: (body: string) => axios.post(`/api/v1/p2p/requests/${id}/messages`, { body }),
    onSuccess: () => {
      setMsg("");
      qc.invalidateQueries({ queryKey: ["p2p-messages", id] });
    },
  });

  const statusColor: Record<string, string> = {
    open: "bg-green-100 text-green-700",
    matched: "bg-blue-100 text-blue-700",
    escrow: "bg-yellow-100 text-yellow-700",
    completed: "bg-emerald-100 text-emerald-700",
    cancelled: "bg-red-100 text-red-600",
    disputed: "bg-orange-100 text-orange-700",
  };

  if (isLoading) return <div className="text-center py-16 text-gray-400">Loading...</div>;
  if (!er) return <div className="text-center py-16 text-gray-400">Not found</div>;

  return (
    <div className="max-w-3xl mx-auto px-4 py-8">
      <Link href="/p2p" className="flex items-center gap-2 text-gray-500 hover:text-gray-700 mb-6 text-sm">
        <ArrowLeft className="w-4 h-4" /> Back to Marketplace
      </Link>

      {/* Request Card */}
      <div className="bg-white rounded-xl border p-6 mb-6">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <span className="bg-blue-100 text-blue-700 text-sm font-bold px-3 py-1 rounded">{er.from_currency}</span>
            <ArrowLeftRight className="w-5 h-5 text-gray-400" />
            <span className="bg-green-100 text-green-700 text-sm font-bold px-3 py-1 rounded">{er.to_currency}</span>
          </div>
          <span className={`text-sm px-3 py-1 rounded-full font-medium ${statusColor[er.status] ?? "bg-gray-100"}`}>
            {er.status}
          </span>
        </div>

        <div className="grid grid-cols-3 gap-4 text-sm mb-4">
          <div>
            <span className="text-gray-500">Selling</span>
            <p className="text-lg font-bold">{er.from_amount.toLocaleString()} {er.from_currency}</p>
          </div>
          <div>
            <span className="text-gray-500">Buying</span>
            <p className="text-lg font-bold">{er.to_amount.toLocaleString()} {er.to_currency}</p>
          </div>
          <div>
            <span className="text-gray-500">Rate</span>
            <p className="text-lg font-bold">{er.desired_rate.toFixed(4)}</p>
          </div>
        </div>

        {er.use_escrow && (
          <div className="flex items-center gap-2 text-sm text-yellow-700 bg-yellow-50 rounded-lg px-3 py-2 mb-4">
            <Shield className="w-4 h-4" /> Escrow protection enabled
          </div>
        )}

        {er.notes && <p className="text-sm text-gray-600 bg-gray-50 rounded-lg p-3">{er.notes}</p>}

        {/* Actions */}
        <div className="flex gap-3 mt-5">
          {er.status === "open" && (
            <button onClick={() => acceptMut.mutate()} disabled={acceptMut.isPending} className="flex items-center gap-2 bg-blue-600 text-white px-5 py-2 rounded-lg hover:bg-blue-700 disabled:opacity-50 transition text-sm">
              <CheckCircle className="w-4 h-4" /> Accept & Match
            </button>
          )}
          {er.status === "matched" && (
            <button onClick={() => completeMut.mutate()} disabled={completeMut.isPending} className="flex items-center gap-2 bg-emerald-600 text-white px-5 py-2 rounded-lg hover:bg-emerald-700 disabled:opacity-50 transition text-sm">
              <CheckCircle className="w-4 h-4" /> Confirm Completed
            </button>
          )}
          {(er.status === "open" || er.status === "matched") && (
            <button onClick={() => cancelMut.mutate()} disabled={cancelMut.isPending} className="flex items-center gap-2 bg-red-50 text-red-600 border border-red-200 px-5 py-2 rounded-lg hover:bg-red-100 disabled:opacity-50 transition text-sm">
              <XCircle className="w-4 h-4" /> Cancel
            </button>
          )}
        </div>
      </div>

      {/* Chat (visible after matching) */}
      {er.status !== "open" && (
        <div className="bg-white rounded-xl border overflow-hidden">
          <div className="px-5 py-3 border-b font-semibold text-sm">Chat</div>

          <div className="h-64 overflow-y-auto px-5 py-3 space-y-2">
            {messages.length === 0 && <p className="text-gray-400 text-sm text-center py-8">No messages yet</p>}
            {messages.map((m) => (
              <div key={m.id} className={`text-sm p-2 rounded-lg max-w-[80%] ${m.sender_id === er.user_id ? "bg-blue-50 ml-auto text-right" : "bg-gray-50"}`}>
                <p>{m.body}</p>
                <span className="text-xs text-gray-400">{new Date(m.created_at).toLocaleTimeString()}</span>
              </div>
            ))}
            <div ref={chatEnd} />
          </div>

          <form
            className="flex border-t"
            onSubmit={(e) => { e.preventDefault(); if (msg.trim()) sendMut.mutate(msg.trim()); }}
          >
            <input
              className="flex-1 px-4 py-3 text-sm outline-none"
              placeholder="Type a message..."
              value={msg}
              onChange={(e) => setMsg(e.target.value)}
            />
            <button type="submit" disabled={!msg.trim() || sendMut.isPending} className="px-4 text-blue-600 disabled:opacity-40">
              <Send className="w-5 h-5" />
            </button>
          </form>
        </div>
      )}
    </div>
  );
}
