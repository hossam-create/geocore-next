'use client';

import { useEffect, useState } from 'react';
import { TrendingUp, AlertTriangle, Trophy, Zap, Volume2, VolumeX } from 'lucide-react';
import { useLiveSounds, type SoundType } from './useLiveSounds';

interface BidNotification {
  id: string;
  type: 'new_bid' | 'outbid' | 'sold' | 'extended' | 'won' | 'error';
  message: string;
  amountCents?: number;
  timestamp: number;
}

interface LiveBidNotificationProps {
  notifications: BidNotification[];
  onDismiss: (id: string) => void;
  soundEnabled?: boolean;
  onToggleSound?: () => void;
}

function formatCents(cents: number): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(cents / 100);
}

const typeConfig: Record<string, { bg: string; border: string; text: string; icon: React.ReactNode; sound: SoundType }> = {
  new_bid: {
    bg: 'bg-blue-50',
    border: 'border-blue-200',
    text: 'text-blue-800',
    icon: <TrendingUp className="w-4 h-4" />,
    sound: 'bid',
  },
  outbid: {
    bg: 'bg-amber-50',
    border: 'border-amber-300',
    text: 'text-amber-800',
    icon: <AlertTriangle className="w-4 h-4" />,
    sound: 'outbid',
  },
  sold: {
    bg: 'bg-green-50',
    border: 'border-green-200',
    text: 'text-green-800',
    icon: <Trophy className="w-4 h-4" />,
    sound: 'sold',
  },
  extended: {
    bg: 'bg-purple-50',
    border: 'border-purple-200',
    text: 'text-purple-800',
    icon: <Zap className="w-4 h-4" />,
    sound: 'extended',
  },
  won: {
    bg: 'bg-emerald-50',
    border: 'border-emerald-300',
    text: 'text-emerald-800',
    icon: <Trophy className="w-4 h-4" />,
    sound: 'won',
  },
  error: {
    bg: 'bg-red-50',
    border: 'border-red-200',
    text: 'text-red-800',
    icon: <AlertTriangle className="w-4 h-4" />,
    sound: 'error',
  },
};

export default function LiveBidNotification({
  notifications,
  onDismiss,
  soundEnabled = true,
  onToggleSound,
}: LiveBidNotificationProps) {
  const { play } = useLiveSounds(soundEnabled);
  const [animatingIds, setAnimatingIds] = useState<Set<string>>(new Set());

  // Play sound + animate on new notifications
  useEffect(() => {
    notifications.forEach((n) => {
      if (!animatingIds.has(n.id)) {
        const config = typeConfig[n.type];
        if (config) play(config.sound);
        setAnimatingIds((prev) => new Set(prev).add(n.id));
      }
    });
  }, [notifications, play]);

  // Auto-dismiss after 4s
  useEffect(() => {
    const timers = notifications.map((n) =>
      setTimeout(() => onDismiss(n.id), 4000)
    );
    return () => timers.forEach(clearTimeout);
  }, [notifications, onDismiss]);

  if (notifications.length === 0) return null;

  return (
    <div className="fixed top-4 right-4 z-50 space-y-2 max-w-sm">
      {/* Sound toggle */}
      {onToggleSound && (
        <button
          onClick={onToggleSound}
          className="absolute -top-1 -left-8 p-1.5 rounded-full bg-white shadow-md border border-gray-100 hover:bg-gray-50 transition-colors"
          title={soundEnabled ? 'Mute sounds' : 'Unmute sounds'}
        >
          {soundEnabled ? <Volume2 className="w-3.5 h-3.5 text-gray-600" /> : <VolumeX className="w-3.5 h-3.5 text-gray-400" />}
        </button>
      )}

      {notifications.slice(-3).map((n) => {
        const config = typeConfig[n.type] ?? typeConfig.new_bid;
        return (
          <div
            key={n.id}
            className={`flex items-center gap-2.5 px-4 py-3 rounded-xl border shadow-lg ${config.bg} ${config.border} animate-slide-in`}
            style={{
              animation: 'slideIn 0.3s ease-out, fadeOut 0.5s ease-in 3.5s forwards',
            }}
          >
            <span className={config.text}>{config.icon}</span>
            <div className="flex-1 min-w-0">
              <p className={`text-sm font-semibold ${config.text}`}>{n.message}</p>
              {n.amountCents != null && (
                <p className="text-xs text-gray-500 mt-0.5">{formatCents(n.amountCents)}</p>
              )}
            </div>
            <button
              onClick={() => onDismiss(n.id)}
              className="text-gray-400 hover:text-gray-600 text-xs ml-2"
            >
              ✕
            </button>
          </div>
        );
      })}

      <style jsx>{`
        @keyframes slideIn {
          from { transform: translateX(100%); opacity: 0; }
          to { transform: translateX(0); opacity: 1; }
        }
        @keyframes fadeOut {
          from { opacity: 1; }
          to { opacity: 0; }
        }
      `}</style>
    </div>
  );
}

export type { BidNotification };
