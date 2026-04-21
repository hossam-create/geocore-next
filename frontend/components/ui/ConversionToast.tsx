'use client';

import { useEffect, useState } from 'react';
import { Bell, TrendingUp, Truck, Gavel, X } from 'lucide-react';

interface ToastItem {
  id: string;
  icon: typeof Bell;
  message: string;
  color: string;
  bg: string;
}

const TOAST_ICONS = {
  offer: { icon: Gavel, color: 'text-blue-600', bg: 'bg-blue-50 border-blue-200' },
  traveler: { icon: Truck, color: 'text-emerald-600', bg: 'bg-emerald-50 border-emerald-200' },
  demand: { icon: TrendingUp, color: 'text-orange-600', bg: 'bg-orange-50 border-orange-200' },
  notification: { icon: Bell, color: 'text-purple-600', bg: 'bg-purple-50 border-purple-200' },
};

export type ToastType = keyof typeof TOAST_ICONS;

let toastQueue: ToastItem[] = [];
let toastListeners: Array<(toasts: ToastItem[]) => void> = [];

function notifyListeners() {
  toastListeners.forEach(fn => fn([...toastQueue]));
}

export function showConversionToast(type: ToastType, message: string) {
  const meta = TOAST_ICONS[type];
  const id = `${Date.now()}-${Math.random().toString(36).slice(2, 6)}`;
  const item: ToastItem = { id, icon: meta.icon, message, color: meta.color, bg: meta.bg };
  toastQueue = [...toastQueue, item];
  notifyListeners();
  setTimeout(() => {
    toastQueue = toastQueue.filter(t => t.id !== id);
    notifyListeners();
  }, 5000);
}

export function ConversionToastContainer() {
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  useEffect(() => {
    toastListeners.push(setToasts);
    return () => {
      toastListeners = toastListeners.filter(fn => fn !== setToasts);
    };
  }, []);

  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-4 right-4 z-50 space-y-2 max-w-sm">
      {toasts.map(t => {
        const Icon = t.icon;
        return (
          <div
            key={t.id}
            className={`flex items-center gap-2.5 rounded-xl border px-4 py-3 shadow-lg ${t.bg} animate-slide-in`}
          >
            <Icon size={16} className={t.color} />
            <p className="text-sm font-medium text-gray-800 flex-1">{t.message}</p>
            <button
              onClick={() => {
                toastQueue = toastQueue.filter(x => x.id !== t.id);
                notifyListeners();
              }}
              className="text-gray-400 hover:text-gray-600"
            >
              <X size={14} />
            </button>
          </div>
        );
      })}
    </div>
  );
}
