'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import dynamic from 'next/dynamic';
import api from '@/lib/api';
import { useAuthStore } from '@/store/auth';
import {
  Radio, Video, StopCircle, CheckCircle, AlertCircle,
  ChevronLeft, Link2, Copy, Users
} from 'lucide-react';

const LiveStreamPlayer = dynamic(
  () => import('@/components/livestream/LiveStreamPlayer'),
  { ssr: false }
);

interface LiveSession {
  id: string;
  title: string;
  status: string;
  room_name: string;
}

interface StartResponse {
  session: LiveSession;
  token: string;
  livekit_url: string;
  simulated: boolean;
}

interface AuctionOption {
  id: string;
  title: string;
  status: string;
}

export default function GoLivePage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();

  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [auctionId, setAuctionId] = useState('');
  const [creating, setCreating] = useState(false);
  const [session, setSession] = useState<LiveSession | null>(null);
  const [streamData, setStreamData] = useState<StartResponse | null>(null);
  const [error, setError] = useState('');
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!isAuthenticated) router.push('/login?next=/sell/live');
  }, [isAuthenticated, router]);

  const { data: myAuctions = [] } = useQuery<AuctionOption[]>({
    queryKey: ['my-auctions-for-stream'],
    queryFn: async () => {
      const res = await api.get('/auctions?mine=true&status=active&limit=20');
      return res.data?.data ?? [];
    },
    enabled: isAuthenticated,
  });

  const handleCreate = async () => {
    if (!title.trim()) {
      setError('Title is required');
      return;
    }
    setCreating(true);
    setError('');
    try {
      const res = await api.post('/livestream', {
        title: title.trim(),
        description: description.trim(),
        auction_id: auctionId || undefined,
      });
      setSession(res.data?.data ?? res.data);
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message;
      setError(msg ?? 'Failed to create session');
    } finally {
      setCreating(false);
    }
  };

  const handleStart = async () => {
    if (!session) return;
    setError('');
    try {
      const res = await api.post(`/livestream/${session.id}/start`);
      const data: StartResponse = res.data?.data ?? res.data;
      setStreamData(data);
      setSession(data.session);
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message;
      setError(msg ?? 'Failed to start stream');
    }
  };

  const handleEnd = async () => {
    if (!session) return;
    setError('');
    try {
      await api.post(`/livestream/${session.id}/end`);
      setStreamData(null);
      setSession((s) => s ? { ...s, status: 'ended' } : s);
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message;
      setError(msg ?? 'Failed to end stream');
    }
  };

  const viewerLink = session
    ? `${typeof window !== 'undefined' ? window.location.origin : ''}/auctions/live/${session.id}`
    : '';

  const copyLink = () => {
    navigator.clipboard.writeText(viewerLink).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  };

  // ── Step 1: Create session ────────────────────────────────────────────────
  if (!session) {
    return (
      <div className="max-w-xl mx-auto px-4 py-10">
        <button
          onClick={() => router.back()}
          className="flex items-center gap-1.5 text-gray-500 hover:text-[#0071CE] text-sm mb-6 transition-colors"
        >
          <ChevronLeft className="w-4 h-4" /> Back
        </button>

        <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden">
          <div className="bg-gradient-to-r from-red-500 to-orange-500 px-6 py-5 text-white">
            <h1 className="text-xl font-bold flex items-center gap-2">
              <Radio className="w-5 h-5" /> Start a Live Auction
            </h1>
            <p className="text-red-100 text-sm mt-1">Go live and let viewers bid in real-time</p>
          </div>

          <div className="p-6 space-y-5">
            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-1.5">Stream Title *</label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="e.g. Rare vintage watch auction"
                className="w-full border border-gray-200 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-red-400"
              />
            </div>

            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-1.5">Description</label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={3}
                placeholder="Tell viewers what you're auctioning…"
                className="w-full border border-gray-200 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-red-400 resize-none"
              />
            </div>

            {myAuctions.length > 0 && (
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-1.5">
                  Link to an Auction (optional)
                </label>
                <select
                  value={auctionId}
                  onChange={(e) => setAuctionId(e.target.value)}
                  className="w-full border border-gray-200 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-red-400 bg-white"
                >
                  <option value="">— Select auction —</option>
                  {myAuctions.map((a) => (
                    <option key={a.id} value={a.id}>{a.title}</option>
                  ))}
                </select>
              </div>
            )}

            {error && (
              <div className="flex items-center gap-2 bg-red-50 border border-red-200 rounded-xl p-3">
                <AlertCircle className="w-4 h-4 text-red-500" />
                <p className="text-sm text-red-700">{error}</p>
              </div>
            )}

            <button
              onClick={handleCreate}
              disabled={creating || !title.trim()}
              className="w-full bg-red-500 hover:bg-red-600 text-white font-bold py-3 rounded-xl transition-colors flex items-center justify-center gap-2 disabled:opacity-50"
            >
              <Video className="w-4 h-4" />
              {creating ? 'Creating…' : 'Create Live Session'}
            </button>
          </div>
        </div>
      </div>
    );
  }

  // ── Step 2: Session created, ready to start or already streaming ─────────
  const isStreaming = session.status === 'live';
  const isEnded = session.status === 'ended' || session.status === 'cancelled';

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <button
        onClick={() => router.back()}
        className="flex items-center gap-1.5 text-gray-500 hover:text-[#0071CE] text-sm mb-5 transition-colors"
      >
        <ChevronLeft className="w-4 h-4" /> Back
      </button>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
        {/* Preview + controls */}
        <div className="lg:col-span-2 space-y-4">
          {isStreaming && streamData && (
            <LiveStreamPlayer
              token={streamData.token}
              livekitUrl={streamData.livekit_url}
              simulated={streamData.simulated}
              viewerCount={0}
            />
          )}

          {!isStreaming && !isEnded && (
            <div className="aspect-video bg-gray-900 rounded-2xl flex flex-col items-center justify-center gap-4">
              <Radio className="w-14 h-14 text-white/20" />
              <p className="text-white/50 text-sm">Your camera preview will appear here</p>
              <p className="text-white/30 text-xs">Click &quot;Go Live&quot; to start broadcasting</p>
            </div>
          )}

          {isEnded && (
            <div className="aspect-video bg-gray-100 rounded-2xl flex items-center justify-center">
              <div className="text-center">
                <CheckCircle className="w-12 h-12 text-green-400 mx-auto mb-3" />
                <p className="text-gray-600 font-semibold">Stream Ended</p>
                <button
                  onClick={() => router.push('/auctions/live')}
                  className="mt-4 text-sm text-[#0071CE] hover:underline"
                >
                  Back to Live Auctions
                </button>
              </div>
            </div>
          )}

          {/* Controls */}
          {!isEnded && (
            <div className="flex gap-3">
              {!isStreaming ? (
                <button
                  onClick={handleStart}
                  className="flex-1 bg-red-500 hover:bg-red-600 text-white font-bold py-3 rounded-xl transition-colors flex items-center justify-center gap-2"
                >
                  <Radio className="w-4 h-4 animate-pulse" />
                  Go Live
                </button>
              ) : (
                <button
                  onClick={handleEnd}
                  className="flex-1 bg-gray-800 hover:bg-gray-900 text-white font-bold py-3 rounded-xl transition-colors flex items-center justify-center gap-2"
                >
                  <StopCircle className="w-4 h-4" />
                  End Stream
                </button>
              )}
            </div>
          )}

          {error && (
            <div className="flex items-center gap-2 bg-red-50 border border-red-200 rounded-xl p-3">
              <AlertCircle className="w-4 h-4 text-red-500" />
              <p className="text-sm text-red-700">{error}</p>
            </div>
          )}
        </div>

        {/* Session info panel */}
        <div className="space-y-4">
          <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-4 space-y-4">
            <div>
              <p className="text-xs text-gray-500 uppercase tracking-wide font-semibold mb-1">Session</p>
              <p className="font-semibold text-gray-900 text-sm">{session.title}</p>
              <span className={`inline-flex items-center gap-1 mt-1 text-xs font-semibold px-2 py-0.5 rounded-full ${
                isStreaming ? 'bg-red-50 text-red-600' : isEnded ? 'bg-gray-100 text-gray-500' : 'bg-amber-50 text-amber-700'
              }`}>
                {isStreaming && <span className="w-1.5 h-1.5 bg-red-500 rounded-full animate-pulse" />}
                {session.status.toUpperCase()}
              </span>
            </div>

            <div>
              <p className="text-xs text-gray-500 uppercase tracking-wide font-semibold mb-1.5 flex items-center gap-1">
                <Link2 className="w-3 h-3" /> Share Link
              </p>
              <div className="flex items-center gap-2">
                <input
                  readOnly
                  value={viewerLink}
                  className="flex-1 text-xs bg-gray-50 border border-gray-200 rounded-lg px-2 py-1.5 text-gray-600 truncate"
                />
                <button
                  onClick={copyLink}
                  className="p-1.5 hover:bg-gray-100 rounded-lg transition-colors flex-shrink-0"
                  title="Copy link"
                >
                  {copied ? <CheckCircle className="w-4 h-4 text-green-500" /> : <Copy className="w-4 h-4 text-gray-500" />}
                </button>
              </div>
            </div>

            {streamData?.simulated && (
              <div className="bg-amber-50 border border-amber-200 rounded-xl p-3">
                <p className="text-xs text-amber-700 font-semibold">Simulation Mode</p>
                <p className="text-xs text-amber-600 mt-0.5">
                  Set <code className="bg-amber-100 px-1 rounded">LIVEKIT_API_KEY</code> in backend .env to enable real video streaming.
                </p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
