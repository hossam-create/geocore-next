"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import { disputesApi } from "@/lib/api";
import { useToastStore } from "@/lib/toast";
import { ArrowLeft, MessageSquare, DollarSign, AlertTriangle, CheckCircle } from "lucide-react";

export default function DisputeDetailPage() {
  const params = useParams();
  const id = params.id as string;
  const router = useRouter();
  const qc = useQueryClient();
  const toast = useToastStore();
  const [reply, setReply] = useState("");

  const { data: dispute, isLoading } = useQuery({
    queryKey: ["dispute", id],
    queryFn: () => disputesApi.get(id),
    enabled: !!id,
  });

  const resolveMutation = useMutation({
    mutationFn: (data: { resolution: string; refund_amount?: number }) =>
      disputesApi.resolve(id, data),
    onSuccess: () => {
      toast.showToast({ title: "Dispute resolved", type: "success" });
      qc.invalidateQueries({ queryKey: ["dispute", id] });
    },
    onError: () => toast.showToast({ title: "Failed to resolve dispute", type: "error" }),
  });

  const replyMutation = useMutation({
    mutationFn: (body: string) => disputesApi.reply(id, body) as Promise<any>,
    onSuccess: () => {
      setReply("");
      qc.invalidateQueries({ queryKey: ["dispute", id] });
    },
    onError: () => toast.showToast({ title: "Failed to send reply", type: "error" }),
  });

  if (isLoading) return <div className="p-8 text-center text-gray-500">Loading dispute...</div>;
  if (!dispute) return <div className="p-8 text-center text-gray-500">Dispute not found</div>;

  return (
    <div className="space-y-6">
      <PageHeader
        title={`Dispute #${id.slice(0, 8)}`}
        actions={
          <button onClick={() => router.back()} className="flex items-center gap-2 text-sm text-gray-500 hover:text-gray-800">
            <ArrowLeft className="w-4 h-4" /> Back
          </button>
        }
      />

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main content */}
        <div className="lg:col-span-2 space-y-6">
          <div className="bg-white rounded-xl border p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-semibold text-lg">Dispute Details</h3>
              <StatusBadge status={dispute.status} />
            </div>
            <dl className="grid grid-cols-2 gap-4 text-sm">
              <div><dt className="text-gray-500">Type</dt><dd className="font-medium">{dispute.type || "general"}</dd></div>
              <div><dt className="text-gray-500">Order ID</dt><dd className="font-mono text-xs">{dispute.order_id || "—"}</dd></div>
              <div><dt className="text-gray-500">Buyer</dt><dd>{dispute.buyer_name || dispute.buyer_id || "—"}</dd></div>
              <div><dt className="text-gray-500">Seller</dt><dd>{dispute.seller_name || dispute.seller_id || "—"}</dd></div>
              <div><dt className="text-gray-500">Amount</dt><dd className="font-semibold">${dispute.amount || 0}</dd></div>
              <div><dt className="text-gray-500">Created</dt><dd>{new Date(dispute.created_at).toLocaleDateString()}</dd></div>
            </dl>
          </div>

          {/* Messages */}
          <div className="bg-white rounded-xl border p-6">
            <h3 className="font-semibold mb-4 flex items-center gap-2"><MessageSquare className="w-4 h-4" /> Messages</h3>
            <div className="space-y-3 max-h-64 overflow-y-auto">
              {(dispute.messages || []).map((msg: any, i: number) => (
                <div key={i} className={`p-3 rounded-lg text-sm ${msg.is_admin ? "bg-blue-50 border-l-4 border-blue-400" : "bg-gray-50"}`}>
                  <div className="flex justify-between mb-1">
                    <span className="font-medium">{msg.sender_name || (msg.is_admin ? "Admin" : "User")}</span>
                    <span className="text-xs text-gray-400">{new Date(msg.created_at).toLocaleString()}</span>
                  </div>
                  <p>{msg.body}</p>
                </div>
              ))}
              {(!dispute.messages || dispute.messages.length === 0) && (
                <p className="text-gray-400 text-sm">No messages yet</p>
              )}
            </div>
          </div>

          {/* Reply */}
          {dispute.status !== "resolved" && (
            <div className="bg-white rounded-xl border p-6">
              <h3 className="font-semibold mb-3">Add Reply</h3>
              <textarea
                value={reply}
                onChange={(e) => setReply(e.target.value)}
                className="w-full border rounded-lg p-3 text-sm h-24 resize-none"
                placeholder="Type your response..."
              />
              <button
                onClick={() => reply.length > 0 && replyMutation.mutate(reply)}
                disabled={replyMutation.isPending || reply.length === 0}
                className="mt-2 px-4 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-700 disabled:opacity-50"
              >
                Send Reply
              </button>
            </div>
          )}
        </div>

        {/* Sidebar actions */}
        <div className="space-y-6">
          <div className="bg-white rounded-xl border p-6">
            <h3 className="font-semibold mb-4 flex items-center gap-2"><AlertTriangle className="w-4 h-4 text-amber-500" /> Actions</h3>
            {dispute.status !== "resolved" && (
              <div className="space-y-3">
                <button
                  onClick={() => resolveMutation.mutate({ resolution: "resolved_favor_buyer" })}
                  disabled={resolveMutation.isPending}
                  className="w-full px-4 py-2 bg-green-600 text-white rounded-lg text-sm hover:bg-green-700 disabled:opacity-50 flex items-center justify-center gap-2"
                >
                  <CheckCircle className="w-4 h-4" /> Resolve (Favor Buyer)
                </button>
                <button
                  onClick={() => resolveMutation.mutate({ resolution: "resolved_favor_seller" })}
                  disabled={resolveMutation.isPending}
                  className="w-full px-4 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-700 disabled:opacity-50 flex items-center justify-center gap-2"
                >
                  <CheckCircle className="w-4 h-4" /> Resolve (Favor Seller)
                </button>
                <button
                  onClick={() => resolveMutation.mutate({ resolution: "split", refund_amount: (dispute.amount || 0) / 2 })}
                  disabled={resolveMutation.isPending}
                  className="w-full px-4 py-2 bg-amber-600 text-white rounded-lg text-sm hover:bg-amber-700 disabled:opacity-50 flex items-center justify-center gap-2"
                >
                  <DollarSign className="w-4 h-4" /> Split Resolution
                </button>
              </div>
            )}
            {dispute.status === "resolved" && (
              <p className="text-sm text-green-600 font-medium">✓ This dispute has been resolved</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
