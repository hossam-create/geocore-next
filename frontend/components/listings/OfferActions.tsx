'use client';

import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { formatPrice } from '@/lib/utils';
import { useAuthStore } from '@/store/auth';
import { showConversionToast } from '@/components/ui/ConversionToast';
import { Gavel, ArrowRightLeft, Check, X, Clock, Send, RotateCcw } from 'lucide-react';

type OfferStatus = 'pending' | 'accepted' | 'countered' | 'rejected' | 'expired';

interface Offer {
  id: string;
  amount: number;
  currency: string;
  status: OfferStatus;
  counter_amount?: number;
  counter_currency?: string;
  created_at: string;
  updated_at?: string;
  from_user_id?: string;
  to_user_id?: string;
}

interface OfferActionsProps {
  listingId: string;
  listingPrice: number;
  currency?: string;
  isAuction?: boolean;
  isSeller?: boolean;
  onOfferSent?: () => void;
  onOfferAccepted?: () => void;
}

export function OfferActions({ listingId, listingPrice, currency = 'AED', isAuction, isSeller, onOfferSent, onOfferAccepted }: OfferActionsProps) {
  const { isAuthenticated } = useAuthStore();
  const qc = useQueryClient();
  const [offerAmount, setOfferAmount] = useState('');
  const [counterAmount, setCounterAmount] = useState('');
  const [message, setMessage] = useState('');
  const [showOffer, setShowOffer] = useState(false);
  const [showCounter, setShowCounter] = useState<string | false>(false);

  // Fetch existing offers for this listing
  const { data: offersData } = useQuery({
    queryKey: ['listing-offers', listingId],
    queryFn: () => api.get(`/listings/${listingId}/offers`).then(r => (r.data?.data ?? r.data) as Offer[]),
    enabled: isAuthenticated && !isAuction,
    retry: false,
    staleTime: 30_000,
  });

  const offers: Offer[] = offersData ?? [];
  const pendingOffers = offers.filter(o => o.status === 'pending');
  const counteredOffers = offers.filter(o => o.status === 'countered');
  const activeOffers = [...pendingOffers, ...counteredOffers];

  const offerMutation = useMutation({
    mutationFn: (amount: number) => api.post(`/listings/${listingId}/offers`, { amount }),
    onSuccess: () => {
      setMessage('تم إرسال عرضك');
      setOfferAmount('');
      setShowOffer(false);
      showConversionToast('offer', 'تم إرسال عرضك بنجاح');
      qc.invalidateQueries({ queryKey: ['listing', listingId] });
      qc.invalidateQueries({ queryKey: ['listing-offers', listingId] });
      onOfferSent?.();
    },
    onError: (err: any) => {
      setMessage(err?.response?.data?.message || 'Failed to send offer.');
    },
  });

  const acceptMutation = useMutation({
    mutationFn: (offerId: string) => api.patch(`/listings/${listingId}/offers/${offerId}/accept`, {}),
    onSuccess: () => {
      setMessage('تم قبول العرض');
      showConversionToast('offer', 'تم قبول العرض بنجاح');
      qc.invalidateQueries({ queryKey: ['listing', listingId] });
      qc.invalidateQueries({ queryKey: ['listing-offers', listingId] });
      onOfferAccepted?.();
    },
    onError: () => setMessage('فشل قبول العرض'),
  });

  const counterMutation = useMutation({
    mutationFn: ({ offerId, amount }: { offerId: string; amount: number }) =>
      api.patch(`/listings/${listingId}/offers/${offerId}/counter`, { amount }),
    onSuccess: () => {
      setMessage('تم إرسال العرض المضاد');
      setCounterAmount('');
      setShowCounter(false);
      showConversionToast('offer', 'تم إرسال العرض المضاد');
      qc.invalidateQueries({ queryKey: ['listing-offers', listingId] });
    },
    onError: () => setMessage('فشل إرسال العرض المضاد'),
  });

  const rejectMutation = useMutation({
    mutationFn: (offerId: string) => api.patch(`/listings/${listingId}/offers/${offerId}/reject`, {}),
    onSuccess: () => {
      setMessage('تم رفض العرض');
      showConversionToast('notification', 'تم رفض العرض');
      qc.invalidateQueries({ queryKey: ['listing-offers', listingId] });
    },
    onError: () => setMessage('فشل رفض العرض'),
  });

  if (isAuction) return null;

  return (
    <div className="mt-3 space-y-3">
      {/* ── Seller view: incoming offers ── */}
      {isSeller && activeOffers.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide">Incoming Offers</p>
          {activeOffers.map(offer => (
            <div key={offer.id} className="rounded-xl border border-gray-200 bg-white p-3 space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-sm font-bold text-gray-900">
                  {formatPrice(offer.amount, offer.currency || currency)}
                </span>
                <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${
                  offer.status === 'countered' ? 'bg-amber-50 text-amber-700' : 'bg-blue-50 text-blue-700'
                }`}>
                  {offer.status === 'countered' ? 'Countered' : 'Pending'}
                </span>
              </div>

              {/* Accept / Counter / Reject actions */}
              <div className="flex gap-2">
                <button
                  onClick={() => acceptMutation.mutate(offer.id)}
                  disabled={acceptMutation.isPending}
                  className="flex-1 bg-green-600 hover:bg-green-700 text-white font-bold py-2 rounded-lg text-xs flex items-center justify-center gap-1 disabled:opacity-60"
                >
                  <Check size={13} /> Accept
                </button>
                <button
                  onClick={() => setShowCounter(showCounter === offer.id ? false : offer.id)}
                  className="flex-1 border-2 border-[#0071CE] text-[#0071CE] font-bold py-2 rounded-lg text-xs flex items-center justify-center gap-1 hover:bg-blue-50"
                >
                  <ArrowRightLeft size={13} /> Counter
                </button>
                <button
                  onClick={() => rejectMutation.mutate(offer.id)}
                  disabled={rejectMutation.isPending}
                  className="border-2 border-red-200 text-red-500 font-bold py-2 px-3 rounded-lg text-xs flex items-center justify-center gap-1 hover:bg-red-50 disabled:opacity-60"
                >
                  <X size={13} /> Reject
                </button>
              </div>

              {/* Counter offer input */}
              {showCounter === offer.id && (
                <div className="flex gap-2 mt-1">
                  <input
                    type="number"
                    value={counterAmount}
                    onChange={(e) => { setCounterAmount(e.target.value); setMessage(''); }}
                    placeholder={`Counter (${offer.currency || currency})`}
                    className="flex-1 border border-gray-200 rounded-lg px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
                  />
                  <button
                    onClick={() => {
                      const amount = Number(counterAmount);
                      if (!amount || amount <= 0) { setMessage('Enter a valid amount.'); return; }
                      counterMutation.mutate({ offerId: offer.id, amount });
                    }}
                    disabled={counterMutation.isPending}
                    className="bg-[#0071CE] hover:bg-[#005BA1] text-white font-bold px-4 py-2 rounded-lg text-sm disabled:opacity-60 flex items-center gap-1"
                  >
                    <Send size={13} /> {counterMutation.isPending ? 'Sending...' : 'Send'}
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* ── Buyer view: my offer status ── */}
      {!isSeller && offers.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide">Your Offers</p>
          {offers.map(offer => (
            <div key={offer.id} className="rounded-xl border border-gray-200 bg-white p-3">
              <div className="flex items-center justify-between">
                <span className="text-sm font-bold text-gray-900">
                  {formatPrice(offer.amount, offer.currency || currency)}
                </span>
                <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${
                  offer.status === 'accepted' ? 'bg-green-50 text-green-700' :
                  offer.status === 'countered' ? 'bg-amber-50 text-amber-700' :
                  offer.status === 'rejected' ? 'bg-red-50 text-red-600' :
                  offer.status === 'expired' ? 'bg-gray-100 text-gray-500' :
                  'bg-blue-50 text-blue-700'
                }`}>
                  {offer.status === 'pending' && <Clock size={9} className="inline mr-0.5" />}
                  {offer.status === 'countered' && <ArrowRightLeft size={9} className="inline mr-0.5" />}
                  {offer.status === 'accepted' && <Check size={9} className="inline mr-0.5" />}
                  {offer.status === 'rejected' && <X size={9} className="inline mr-0.5" />}
                  {offer.status === 'expired' && <Clock size={9} className="inline mr-0.5" />}
                  {offer.status === 'pending' ? 'Waiting for response' :
                   offer.status === 'countered' ? `Countered: ${formatPrice(offer.counter_amount ?? 0, offer.counter_currency || currency)}` :
                   offer.status === 'accepted' ? 'Accepted!' :
                   offer.status === 'rejected' ? 'Rejected' :
                   'Expired'}
                </span>
              </div>
              {offer.status === 'countered' && offer.counter_amount && (
                <div className="mt-2 flex gap-2">
                  <button
                    onClick={() => acceptMutation.mutate(offer.id)}
                    disabled={acceptMutation.isPending}
                    className="flex-1 bg-green-600 hover:bg-green-700 text-white font-bold py-2 rounded-lg text-xs flex items-center justify-center gap-1 disabled:opacity-60"
                  >
                    <Check size={13} /> Accept Counter
                  </button>
                  <button
                    onClick={() => rejectMutation.mutate(offer.id)}
                    className="border-2 border-red-200 text-red-500 font-bold py-2 px-3 rounded-lg text-xs hover:bg-red-50"
                  >
                    <X size={13} />
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* ── Make Offer button ── */}
      {!showOffer && !isSeller && (
        <button
          onClick={() => {
            if (!isAuthenticated) return;
            setShowOffer(true);
          }}
          className="w-full border-2 border-[#0071CE] text-[#0071CE] font-bold py-3 rounded-xl hover:bg-blue-50 transition-colors text-sm flex items-center justify-center gap-2"
        >
          <Gavel size={16} /> Make Offer
        </button>
      )}

      {showOffer && (
        <div className="space-y-2">
          <div className="flex gap-2">
            <input
              type="number"
              value={offerAmount}
              onChange={(e) => { setOfferAmount(e.target.value); setMessage(''); }}
              placeholder={`Your offer (${currency})`}
              className="flex-1 border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
            />
            <button
              onClick={() => {
                const amount = Number(offerAmount);
                if (!amount || amount <= 0) { setMessage('Enter a valid amount.'); return; }
                offerMutation.mutate(amount);
              }}
              disabled={offerMutation.isPending}
              className="bg-[#0071CE] hover:bg-[#005BA1] text-white font-bold px-5 py-3 rounded-xl transition-colors disabled:opacity-60"
            >
              {offerMutation.isPending ? 'Sending...' : 'Send'}
            </button>
          </div>
          <button onClick={() => setShowOffer(false)} className="text-xs text-gray-400 hover:text-gray-600">Cancel</button>
        </div>
      )}

      {message && (
        <p className={`text-sm ${message.includes('تم') || message.includes('success') ? 'text-green-600' : 'text-red-500'}`}>
          {message}
        </p>
      )}

      {/* Inline hint */}
      {listingPrice > 0 && !showOffer && !isSeller && (
        <p className="text-xs text-gray-400">
          Listed at {formatPrice(listingPrice, currency)} — try offering less
        </p>
      )}
    </div>
  );
}
