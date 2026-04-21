'use client';

import { useEffect, useRef, useCallback } from 'react';

type SoundType = 'bid' | 'outbid' | 'sold' | 'extended' | 'won' | 'error';

// Web Audio API — no external files needed, generates tones programmatically.
function createBeep(frequency: number, duration: number, volume = 0.15): OscillatorNode {
  const ctx = new AudioContext();
  const osc = ctx.createOscillator();
  const gain = ctx.createGain();
  osc.connect(gain);
  gain.connect(ctx.destination);
  osc.frequency.value = frequency;
  gain.gain.value = volume;
  osc.start();
  osc.stop(ctx.currentTime + duration / 1000);
  return osc;
}

const SOUNDS: Record<SoundType, () => void> = {
  bid: () => createBeep(880, 120, 0.1),        // short high ping
  outbid: () => createBeep(440, 250, 0.15),     // lower, longer — urgency
  sold: () => {                                   // two-tone celebration
    createBeep(660, 100, 0.12);
    setTimeout(() => createBeep(880, 150, 0.12), 120);
  },
  extended: () => createBeep(550, 180, 0.1),    // medium tone
  won: () => {                                    // ascending triple
    createBeep(440, 80, 0.1);
    setTimeout(() => createBeep(660, 80, 0.1), 100);
    setTimeout(() => createBeep(880, 120, 0.12), 200);
  },
  error: () => createBeep(220, 300, 0.15),       // low buzz
};

export function useLiveSounds(enabled = true) {
  const enabledRef = useRef(enabled);
  enabledRef.current = enabled;

  const play = useCallback((type: SoundType) => {
    if (!enabledRef.current) return;
    try {
      SOUNDS[type]();
    } catch {
      // AudioContext may not be available (SSR, permissions, etc.)
    }
  }, []);

  return { play };
}

export type { SoundType };
