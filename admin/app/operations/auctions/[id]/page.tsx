"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { auctionsApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import DataTable from "@/components/shared/DataTable";
import ConfirmDialog from "@/components/shared/ConfirmDialog";
import { useToastStore } from "@/lib/toast";
import {
  ArrowLeft, Check, X, Clock, Trash2, Gavel, DollarSign,
  User, Calendar, Timer,
} from "lucide-react";

interface AuctionDetail {
  id: string;
  title: string;
  description?: string;
  start_price: number;
  current_bid?: number;
  reserve_price?: number;
  bid_count?: number;
  status: string;
  auction_type?: string;
  seller_id?: string;
  seller_name?: string;
  winner_id?: string;
  starts_at?: string;
  ends_at: string;
  created_at: string;
  [key: string]: unknown;
}

interface Bid {
  id: string;
  user_id: string;
  user_name?: string;
  amount: number;
  created_at: string;
  [key: string]: unknown;
}

export default function AuctionDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [rejectDialog, setRejectDialog] = useState(false);
  const [cancelDialog, setCancelDialog] = useState(false);
  const [extendHours, setExtendHours] = useState(24);

  const { data: auction, isLoading } = useQuery<AuctionDetail>({
    queryKey: ["admin", "auction", id],
    queryFn: () => auctionsApi.get(id),
    enabled: !!id,
  });

  const { data: bids = [] } = useQuery<Bid[]>({
    queryKey: ["admin", "auction", id, "bids"],
    queryFn: async () => {
      try {
        const res = await auctionsApi.bids(id);
        return Array.isArray(res) ? res : [];
      } catch { return []; }
    },
    enabled: !!id,
  });

  const approveMut = useMutation({
    mutationFn: () => auctionsApi.approve(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["admin", "auction", id] }); showToast({ type: "success", title: "Auction approved" }); },
  });
  const rejectMut = useMutation({
    mutationFn: (reason?: string) => auctionsApi.reject(id, reason),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["admin", "auction", id] }); setRejectDialog(false); showToast({ type: "success", title: "Auction rejected" }); },
  });
  const cancelMut = useMutation({
    mutationFn: () => auctionsApi.cancel(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["admin", "auction", id] }); setCancelDialog(false); showToast({ type: "success", title: "Auction cancelled" }); },
  });
  const extendMut = useMutation({
    mutationFn: () => auctionsApi.extend(id, extendHours),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["admin", "auction", id] }); showToast({ type: "success", title: `Extended by ${extendHours}h` }); },
  });
  const deleteBidMut = useMutation({
    mutationFn: (bidId: string) => auctionsApi.deleteBid(id, bidId),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["admin", "auction", id, "bids"] }); showToast({ type: "success", title: "Bid removed" }); },
  });

  if (isLoading) return <div className="text-center py-20 text-slate-400">Loading auction...</div>;
  if (!auction) return <div className="text-center py-20 text-slate-400">Auction not found</div>;

  const reserveMet = auction.reserve_price ? (auction.current_bid ?? 0) >= auction.reserve_price : true;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <button onClick={() => router.push("/operations/auctions")} className="p-2 hover:bg-slate-100 rounded-lg"><ArrowLeft className="w-4 h-4 text-slate-500" /></button>
        <PageHeader title={auction.title} description={`ID: ${auction.id}`} />
      </div>

      {/* Status + Actions */}
      <div className="rounded-xl border border-slate-200 bg-white p-5">
        <div className="flex flex-wrap items-center gap-2 mb-4">
          <StatusBadge status={auction.status} dot />
          {auction.auction_type && <StatusBadge status={auction.auction_type} variant="info" />}
          {!reserveMet && <StatusBadge status="reserve not met" variant="warning" />}
        </div>
        <div className="flex flex-wrap gap-2">
          {auction.status === "pending" && (
            <>
              <button onClick={() => approveMut.mutate()} disabled={approveMut.isPending} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-green-50 text-green-600 hover:bg-green-100"><Check className="w-3.5 h-3.5" /> Approve</button>
              <button onClick={() => setRejectDialog(true)} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-red-50 text-red-500 hover:bg-red-100"><X className="w-3.5 h-3.5" /> Reject</button>
            </>
          )}
          {(auction.status === "active" || auction.status === "live") && (
            <button onClick={() => setCancelDialog(true)} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-red-50 text-red-500 hover:bg-red-100"><X className="w-3.5 h-3.5" /> Cancel</button>
          )}
        </div>
      </div>

      {/* Info Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="rounded-xl border border-slate-200 bg-white p-4">
          <div className="flex items-center gap-2 mb-1"><DollarSign className="w-4 h-4 text-slate-400" /><span className="text-xs text-slate-400">Start Price</span></div>
          <p className="text-lg font-bold text-slate-800">${auction.start_price?.toLocaleString()}</p>
        </div>
        <div className="rounded-xl border border-slate-200 bg-white p-4">
          <div className="flex items-center gap-2 mb-1"><Gavel className="w-4 h-4 text-indigo-500" /><span className="text-xs text-slate-400">Current Bid</span></div>
          <p className="text-lg font-bold text-indigo-600">${(auction.current_bid ?? 0).toLocaleString()}</p>
        </div>
        <div className="rounded-xl border border-slate-200 bg-white p-4">
          <div className="flex items-center gap-2 mb-1"><User className="w-4 h-4 text-slate-400" /><span className="text-xs text-slate-400">Bids</span></div>
          <p className="text-lg font-bold text-slate-800">{auction.bid_count ?? bids.length}</p>
        </div>
        <div className="rounded-xl border border-slate-200 bg-white p-4">
          <div className="flex items-center gap-2 mb-1"><Timer className="w-4 h-4 text-slate-400" /><span className="text-xs text-slate-400">Ends At</span></div>
          <p className="text-sm font-medium text-slate-700">{new Date(auction.ends_at).toLocaleString()}</p>
        </div>
      </div>

      {/* Extend */}
      <div className="rounded-xl border border-slate-200 bg-white p-5">
        <h3 className="text-sm font-semibold text-slate-700 mb-3">Extend Auction</h3>
        <div className="flex items-center gap-3">
          <select value={extendHours} onChange={(e) => setExtendHours(+e.target.value)} className="border rounded-lg px-3 py-2 text-sm">
            <option value={1}>1 hour</option>
            <option value={6}>6 hours</option>
            <option value={12}>12 hours</option>
            <option value={24}>24 hours</option>
            <option value={48}>48 hours</option>
            <option value={72}>72 hours</option>
          </select>
          <button onClick={() => extendMut.mutate()} disabled={extendMut.isPending} className="px-4 py-2 text-sm rounded-lg text-white bg-indigo-600 disabled:opacity-50"><Clock className="w-3.5 h-3.5 inline mr-1" /> Extend</button>
        </div>
      </div>

      {/* Bid History */}
      <div className="rounded-xl border border-slate-200 bg-white p-5">
        <h3 className="text-sm font-semibold text-slate-700 mb-3">Bid History</h3>
        <DataTable
          columns={[
            { key: "user_name", label: "Bidder", render: (b: Bid) => b.user_name ?? b.user_id },
            { key: "amount", label: "Amount", render: (b: Bid) => <span className="font-bold">${b.amount.toLocaleString()}</span> },
            { key: "created_at", label: "Time", render: (b: Bid) => new Date(b.created_at).toLocaleString() },
            { key: "actions", label: "", render: (b: Bid) => (
              <button onClick={() => { if (confirm("Delete this bid?")) deleteBidMut.mutate(b.id); }} className="p-1 hover:bg-red-50 rounded"><Trash2 className="w-3.5 h-3.5 text-red-400" /></button>
            )},
          ]}
          data={bids}
          emptyMessage="No bids yet."
          rowKey={(b: Bid) => b.id}
        />
      </div>

      <ConfirmDialog open={rejectDialog} title="Reject Auction" message="This auction will be rejected. The seller will be notified." confirmLabel="Reject" variant="danger" requireReason reasonLabel="Rejection Reason" onConfirm={(reason) => rejectMut.mutate(reason)} onCancel={() => setRejectDialog(false)} isLoading={rejectMut.isPending} />
      <ConfirmDialog open={cancelDialog} title="Cancel Auction" message="This will cancel the auction and notify all bidders. This cannot be undone." confirmLabel="Cancel Auction" variant="danger" onConfirm={() => cancelMut.mutate()} onCancel={() => setCancelDialog(false)} isLoading={cancelMut.isPending} />
    </div>
  );
}
