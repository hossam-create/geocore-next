'use client';
import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { useAuthStore } from '@/store/auth';
import api from '@/lib/api';
import {
  Bell, Mail, Smartphone, MessageSquare, ArrowLeft, Loader2,
  CheckCircle, Save
} from 'lucide-react';

interface NotificationPreferences {
  // Email
  email_new_message: boolean;
  email_auction_outbid: boolean;
  email_order_update: boolean;
  email_price_drop: boolean;
  email_promo_offers: boolean;

  // Push
  push_new_message: boolean;
  push_auction_outbid: boolean;
  push_order_update: boolean;
  push_price_drop: boolean;
  push_promo_offers: boolean;

  // SMS
  sms_new_message: boolean;
  sms_auction_outbid: boolean;
  sms_order_update: boolean;
  sms_price_drop: boolean;
  sms_promo_offers: boolean;
}

const DEFAULT_PREFS: NotificationPreferences = {
  email_new_message: true,
  email_auction_outbid: true,
  email_order_update: true,
  email_price_drop: true,
  email_promo_offers: true,
  push_new_message: true,
  push_auction_outbid: true,
  push_order_update: true,
  push_price_drop: true,
  push_promo_offers: true,
  sms_new_message: false,
  sms_auction_outbid: false,
  sms_order_update: true,
  sms_price_drop: false,
  sms_promo_offers: false,
};

const EVENT_LABELS = {
  new_message: { label: 'New Message', desc: 'When you receive a new message' },
  auction_outbid: { label: 'Auction Outbid', desc: 'When someone outbids you' },
  order_update: { label: 'Order Update', desc: 'Status changes on your orders' },
  price_drop: { label: 'Price Drop', desc: 'Price drops on your watchlist items' },
  promo_offers: { label: 'Promo Offers', desc: 'Special deals and promotions' },
};

export default function NotificationSettingsPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [mounted, setMounted] = useState(false);
  const [prefs, setPrefs] = useState<NotificationPreferences>(DEFAULT_PREFS);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (mounted && !isAuthenticated) {
      router.push('/login?redirect=/settings/notifications');
    }
  }, [mounted, isAuthenticated, router]);

  useEffect(() => {
    if (!isAuthenticated) return;

    const fetchPrefs = async () => {
      setLoading(true);
      try {
        const res = await api.get('/users/me/notification-preferences');
        setPrefs({ ...DEFAULT_PREFS, ...res.data });
      } catch {
        // Use defaults
      } finally {
        setLoading(false);
      }
    };

    fetchPrefs();
  }, [isAuthenticated]);

  if (!mounted || !isAuthenticated) return null;

  const handleToggle = (channel: 'email' | 'push' | 'sms', event: string) => {
    const key = `${channel}_${event}` as keyof NotificationPreferences;
    setPrefs((p) => ({ ...p, [key]: !p[key] }));
    setSaved(false);
  };

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      await api.patch('/users/me/notification-preferences', prefs);
      setSaved(true);
      setTimeout(() => setSaved(false), 3000);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to save preferences');
    } finally {
      setSaving(false);
    }
  };

  const Toggle = ({ checked, onChange }: { checked: boolean; onChange: () => void }) => (
    <button
      onClick={onChange}
      className={`relative w-11 h-6 rounded-full transition-colors ${
        checked ? 'bg-[#0071CE]' : 'bg-gray-200'
      }`}
    >
      <span
        className={`absolute top-1 left-1 w-4 h-4 rounded-full bg-white transition-transform ${
          checked ? 'translate-x-5' : 'translate-x-0'
        }`}
      />
    </button>
  );

  return (
    <div className="mx-auto max-w-2xl px-4 py-8">
      {/* Header */}
      <div className="mb-6 flex items-center gap-3">
        <Link href="/profile" className="rounded-lg p-2 hover:bg-gray-100">
          <ArrowLeft size={20} className="text-gray-600" />
        </Link>
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Notification Settings</h1>
          <p className="text-sm text-gray-500">Control how you receive notifications</p>
        </div>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 size={32} className="animate-spin text-[#0071CE]" />
        </div>
      ) : (
        <>
          {error && (
            <div className="mb-6 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
              {error}
            </div>
          )}

          {/* Email Notifications */}
          <div className="mb-6 rounded-xl border border-gray-200 bg-white overflow-hidden">
            <div className="flex items-center gap-3 px-5 py-4 border-b border-gray-100 bg-gray-50">
              <Mail size={20} className="text-[#0071CE]" />
              <h2 className="text-sm font-semibold text-gray-900">Email Notifications</h2>
            </div>
            <div className="divide-y divide-gray-100">
              {Object.entries(EVENT_LABELS).map(([event, { label, desc }]) => (
                <div key={`email-${event}`} className="flex items-center justify-between px-5 py-4">
                  <div>
                    <p className="text-sm font-medium text-gray-900">{label}</p>
                    <p className="text-xs text-gray-400">{desc}</p>
                  </div>
                  <Toggle
                    checked={prefs[`email_${event}` as keyof NotificationPreferences]}
                    onChange={() => handleToggle('email', event)}
                  />
                </div>
              ))}
            </div>
          </div>

          {/* Push Notifications */}
          <div className="mb-6 rounded-xl border border-gray-200 bg-white overflow-hidden">
            <div className="flex items-center gap-3 px-5 py-4 border-b border-gray-100 bg-gray-50">
              <Bell size={20} className="text-purple-600" />
              <h2 className="text-sm font-semibold text-gray-900">Push Notifications</h2>
            </div>
            <div className="divide-y divide-gray-100">
              {Object.entries(EVENT_LABELS).map(([event, { label, desc }]) => (
                <div key={`push-${event}`} className="flex items-center justify-between px-5 py-4">
                  <div>
                    <p className="text-sm font-medium text-gray-900">{label}</p>
                    <p className="text-xs text-gray-400">{desc}</p>
                  </div>
                  <Toggle
                    checked={prefs[`push_${event}` as keyof NotificationPreferences]}
                    onChange={() => handleToggle('push', event)}
                  />
                </div>
              ))}
            </div>
          </div>

          {/* SMS Notifications */}
          <div className="mb-6 rounded-xl border border-gray-200 bg-white overflow-hidden">
            <div className="flex items-center gap-3 px-5 py-4 border-b border-gray-100 bg-gray-50">
              <Smartphone size={20} className="text-green-600" />
              <h2 className="text-sm font-semibold text-gray-900">SMS Notifications</h2>
            </div>
            <div className="divide-y divide-gray-100">
              {Object.entries(EVENT_LABELS).map(([event, { label, desc }]) => (
                <div key={`sms-${event}`} className="flex items-center justify-between px-5 py-4">
                  <div>
                    <p className="text-sm font-medium text-gray-900">{label}</p>
                    <p className="text-xs text-gray-400">{desc}</p>
                  </div>
                  <Toggle
                    checked={prefs[`sms_${event}` as keyof NotificationPreferences]}
                    onChange={() => handleToggle('sms', event)}
                  />
                </div>
              ))}
            </div>
          </div>

          {/* Save Button */}
          <div className="flex items-center justify-between">
            <Link href="/profile" className="text-sm text-gray-500 hover:text-gray-700">
              ← Back to Profile
            </Link>
            <button
              onClick={handleSave}
              disabled={saving}
              className="flex items-center gap-2 rounded-xl bg-[#0071CE] px-6 py-3 text-sm font-semibold text-white hover:bg-[#005ba3] disabled:bg-gray-300 transition-colors"
            >
              {saving ? (
                <>
                  <Loader2 size={16} className="animate-spin" /> Saving...
                </>
              ) : saved ? (
                <>
                  <CheckCircle size={16} /> Saved!
                </>
              ) : (
                <>
                  <Save size={16} /> Save Preferences
                </>
              )}
            </button>
          </div>
        </>
      )}
    </div>
  );
}
