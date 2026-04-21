'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Gavel, TrendingUp, Clock, AlertCircle } from 'lucide-react';
import { formatPrice } from '@/lib/utils';
import api from '@/lib/api';

interface LiveStreamBidPanelProps {
  auctionId?: string;
  currentBid: number;
  minIncrement: number;
  currency: string;
  endsAt?: string;
  isAuthenticated: boolean;
}

export default function LiveStreamBidPanel({
  auctionId,
  currentBid,
  minIncrement,
  currency,
  endsAt,
  isAuthenticated,
}: LiveStreamBidPanelProps) {
  const router = useRouter();
  const [bidAmount, setBidAmount] = useState<number>(currentBid + minIncrement);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const handleBid = async () => {
    if (!isAuthenticated) {
      router.push('/login?next=' + window.location.pathname);
      return;
    }
    if (!auctionId) return;
    if (bidAmount <= currentBid) {
      setError(`Bid must be higher than ${formatPrice(currentBid, currency)}`);
      return;
    }

    setSubmitting(true);
    setError('');
    setSuccess('');

    try {
      await api.post(`/auctions/${auctionId}/bids`, { amount: bidAmount });
      setSuccess(`Bid of ${formatPrice(bidAmount, currency)} placed!`);
      setBidAmount(bidAmount + minIncrement);
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message;
      setError(msg ?? 'Failed to place bid. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  if (!auctionId) {
    return (
      <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-5 text-center">
        <Gavel className="w-8 h-8 text-gray-300 mx-auto mb-2" />
        <p className="text-sm text-gray-500">No auction linked to this stream</p>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden">
      <div className="bg-gradient-to-r from-[#0071CE] to-[#003f75] px-4 py-3">
        <p className="text-white font-bold text-sm flex items-center gap-2">
          <Gavel className="w-4 h-4" /> Live Auction
        </p>
        {endsAt && (
          <p className="text-blue-200 text-xs flex items-center gap-1 mt-0.5">
            <Clock className="w-3 h-3" />
            Ends {new Date(endsAt).toLocaleTimeString()}
          </p>
        )}
      </div>

      <div className="p-4 space-y-4">
        <div className="bg-gray-50 rounded-xl p-3 text-center">
          <p className="text-xs text-gray-500 uppercase tracking-wide mb-1">Current Bid</p>
          <p className="text-3xl font-extrabold text-gray-900">{formatPrice(currentBid, currency)}</p>
          <p className="text-xs text-gray-400 mt-1 flex items-center justify-center gap-1">
            <TrendingUp className="w-3 h-3" /> Min increment: {formatPrice(minIncrement, currency)}
          </p>
        </div>

        <div className="flex gap-2">
          <input
            type="number"
            value={bidAmount}
            onChange={(e) => setBidAmount(parseFloat(e.target.value) || 0)}
            min={currentBid + minIncrement}
            step={minIncrement}
            className="flex-1 border border-gray-200 rounded-xl px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-[#0071CE]"
          />
          <button
            onClick={handleBid}
            disabled={submitting}
            className="bg-[#0071CE] hover:bg-[#005BA1] text-white font-bold px-5 py-2.5 rounded-xl transition-colors text-sm disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
          >
            <Gavel className="w-4 h-4" />
            {submitting ? 'Placing…' : 'Bid'}
          </button>
        </div>

        <div className="grid grid-cols-3 gap-2">
          {[currentBid + minIncrement, currentBid + minIncrement * 3, currentBid + minIncrement * 5].map((quick) => (
            <button
              key={quick}
              onClick={() => setBidAmount(quick)}
              className="text-xs border border-gray-200 rounded-lg py-1.5 hover:bg-gray-50 transition-colors text-gray-700 font-medium"
            >
              {formatPrice(quick, currency)}
            </button>
          ))}
        </div>

        {error && (
          <div className="flex items-start gap-2 bg-red-50 border border-red-200 rounded-xl p-3">
            <AlertCircle className="w-4 h-4 text-red-500 flex-shrink-0 mt-0.5" />
            <p className="text-xs text-red-700">{error}</p>
          </div>
        )}
        {success && (
          <div className="bg-green-50 border border-green-200 rounded-xl p-3">
            <p className="text-xs text-green-700 font-semibold">{success}</p>
          </div>
        )}
      </div>
    </div>
  );
}
