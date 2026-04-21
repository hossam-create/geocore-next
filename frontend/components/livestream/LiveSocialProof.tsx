'use client';

import { useEffect, useState } from 'react';
import { Users, TrendingUp, Eye } from 'lucide-react';

interface RecentBidder {
  user_id: string;
  display_name: string;
  amount_cents: number;
  bid_at: string;
}

interface LiveSocialProofProps {
  viewerCount: number;
  bidCount: number;
  recentBidders: RecentBidder[];
}

function formatCents(cents: number): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(cents / 100);
}

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const secs = Math.floor(diff / 1000);
  if (secs < 5) return 'just now';
  if (secs < 60) return `${secs}s ago`;
  const mins = Math.floor(secs / 60);
  return `${mins}m ago`;
}

export default function LiveSocialProof({ viewerCount, bidCount, recentBidders }: LiveSocialProofProps) {
  const [showBidders, setShowBidders] = useState(false);

  // Pulse animation when bid count changes
  const [pulse, setPulse] = useState(false);
  useEffect(() => {
    setPulse(true);
    const t = setTimeout(() => setPulse(false), 600);
    return () => clearTimeout(t);
  }, [bidCount]);

  if (bidCount === 0 && viewerCount === 0) return null;

  return (
    <div className="space-y-2">
      {/* Stats row */}
      <div className="flex items-center gap-3 text-xs">
        {viewerCount > 0 && (
          <span className="flex items-center gap-1 text-gray-500">
            <Eye className="w-3.5 h-3.5" />
            {viewerCount} watching
          </span>
        )}
        {bidCount > 0 && (
          <span
            className={`flex items-center gap-1 font-semibold text-blue-600 transition-transform ${
              pulse ? 'scale-110' : 'scale-100'
            }`}
          >
            <TrendingUp className="w-3.5 h-3.5" />
            {bidCount} bid{bidCount !== 1 && 's'}
          </span>
        )}
      </div>

      {/* Recent bidders */}
      {recentBidders.length > 0 && (
        <div>
          <button
            onClick={() => setShowBidders(!showBidders)}
            className="flex items-center gap-1.5 text-xs text-gray-500 hover:text-gray-700 transition-colors"
          >
            <Users className="w-3.5 h-3.5" />
            {recentBidders.length} recent bidder{recentBidders.length !== 1 && 's'}
            <span className="text-gray-400">{showBidders ? '▲' : '▼'}</span>
          </button>

          {showBidders && (
            <div className="mt-1.5 space-y-1 animate-fade-in">
              {recentBidders.slice(0, 5).map((b) => (
                <div key={b.user_id} className="flex items-center justify-between text-xs bg-gray-50 rounded-lg px-2.5 py-1.5">
                  <span className="text-gray-700 font-medium truncate max-w-[120px]">
                    {b.display_name}
                  </span>
                  <span className="text-gray-500 ml-2">
                    {formatCents(b.amount_cents)} · {timeAgo(b.bid_at)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Social proof nudge — "X people are bidding" */}
      {bidCount >= 3 && (
        <p className="text-xs text-amber-600 font-medium animate-pulse">
          🔥 {bidCount} people are bidding — don&apos;t miss out!
        </p>
      )}

      <style jsx>{`
        @keyframes fade-in {
          from { opacity: 0; transform: translateY(-4px); }
          to { opacity: 1; transform: translateY(0); }
        }
        .animate-fade-in { animation: fade-in 0.2s ease-out; }
      `}</style>
    </div>
  );
}
