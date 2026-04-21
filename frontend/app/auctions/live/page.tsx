'use client';

import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import api from '@/lib/api';
import { Radio, Eye, Users, Calendar, ArrowRight } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';
import { useTranslations } from 'next-intl';

interface LiveSession {
  id: string;
  title: string;
  description: string;
  status: string;
  viewer_count: number;
  started_at: string | null;
  thumbnail_url: string | null;
  auction_id: string | null;
  host_id: string;
}

function SessionCard({ s }: { s: LiveSession }) {
  const isLive = s.status === 'live';
  return (
    <Link
      href={`/auctions/live/${s.id}`}
      className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden hover:shadow-md transition-shadow group"
    >
      <div className="relative aspect-video bg-gray-900">
        {s.thumbnail_url ? (
          <img src={s.thumbnail_url} alt={s.title} className="w-full h-full object-cover" />
        ) : (
          <div className="w-full h-full flex items-center justify-center">
            <Radio className="w-12 h-12 text-white/20" />
          </div>
        )}
        <div className="absolute inset-0 bg-gradient-to-t from-black/60 to-transparent" />

        {isLive && (
          <span className="absolute top-3 left-3 bg-red-500 text-white text-xs font-bold px-2.5 py-1 rounded-full flex items-center gap-1.5 animate-pulse">
            <span className="w-1.5 h-1.5 bg-white rounded-full" />
            LIVE
          </span>
        )}
        {!isLive && (
          <span className="absolute top-3 left-3 bg-gray-600 text-white text-xs font-semibold px-2.5 py-1 rounded-full capitalize">
            {s.status}
          </span>
        )}

        <span className="absolute top-3 right-3 bg-black/50 text-white text-xs px-2 py-1 rounded-full flex items-center gap-1">
          <Eye className="w-3 h-3" /> {s.viewer_count}
        </span>
      </div>

      <div className="p-4">
        <h3 className="font-semibold text-gray-900 group-hover:text-[#0071CE] transition-colors line-clamp-2 mb-1">
          {s.title}
        </h3>
        {s.description && (
          <p className="text-xs text-gray-500 line-clamp-2 mb-3">{s.description}</p>
        )}
        <div className="flex items-center justify-between text-xs text-gray-400">
          <span className="flex items-center gap-1">
            <Calendar className="w-3 h-3" />
            {s.started_at
              ? `Started ${formatDistanceToNow(new Date(s.started_at), { addSuffix: true })}`
              : 'Not started yet'}
          </span>
          <span className="flex items-center gap-1 text-[#0071CE] font-medium">
            Watch <ArrowRight className="w-3 h-3" />
          </span>
        </div>
      </div>
    </Link>
  );
}

export default function LiveAuctionsPage() {
  const t = useTranslations("livestream");
  const { data: sessions = [], isLoading } = useQuery<LiveSession[]>({
    queryKey: ['livestream', 'live'],
    queryFn: async () => {
      const res = await api.get('/livestream?status=live');
      return res.data?.data ?? res.data ?? [];
    },
    refetchInterval: 15_000,
  });

  const { data: scheduled = [] } = useQuery<LiveSession[]>({
    queryKey: ['livestream', 'scheduled'],
    queryFn: async () => {
      const res = await api.get('/livestream?status=scheduled');
      return res.data?.data ?? res.data ?? [];
    },
  });

  return (
    <div className="max-w-6xl mx-auto px-4 py-10">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-extrabold text-gray-900 flex items-center gap-3">
            <span className="w-10 h-10 bg-red-500 rounded-xl flex items-center justify-center">
              <Radio className="w-5 h-5 text-white" />
            </span>
            {t("title")}
          </h1>
          <p className="text-gray-500 text-sm mt-1">{t("subtitle")}</p>
        </div>
        <Link
          href="/sell/live"
          className="bg-[#0071CE] text-white font-semibold px-5 py-2.5 rounded-xl hover:bg-[#005BA1] transition-colors text-sm flex items-center gap-2"
        >
          <Radio className="w-4 h-4" /> {t("goLive")}
        </Link>
      </div>

      {/* Live now */}
      <section className="mb-10">
        <h2 className="text-lg font-bold text-gray-800 mb-4 flex items-center gap-2">
          <span className="w-2 h-2 bg-red-500 rounded-full animate-pulse" />
          Live Now
          {sessions.length > 0 && (
            <span className="text-sm font-normal text-gray-500">({sessions.length})</span>
          )}
        </h2>

        {isLoading ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
            {[1, 2, 3].map((i) => (
              <div key={i} className="rounded-2xl bg-gray-100 animate-pulse aspect-video" />
            ))}
          </div>
        ) : sessions.length === 0 ? (
          <div className="text-center py-16 bg-gray-50 rounded-2xl border border-dashed border-gray-200">
            <Radio className="w-12 h-12 text-gray-300 mx-auto mb-3" />
            <p className="text-gray-500 font-medium">{t("noStreams")}</p>
            <p className="text-sm text-gray-400 mt-1">Be the first to go live!</p>
            <Link href="/sell/live" className="mt-4 inline-block text-[#0071CE] text-sm hover:underline">
              Start a live auction →
            </Link>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
            {sessions.map((s) => <SessionCard key={s.id} s={s} />)}
          </div>
        )}
      </section>

      {/* Scheduled */}
      {scheduled.length > 0 && (
        <section>
          <h2 className="text-lg font-bold text-gray-800 mb-4 flex items-center gap-2">
            <Calendar className="w-5 h-5 text-gray-400" />
            Upcoming
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
            {scheduled.map((s) => <SessionCard key={s.id} s={s} />)}
          </div>
        </section>
      )}
    </div>
  );
}
