"use client";

import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import { Puzzle, Settings, Power } from "lucide-react";

interface Addon {
  name: string;
  slug: string;
  version: string;
  description: string;
  is_installed: boolean;
  is_active: boolean;
}

const ADDONS: Addon[] = [
  { name: "Crowdshipping", slug: "crowdshipping", version: "1.0.0", description: "Enable traveler-based shipping for cross-border delivery", is_installed: false, is_active: false },
  { name: "Crypto Payments", slug: "crypto-payments", version: "1.0.0", description: "Accept Bitcoin, Ethereum, and USDT payments", is_installed: false, is_active: false },
  { name: "AI Search", slug: "ai-search", version: "0.9.0", description: "AI-powered semantic search for listings and auctions", is_installed: false, is_active: false },
  { name: "AR Preview", slug: "ar-preview", version: "0.5.0", description: "Augmented reality product preview for mobile users", is_installed: false, is_active: false },
  { name: "Loyalty Points", slug: "loyalty-points", version: "1.0.0", description: "Reward users with points for purchases and referrals", is_installed: true, is_active: true },
  { name: "Livestream Selling", slug: "livestream", version: "0.8.0", description: "Live video selling with real-time bidding", is_installed: false, is_active: false },
  { name: "Multi-Currency", slug: "multi-currency", version: "1.0.0", description: "Support for multiple currencies with real-time exchange rates", is_installed: true, is_active: false },
  { name: "SMS Notifications", slug: "sms-notifications", version: "1.0.0", description: "Send SMS notifications via Twilio or local providers", is_installed: false, is_active: false },
];

export default function AddonsPage() {
  return (
    <div>
      <PageHeader title="Addons" description="Extend your platform with additional features" />

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {ADDONS.map((addon) => (
          <div key={addon.slug} className="rounded-xl border border-slate-200 bg-white p-5 hover:shadow-sm transition-shadow">
            <div className="flex items-start justify-between mb-3">
              <div className="flex items-center gap-3">
                <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${addon.is_active ? "bg-indigo-50" : "bg-slate-100"}`}>
                  <Puzzle className={`w-5 h-5 ${addon.is_active ? "text-indigo-500" : "text-slate-400"}`} />
                </div>
                <div>
                  <h3 className="text-sm font-semibold text-slate-800">{addon.name}</h3>
                  <p className="text-[10px] text-slate-400">v{addon.version}</p>
                </div>
              </div>
              {addon.is_active ? (
                <StatusBadge status="active" />
              ) : addon.is_installed ? (
                <StatusBadge status="installed" variant="neutral" />
              ) : (
                <StatusBadge status="available" variant="info" />
              )}
            </div>

            <p className="text-xs text-slate-500 mb-4 leading-relaxed">{addon.description}</p>

            <div className="flex gap-2 pt-3 border-t border-slate-100">
              {!addon.is_installed ? (
                <button className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-indigo-50 text-indigo-600 hover:bg-indigo-100">
                  <Power className="w-3 h-3" /> Install
                </button>
              ) : (
                <>
                  <button className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg ${addon.is_active ? "bg-red-50 text-red-500 hover:bg-red-100" : "bg-green-50 text-green-600 hover:bg-green-100"}`}>
                    <Power className="w-3 h-3" /> {addon.is_active ? "Disable" : "Enable"}
                  </button>
                  <button className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-slate-100 text-slate-600 hover:bg-slate-200">
                    <Settings className="w-3 h-3" /> Configure
                  </button>
                </>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
