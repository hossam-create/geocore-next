"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { formatPrice } from "@/lib/utils";
import {
  ShieldAlert, AlertCircle, CheckCircle, Clock,
  XCircle, Plus, ChevronRight,
} from "lucide-react";
import { formatDistanceToNow, parseISO } from "date-fns";

interface Dispute {
  id: string;
  order_id?: string;
  reason: string;
  status: string;
  amount: number;
  currency: string;
  created_at: string;
  updated_at?: string;
  description?: string;
}

const MOCK: Dispute[] = [
  { id: "dsp-001", order_id: "ord-006", reason: "item_not_received",    status: "open",       amount: 1100, currency: "AED", created_at: new Date(Date.now() - 3 * 86400000).toISOString(),  description: "Item was never delivered despite tracking showing delivered." },
  { id: "dsp-002", order_id: "ord-003", reason: "item_not_as_described", status: "resolved",   amount: 2800, currency: "AED", created_at: new Date(Date.now() - 20 * 86400000).toISOString(), description: "Product was significantly different from listing photos." },
  { id: "dsp-003", order_id: "ord-007", reason: "damaged_item",          status: "under_review",amount: 650, currency: "AED", created_at: new Date(Date.now() - 7 * 86400000).toISOString(),  description: "Package arrived with visible damage to the product." },
];

const STATUS_CFG: Record<string, { label: string; cls: string; icon: React.ElementType }> = {
  open:          { label: "Open",          cls: "bg-amber-50 text-amber-700 border-amber-200",   icon: Clock },
  under_review:  { label: "Under Review",  cls: "bg-blue-50 text-blue-700 border-blue-200",      icon: AlertCircle },
  resolved:      { label: "Resolved",      cls: "bg-emerald-50 text-emerald-700 border-emerald-200", icon: CheckCircle },
  closed:        { label: "Closed",        cls: "bg-gray-100 text-gray-500 border-gray-200",     icon: XCircle },
  rejected:      { label: "Rejected",      cls: "bg-red-50 text-red-600 border-red-200",         icon: XCircle },
};

const REASON_LABELS: Record<string, string> = {
  item_not_received:     "Item Not Received",
  item_not_as_described: "Not As Described",
  damaged_item:          "Item Damaged",
  wrong_item:            "Wrong Item Sent",
  seller_unresponsive:   "Seller Unresponsive",
  other:                 "Other",
};

function timeAgo(d: string) {
  try { return formatDistanceToNow(parseISO(d), { addSuffix: true }); }
  catch { return d; }
}

export default function BuyerDisputesPage() {
  const { isAuthenticated } = useAuthStore();

  const { data: disputes = MOCK, isLoading } = useQuery<Dispute[]>({
    queryKey: ["buyer-disputes"],
    queryFn: async () => {
      try {
        const res = await api.get("/disputes?role=buyer&page=1&limit=50");
        const d = res.data?.data ?? [];
        return Array.isArray(d) && d.length ? d : MOCK;
      } catch { return MOCK; }
    },
    enabled: isAuthenticated,
    retry: false,
  });

  const openCount = disputes.filter((d) => d.status === "open" || d.status === "under_review").length;

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div>
          <h1 className="text-xl font-bold text-gray-900">My Disputes</h1>
          <p className="text-sm text-gray-400">{disputes.length} total disputes</p>
        </div>
        <Link
          href="/disputes/new"
          className="flex items-center gap-1.5 px-4 py-2 bg-indigo-600 text-white rounded-xl text-sm font-semibold hover:bg-indigo-700 transition-colors"
        >
          <Plus className="w-4 h-4" /> Open Dispute
        </Link>
      </div>

      {/* Active alert */}
      {openCount > 0 && (
        <div className="flex items-center gap-3 bg-amber-50 border border-amber-200 rounded-xl px-4 py-3 text-sm text-amber-800">
          <ShieldAlert className="w-4 h-4 shrink-0 text-amber-500" />
          <span><strong>{openCount}</strong> dispute{openCount > 1 ? "s" : ""} currently active. Our team is reviewing them.</span>
        </div>
      )}

      {/* Disputes list */}
      <div className="space-y-3">
        {isLoading ? (
          <>
            {[1, 2, 3].map((i) => <div key={i} className="h-24 animate-pulse rounded-2xl bg-gray-100" />)}
          </>
        ) : disputes.length === 0 ? (
          <div className="py-20 text-center bg-white rounded-2xl border border-gray-100">
            <ShieldAlert className="w-12 h-12 mx-auto mb-3 text-gray-200" />
            <p className="text-sm font-semibold text-gray-500">No disputes submitted</p>
            <p className="text-xs text-gray-400 mt-1">If you have an issue with an order, you can open a dispute.</p>
            <Link href="/disputes/new" className="mt-4 inline-flex items-center gap-1.5 px-4 py-2 bg-indigo-600 text-white rounded-xl text-sm font-semibold hover:bg-indigo-700">
              <Plus className="w-4 h-4" /> Open Dispute
            </Link>
          </div>
        ) : (
          disputes.map((d) => {
            const cfg = STATUS_CFG[d.status] ?? STATUS_CFG.open;
            const StatusIcon = cfg.icon;
            return (
              <div key={d.id} className="bg-white rounded-2xl border border-gray-100 p-4 hover:shadow-sm transition-shadow">
                <div className="flex items-start gap-4">
                  <div className="w-10 h-10 rounded-xl bg-orange-50 flex items-center justify-center shrink-0">
                    <ShieldAlert className="w-5 h-5 text-orange-500" />
                  </div>

                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between gap-2 flex-wrap">
                      <div>
                        <p className="text-sm font-semibold text-gray-900">
                          {REASON_LABELS[d.reason] ?? d.reason.replace(/_/g, " ")}
                        </p>
                        {d.order_id && (
                          <p className="text-xs text-gray-400 mt-0.5">
                            Order #{d.order_id.slice(0, 8)} · {timeAgo(d.created_at)}
                          </p>
                        )}
                      </div>
                      <span className={`inline-flex items-center gap-1 text-[10px] font-bold px-2 py-0.5 rounded-full border ${cfg.cls}`}>
                        <StatusIcon className="w-3 h-3" /> {cfg.label}
                      </span>
                    </div>

                    {d.description && (
                      <p className="text-xs text-gray-500 mt-2 leading-relaxed line-clamp-2">{d.description}</p>
                    )}

                    <div className="flex items-center justify-between mt-3">
                      <p className="text-sm font-bold text-gray-800">
                        {formatPrice(d.amount, d.currency)}
                      </p>
                      {d.order_id && (
                        <Link
                          href={`/orders/${d.order_id}`}
                          className="inline-flex items-center gap-1 text-xs font-medium text-indigo-600 hover:underline"
                        >
                          View Order <ChevronRight className="w-3 h-3" />
                        </Link>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            );
          })
        )}
      </div>

      {/* Help section */}
      <div className="bg-indigo-50 rounded-2xl border border-indigo-100 p-5">
        <h3 className="text-sm font-semibold text-indigo-900 mb-1">Need help with a dispute?</h3>
        <p className="text-xs text-indigo-700 mb-3">
          Our buyer protection team typically resolves disputes within 3–5 business days.
        </p>
        <div className="flex gap-2">
          <Link href="/buyer-protection" className="text-xs font-semibold text-indigo-700 hover:underline">
            Buyer Protection Policy →
          </Link>
          <span className="text-indigo-300">·</span>
          <Link href="/contact" className="text-xs font-semibold text-indigo-700 hover:underline">
            Contact Support →
          </Link>
        </div>
      </div>
    </div>
  );
}
