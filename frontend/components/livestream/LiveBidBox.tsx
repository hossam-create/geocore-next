'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { Gavel, TrendingUp, Zap, AlertCircle, ShoppingCart, Volume2, VolumeX } from 'lucide-react';
import api from '@/lib/api';
import LiveCountdown from './LiveCountdown';
import LiveStatusBadge from './LiveStatusBadge';
import LiveSocialProof from './LiveSocialProof';
import LiveBidNotification, { type BidNotification } from './LiveBidNotification';
import { useLiveSounds, type SoundType } from './useLiveSounds';

interface LiveItem {
  id: string;
  session_id: string;
  title: string;
  image_url?: string;
  start_price_cents: number;
  buy_now_price_cents?: number | null;
  current_bid_cents: number;
  min_increment_cents: number;
  highest_bidder_id?: string | null;
  bid_count: number;
  status: string;
  ends_at?: string | null;
  extension_count?: number;
}

interface RecentBidder {
  user_id: string;
  display_name: string;
  amount_cents: number;
  bid_at: string;
}

interface LiveBidBoxProps {
  sessionId: string;
  isAuthenticated: boolean;
  currentUserId?: string;
}

function formatCents(cents: number, currency = 'USD'): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency }).format(cents / 100);
}

export default function LiveBidBox({ sessionId, isAuthenticated, currentUserId }: LiveBidBoxProps) {
  const [items, setItems] = useState<LiveItem[]>([]);
  const [bidAmount, setBidAmount] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const wsRef = useRef<WebSocket | null>(null);

  // Real-time UX state
  const [bidPulse, setBidPulse] = useState(false);
  const [outbidAlert, setOutbidAlert] = useState(false);
  const [viewerCount, setViewerCount] = useState(0);
  const [recentBidders, setRecentBidders] = useState<RecentBidder[]>([]);
  const [notifications, setNotifications] = useState<BidNotification[]>([]);
  const [soundEnabled, setSoundEnabled] = useState(true);
  const { play } = useLiveSounds(soundEnabled);

  const notifIdRef = useRef(0);
  const addNotification = useCallback((type: BidNotification['type'], message: string, amountCents?: number) => {
    const id = `n-${++notifIdRef.current}`;
    setNotifications((prev) => [...prev.slice(-4), { id, type, message, amountCents, timestamp: Date.now() }]);
  }, []);
  const dismissNotification = useCallback((id: string) => {
    setNotifications((prev) => prev.filter((n) => n.id !== id));
  }, []);

  // Fetch items
  const fetchItems = useCallback(async () => {
    try {
      const res = await api.get(`/livestream/${sessionId}/items`);
      const data: LiveItem[] = res.data?.data ?? res.data ?? [];
      setItems(data);
    } catch { /* ignore */ }
  }, [sessionId]);

  useEffect(() => {
    fetchItems();
  }, [fetchItems]);

  // WebSocket for real-time bid updates
  useEffect(() => {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const wsUrl = `${proto}//${host}/ws/auctions/${sessionId}`;

    let ws: WebSocket;
    let reconnectTimer: ReturnType<typeof setTimeout>;
    let reconnectDelay = 1000; // start at 1s, exponential backoff
    const maxReconnectDelay = 30000;

    const connect = () => {
      ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onmessage = (e) => {
        try {
          const msg = JSON.parse(e.data);
          const eventType = msg.event;

          // Handle structured LiveEvent types
          switch (eventType) {
            case 'new_bid':
              // Update item state
              if (msg.item_id) {
                setItems((prev) =>
                  prev.map((item) =>
                    item.id === msg.item_id
                      ? {
                          ...item,
                          current_bid_cents: msg.current_bid_cents ?? item.current_bid_cents,
                          highest_bidder_id: msg.highest_bidder_id ?? item.highest_bidder_id,
                          bid_count: msg.bid_count ?? item.bid_count,
                          status: msg.status ?? item.status,
                          ends_at: msg.new_ends_at ?? msg.ends_at ?? item.ends_at,
                        }
                      : item
                  )
                );

                // Animated pulse on bid change
                setBidPulse(true);
                setTimeout(() => setBidPulse(false), 600);

                // Sound
                play('bid');

                // Notification
                const isOwnBid = msg.highest_bidder_id === currentUserId;
                if (!isOwnBid) {
                  addNotification('new_bid', 'New bid placed!', msg.current_bid_cents);
                }
              }

              // Anti-snipe extension
              if (msg.extended) {
                addNotification('extended', 'Time extended — anti-snipe activated!');
                play('extended');
              }
              break;

            case 'outbid':
              // Check if current user was outbid
              if (msg.outbid_user_id === currentUserId) {
                setOutbidAlert(true);
                setTimeout(() => setOutbidAlert(false), 5000);
                addNotification('outbid', 'You were outbid! Bid again to stay in!', msg.current_bid_cents);
                play('outbid');
              }
              // Update item
              if (msg.item_id) {
                setItems((prev) =>
                  prev.map((item) =>
                    item.id === msg.item_id
                      ? {
                          ...item,
                          current_bid_cents: msg.current_bid_cents ?? item.current_bid_cents,
                          highest_bidder_id: msg.highest_bidder_id ?? item.highest_bidder_id,
                          bid_count: msg.bid_count ?? item.bid_count,
                        }
                      : item
                  )
                );
              }
              break;

            case 'auction_end':
            case 'item_sold':
            case 'item_sold_buy_now':
              if (msg.item_id) {
                setItems((prev) =>
                  prev.map((item) =>
                    item.id === msg.item_id
                      ? { ...item, status: msg.status ?? 'sold', current_bid_cents: msg.current_bid_cents ?? item.current_bid_cents }
                      : item
                  )
                );
              }
              if (msg.highest_bidder_id === currentUserId) {
                addNotification('won', '🎉 You won the auction!', msg.current_bid_cents);
                play('won');
              } else {
                addNotification('sold', 'Item sold!', msg.current_bid_cents);
                play('sold');
              }
              break;

            case 'item_unsold':
              if (msg.item_id) {
                setItems((prev) =>
                  prev.map((item) =>
                    item.id === msg.item_id ? { ...item, status: 'unsold' } : item
                  )
                );
              }
              break;

            case 'item_settling':
            case 'item_payment_failed':
            case 'item_activated':
              if (msg.item_id) {
                setItems((prev) =>
                  prev.map((item) =>
                    item.id === msg.item_id
                      ? { ...item, status: msg.status ?? item.status, ends_at: msg.ends_at ?? item.ends_at }
                      : item
                  )
                );
              }
              if (eventType === 'item_activated') {
                addNotification('new_bid', 'New item is live!', undefined);
              }
              break;

            case 'viewer_join':
            case 'viewer_leave':
              if (msg.viewer_count != null) {
                setViewerCount(msg.viewer_count);
              }
              break;

            case 'social_proof':
              if (msg.recent_bidders) {
                setRecentBidders(msg.recent_bidders);
              }
              if (msg.viewer_count != null) {
                setViewerCount(msg.viewer_count);
              }
              break;

            default:
              // Legacy format — backward compatible with old msg.item_id + msg.event
              if (msg.item_id && msg.event) {
                setItems((prev) =>
                  prev.map((item) =>
                    item.id === msg.item_id
                      ? {
                          ...item,
                          current_bid_cents: msg.current_bid_cents ?? item.current_bid_cents,
                          highest_bidder_id: msg.highest_bidder_id ?? item.highest_bidder_id,
                          bid_count: msg.bid_count ?? item.bid_count,
                          status: msg.status ?? item.status,
                          ends_at: msg.ends_at ?? item.ends_at,
                        }
                      : item
                  )
                );
              }
          }
        } catch { /* ignore bad message */ }
      };

      ws.onclose = () => {
        reconnectTimer = setTimeout(connect, reconnectDelay);
        reconnectDelay = Math.min(reconnectDelay * 2, maxReconnectDelay);
      };

      ws.onopen = () => {
        reconnectDelay = 1000; // reset on successful connect
      };
    };

    connect();

    return () => {
      clearTimeout(reconnectTimer);
      ws?.close();
    };
  }, [sessionId, currentUserId, play, addNotification]);

  // Polling fallback every 5s
  useEffect(() => {
    const iv = setInterval(fetchItems, 5000);
    return () => clearInterval(iv);
  }, [fetchItems]);

  const activeItem = items.find((i) => i.status === 'active');
  const pendingItems = items.filter((i) => i.status === 'pending');
  const endedItems = items.filter((i) => i.status === 'sold' || i.status === 'unsold' || i.status === 'payment_failed');

  const handleBid = async (item: LiveItem) => {
    if (!isAuthenticated) {
      window.location.href = '/login?next=' + window.location.pathname;
      return;
    }
    setSubmitting(true);
    setError('');
    setSuccess('');
    setOutbidAlert(false);
    try {
      const idemKey = `${currentUserId}-${item.id}-${Date.now()}`;
      await api.post(`/livestream/${sessionId}/items/${item.id}/bid`, {
        amount_cents: bidAmount,
      }, {
        headers: { 'X-Idempotency-Key': idemKey },
      });
      setSuccess(`Bid of ${formatCents(bidAmount)} placed!`);
      setBidAmount(bidAmount + item.min_increment_cents);
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message;
      setError(msg ?? 'Bid failed. Try again.');
      play('error');
    } finally {
      setSubmitting(false);
    }
  };

  const handleBuyNow = async (item: LiveItem) => {
    if (!isAuthenticated) {
      window.location.href = '/login?next=' + window.location.pathname;
      return;
    }
    setSubmitting(true);
    setError('');
    try {
      await api.post(`/livestream/${sessionId}/items/${item.id}/buy-now`);
      setSuccess('Purchase successful!');
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message;
      setError(msg ?? 'Buy Now failed.');
      play('error');
    } finally {
      setSubmitting(false);
    }
  };

  // Auto-set bid amount when active item changes
  useEffect(() => {
    if (activeItem) {
      const min = activeItem.bid_count === 0
        ? activeItem.start_price_cents
        : activeItem.current_bid_cents + activeItem.min_increment_cents;
      setBidAmount(min);
    }
  }, [activeItem?.id, activeItem?.current_bid_cents]);

  // Leave session on unmount
  useEffect(() => {
    return () => {
      if (isAuthenticated && sessionId) {
        api.post(`/livestream/${sessionId}/leave`).catch(() => {});
      }
    };
  }, [sessionId, isAuthenticated]);

  if (items.length === 0) {
    return (
      <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-5 text-center">
        <Gavel className="w-8 h-8 text-gray-300 mx-auto mb-2" />
        <p className="text-sm text-gray-500">No auction items yet</p>
        <p className="text-xs text-gray-400 mt-1">The host will add items during the stream</p>
      </div>
    );
  }

  return (
    <>
      {/* Floating notifications */}
      <LiveBidNotification
        notifications={notifications}
        onDismiss={dismissNotification}
        soundEnabled={soundEnabled}
        onToggleSound={() => setSoundEnabled(!soundEnabled)}
      />

      <div className="space-y-3">
        {/* Active Item */}
        {activeItem && (
          <div className="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden">
            <div className="bg-gradient-to-r from-red-500 to-orange-500 px-4 py-3 flex items-center justify-between">
              <p className="text-white font-bold text-sm flex items-center gap-2">
                <Gavel className="w-4 h-4" /> Bidding Now
              </p>
              <div className="flex items-center gap-2">
                <LiveCountdown endsAt={activeItem.ends_at} onExpired={fetchItems} extended={(activeItem.extension_count ?? 0) > 0} />
                {/* Sound toggle inline */}
                <button
                  onClick={() => setSoundEnabled(!soundEnabled)}
                  className="text-white/70 hover:text-white transition-colors"
                  title={soundEnabled ? 'Mute' : 'Unmute'}
                >
                  {soundEnabled ? <Volume2 className="w-3.5 h-3.5" /> : <VolumeX className="w-3.5 h-3.5" />}
                </button>
              </div>
            </div>

            {activeItem.image_url && (
              <img src={activeItem.image_url} alt={activeItem.title} className="w-full h-32 object-cover" />
            )}

            <div className="p-4 space-y-3">
              <h3 className="font-semibold text-gray-900 text-sm">{activeItem.title}</h3>

              {/* Animated current bid */}
              <div className={`bg-gray-50 rounded-xl p-3 text-center transition-all duration-300 ${
                bidPulse ? 'scale-[1.02] ring-2 ring-blue-300 bg-blue-50' : ''
              }`}>
                <p className="text-xs text-gray-500 uppercase tracking-wide mb-1">Current Bid</p>
                <p className={`text-2xl font-extrabold transition-colors duration-300 ${
                  bidPulse ? 'text-blue-600' : 'text-gray-900'
                }`}>
                  {formatCents(activeItem.current_bid_cents || activeItem.start_price_cents)}
                </p>
                <p className="text-xs text-gray-400 mt-1 flex items-center justify-center gap-1">
                  <TrendingUp className="w-3 h-3" /> {activeItem.bid_count} bid{activeItem.bid_count !== 1 && 's'}
                  {' · '}Min +{formatCents(activeItem.min_increment_cents)}
                </p>

                {/* Highest bidder indicator */}
                {activeItem.highest_bidder_id === currentUserId ? (
                  <p className="text-xs text-green-600 font-bold mt-1 animate-pulse">🏆 You are the highest bidder!</p>
                ) : activeItem.highest_bidder_id ? (
                  <p className="text-xs text-gray-400 mt-1">Someone else is leading</p>
                ) : null}
              </div>

              {/* Outbid alert */}
              {outbidAlert && (
                <div className="bg-amber-50 border border-amber-300 rounded-xl p-3 flex items-center gap-2 animate-pulse">
                  <Zap className="w-4 h-4 text-amber-600 flex-shrink-0" />
                  <div>
                    <p className="text-xs text-amber-800 font-bold">You were outbid!</p>
                    <p className="text-xs text-amber-600">Place a higher bid to stay in the auction</p>
                  </div>
                </div>
              )}

              {/* Social proof */}
              <LiveSocialProof
                viewerCount={viewerCount}
                bidCount={activeItem.bid_count}
                recentBidders={recentBidders}
              />

              {/* Bid input */}
              <div className="flex gap-2">
                <input
                  type="number"
                  value={bidAmount / 100}
                  onChange={(e) => setBidAmount(Math.round(parseFloat(e.target.value || '0') * 100))}
                  step={activeItem.min_increment_cents / 100}
                  className={`flex-1 border rounded-xl px-3 py-2.5 text-sm focus:outline-none focus:ring-2 transition-colors ${
                    outbidAlert ? 'border-amber-300 focus:ring-amber-400' : 'border-gray-200 focus:ring-red-400'
                  }`}
                />
                <button
                  onClick={() => handleBid(activeItem)}
                  disabled={submitting}
                  className={`text-white font-bold px-5 py-2.5 rounded-xl transition-all text-sm disabled:opacity-50 flex items-center gap-2 ${
                    outbidAlert
                      ? 'bg-amber-500 hover:bg-amber-600 animate-pulse'
                      : 'bg-red-500 hover:bg-red-600'
                  }`}
                >
                  <Gavel className="w-4 h-4" />
                  {submitting ? '…' : outbidAlert ? 'Bid Again' : 'Bid'}
                </button>
              </div>

              {/* Quick bid buttons */}
              <div className="grid grid-cols-3 gap-2">
                {[1, 3, 5].map((mult) => {
                  const base = activeItem.bid_count === 0 ? activeItem.start_price_cents : activeItem.current_bid_cents + activeItem.min_increment_cents;
                  const q = base + activeItem.min_increment_cents * (mult - 1);
                  return (
                    <button
                      key={mult}
                      onClick={() => setBidAmount(q)}
                      className="text-xs border border-gray-200 rounded-lg py-1.5 hover:bg-gray-50 transition-colors text-gray-700 font-medium"
                    >
                      {formatCents(q)}
                    </button>
                  );
                })}
              </div>

              {/* Buy Now */}
              {activeItem.buy_now_price_cents && (
                <button
                  onClick={() => handleBuyNow(activeItem)}
                  disabled={submitting}
                  className="w-full bg-[#0071CE] hover:bg-[#005BA1] text-white font-bold py-2.5 rounded-xl transition-colors text-sm flex items-center justify-center gap-2 disabled:opacity-50"
                >
                  <ShoppingCart className="w-4 h-4" />
                  Buy Now {formatCents(activeItem.buy_now_price_cents)}
                </button>
              )}

              {error && (
                <div className="flex items-start gap-2 bg-red-50 border border-red-200 rounded-xl p-2.5">
                  <AlertCircle className="w-4 h-4 text-red-500 flex-shrink-0 mt-0.5" />
                  <p className="text-xs text-red-700">{error}</p>
                </div>
              )}
              {success && (
                <div className="bg-green-50 border border-green-200 rounded-xl p-2.5">
                  <p className="text-xs text-green-700 font-semibold">{success}</p>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Upcoming items */}
        {pendingItems.length > 0 && (
          <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-4">
            <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">Up Next</p>
            <div className="space-y-2">
              {pendingItems.map((item) => (
                <div key={item.id} className="flex items-center gap-3 p-2 bg-gray-50 rounded-xl">
                  {item.image_url && <img src={item.image_url} className="w-10 h-10 rounded-lg object-cover" alt="" />}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-gray-800 truncate">{item.title}</p>
                    <p className="text-xs text-gray-500">Starting at {formatCents(item.start_price_cents)}</p>
                  </div>
                  <LiveStatusBadge status={item.status} />
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Ended items */}
        {endedItems.length > 0 && (
          <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-4">
            <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">Completed</p>
            <div className="space-y-2">
              {endedItems.map((item) => (
                <div key={item.id} className="flex items-center gap-3 p-2 bg-gray-50 rounded-xl">
                  {item.image_url && <img src={item.image_url} className="w-10 h-10 rounded-lg object-cover" alt="" />}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-gray-800 truncate">{item.title}</p>
                    <p className="text-xs text-gray-500">{formatCents(item.current_bid_cents)} · {item.bid_count} bids</p>
                  </div>
                  <LiveStatusBadge status={item.status} />
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </>
  );
}
