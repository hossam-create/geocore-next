'use client';

import { use, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import dynamic from 'next/dynamic';
import Link from 'next/link';
import api from '@/lib/api';
import { useAuthStore } from '@/store/auth';
import { ChevronLeft, Radio, AlertCircle } from 'lucide-react';
import LiveStreamChat from '@/components/livestream/LiveStreamChat';
import LiveStreamBidPanel from '@/components/livestream/LiveStreamBidPanel';
import LiveBidBox from '@/components/livestream/LiveBidBox';

const LiveStreamPlayer = dynamic(
  () => import('@/components/livestream/LiveStreamPlayer'),
  { ssr: false }
);

interface LiveSession {
  id: string;
  title: string;
  description: string;
  status: string;
  viewer_count: number;
  room_name: string;
  started_at: string | null;
  auction_id: string | null;
}

interface JoinResponse {
  session: LiveSession;
  token: string;
  livekit_url: string;
  simulated: boolean;
}

interface AuctionInfo {
  id: string;
  current_bid: number;
  start_price: number;
  currency: string;
  end_time: string;
  min_bid_increment?: number;
}

export default function LiveViewerPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const { isAuthenticated, user } = useAuthStore();
  const [joinData, setJoinData] = useState<JoinResponse | null>(null);
  const [joinError, setJoinError] = useState('');

  const { data: session, isLoading: sessionLoading } = useQuery<LiveSession>({
    queryKey: ['livestream-session', id],
    queryFn: async () => {
      const res = await api.get(`/livestream/${id}`);
      return res.data?.data ?? res.data;
    },
    refetchInterval: 10_000,
  });

  const { data: auction } = useQuery<AuctionInfo>({
    queryKey: ['auction-for-stream', session?.auction_id],
    queryFn: async () => {
      const res = await api.get(`/auctions/${session!.auction_id}`);
      return res.data?.data ?? res.data;
    },
    enabled: !!session?.auction_id,
    refetchInterval: 5_000,
  });

  useEffect(() => {
    if (!session || session.status !== 'live') return;
    if (joinData) return;

    api
      .post(`/livestream/${id}/join`, { display_name: user?.name || 'Viewer' })
      .then((r) => setJoinData(r.data?.data ?? r.data))
      .catch((e) =>
        setJoinError(e?.response?.data?.message ?? 'Could not join the stream')
      );
  }, [id, session, joinData, user]);

  if (sessionLoading) {
    return (
      <div className="max-w-6xl mx-auto px-4 py-10">
        <div className="animate-pulse space-y-4">
          <div className="h-8 bg-gray-100 rounded-xl w-40" />
          <div className="aspect-video bg-gray-100 rounded-2xl" />
        </div>
      </div>
    );
  }

  if (!session) {
    return (
      <div className="max-w-6xl mx-auto px-4 py-10 text-center">
        <AlertCircle className="w-12 h-12 text-gray-300 mx-auto mb-3" />
        <p className="text-gray-600 font-medium">Stream not found</p>
        <Link href="/auctions/live" className="mt-3 inline-block text-[#0071CE] text-sm hover:underline">
          ← Back to Live Auctions
        </Link>
      </div>
    );
  }

  const isEnded = session.status === 'ended' || session.status === 'cancelled';
  const isScheduled = session.status === 'scheduled';

  return (
    <div className="max-w-6xl mx-auto px-4 py-8">
      <button
        onClick={() => router.back()}
        className="flex items-center gap-1.5 text-gray-500 hover:text-[#0071CE] text-sm mb-5 transition-colors"
      >
        <ChevronLeft className="w-4 h-4" /> Live Auctions
      </button>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
        {/* Left: video + info */}
        <div className="lg:col-span-2 space-y-4">
          {isScheduled && (
            <div className="bg-amber-50 border border-amber-200 rounded-2xl p-4 flex items-center gap-3">
              <Radio className="w-5 h-5 text-amber-600" />
              <p className="text-sm text-amber-800">This stream hasn&apos;t started yet. Refresh when it goes live.</p>
            </div>
          )}

          {isEnded && (
            <div className="bg-gray-50 border border-gray-200 rounded-2xl p-4 text-center">
              <p className="text-gray-600 font-medium">This stream has ended</p>
            </div>
          )}

          {joinError && (
            <div className="bg-red-50 border border-red-200 rounded-2xl p-4 flex items-center gap-3">
              <AlertCircle className="w-5 h-5 text-red-500" />
              <p className="text-sm text-red-700">{joinError}</p>
            </div>
          )}

          {session.status === 'live' && joinData && (
            <LiveStreamPlayer
              token={joinData.token}
              livekitUrl={joinData.livekit_url}
              simulated={joinData.simulated}
              viewerCount={session.viewer_count}
            />
          )}

          {session.status === 'live' && !joinData && !joinError && (
            <div className="aspect-video bg-gray-900 rounded-2xl flex items-center justify-center">
              <div className="w-10 h-10 border-4 border-white/30 border-t-white rounded-full animate-spin" />
            </div>
          )}

          <div>
            <h1 className="text-xl font-bold text-gray-900 flex items-center gap-2">
              {session.status === 'live' && (
                <span className="text-xs font-bold text-red-500 bg-red-50 border border-red-200 px-2 py-0.5 rounded-full animate-pulse">
                  LIVE
                </span>
              )}
              {session.title}
            </h1>
            {session.description && (
              <p className="text-sm text-gray-500 mt-1">{session.description}</p>
            )}
          </div>
        </div>

        {/* Right: live items + bid + chat */}
        <div className="space-y-4 flex flex-col">
          {/* New Live Bid Box with WS support */}
          <LiveBidBox
            sessionId={id}
            isAuthenticated={isAuthenticated}
            currentUserId={user?.id}
          />
          {/* Legacy auction bid panel (shown only if session links to a classic auction) */}
          {session.auction_id && !process.env.NEXT_PUBLIC_FF_LIVE_AUCTION && (
            <LiveStreamBidPanel
              auctionId={session.auction_id ?? undefined}
              currentBid={auction?.current_bid ?? auction?.start_price ?? 0}
              minIncrement={auction?.min_bid_increment ?? 10}
              currency={auction?.currency ?? 'AED'}
              endsAt={auction?.end_time}
              isAuthenticated={isAuthenticated}
            />
          )}
          <div className="flex-1 min-h-[350px]">
            <LiveStreamChat
              sessionId={id}
              currentUser={user?.name || 'Viewer'}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
