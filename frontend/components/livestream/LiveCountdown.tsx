'use client';

import { useState, useEffect } from 'react';
import { Clock, AlertTriangle, Zap } from 'lucide-react';

type CountdownStage = 'safe' | 'warning' | 'urgent' | 'expired';

interface LiveCountdownProps {
  endsAt: string | null | undefined;
  onExpired?: () => void;
  extended?: boolean;
}

const stageConfig: Record<CountdownStage, { bg: string; text: string; pulse: boolean }> = {
  safe:    { bg: 'bg-emerald-100', text: 'text-emerald-700', pulse: false },
  warning: { bg: 'bg-amber-100',   text: 'text-amber-700',   pulse: false },
  urgent:  { bg: 'bg-red-100',     text: 'text-red-700',     pulse: true  },
  expired: { bg: 'bg-gray-100',    text: 'text-gray-500',    pulse: false },
};

export default function LiveCountdown({ endsAt, onExpired, extended }: LiveCountdownProps) {
  const [remaining, setRemaining] = useState('');
  const [stage, setStage] = useState<CountdownStage>('safe');

  useEffect(() => {
    if (!endsAt) return;

    const tick = () => {
      const diff = new Date(endsAt).getTime() - Date.now();
      if (diff <= 0) {
        setRemaining('ENDED');
        setStage('expired');
        onExpired?.();
        return;
      }

      // 3-stage color system
      if (diff < 10_000) {
        setStage('urgent');
      } else if (diff < 30_000) {
        setStage('warning');
      } else {
        setStage('safe');
      }

      const s = Math.floor(diff / 1000);
      const m = Math.floor(s / 60);
      const h = Math.floor(m / 60);
      if (h > 0) {
        setRemaining(`${h}h ${m % 60}m ${s % 60}s`);
      } else if (m > 0) {
        setRemaining(`${m}m ${s % 60}s`);
      } else {
        setRemaining(`${s}s`);
      }
    };

    tick();
    const iv = setInterval(tick, 1000);
    return () => clearInterval(iv);
  }, [endsAt, onExpired]);

  if (!endsAt) return null;

  const cfg = stageConfig[stage];

  return (
    <div
      className={`flex items-center gap-1.5 text-xs font-bold px-2.5 py-1 rounded-full ${cfg.bg} ${cfg.text} ${
        cfg.pulse ? 'animate-pulse' : ''
      }`}
    >
      {stage === 'expired' ? (
        <Clock className="w-3 h-3" />
      ) : stage === 'urgent' ? (
        <AlertTriangle className="w-3 h-3" />
      ) : extended ? (
        <Zap className="w-3 h-3" />
      ) : (
        <Clock className="w-3 h-3" />
      )}
      {remaining}
      {extended && stage !== 'expired' && (
        <span className="text-[10px] font-semibold ml-0.5 opacity-75">EXT</span>
      )}
    </div>
  );
}
