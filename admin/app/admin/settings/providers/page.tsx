"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { settingsApi } from "@/lib/api";
import SettingField from "@/components/settings/SettingField";
import { useToastStore } from "@/lib/toast";
import type { AdminSetting } from "@/lib/types";
import {
  Globe,
  CreditCard,
  Mail,
  HardDrive,
  MapPin,
  MessageSquare,
  Bell,
  ChevronDown,
  ChevronUp,
  CheckCircle2,
  AlertCircle,
} from "lucide-react";
import { useState, useMemo } from "react";

// Provider group definitions
const PROVIDER_GROUPS: Array<{
  id: string;
  title: string;
  description: string;
  icon: React.ReactNode;
  category: string;
  enableKey?: string;
}> = [
  {
    id: "oauth",
    title: "Social Login (OAuth)",
    description: "Google, Apple, and Facebook login credentials",
    icon: <Globe className="w-5 h-5" />,
    category: "oauth",
  },
  {
    id: "payments",
    title: "Payment Gateways",
    description: "Stripe, PayPal, PayMob, Tabby, Tamara and crypto",
    icon: <CreditCard className="w-5 h-5" />,
    category: "payments",
  },
  {
    id: "email",
    title: "Email",
    description: "SMTP, Resend, and SendGrid",
    icon: <Mail className="w-5 h-5" />,
    category: "email",
  },
  {
    id: "storage",
    title: "File Storage",
    description: "Cloudflare R2 and AWS S3 credentials",
    icon: <HardDrive className="w-5 h-5" />,
    category: "storage",
  },
  {
    id: "maps",
    title: "Maps & Geolocation",
    description: "Google Maps and Mapbox API keys",
    icon: <MapPin className="w-5 h-5" />,
    category: "maps",
  },
  {
    id: "sms",
    title: "SMS & OTP",
    description: "Twilio and Vonage for SMS verification",
    icon: <MessageSquare className="w-5 h-5" />,
    category: "sms",
  },
  {
    id: "push",
    title: "Push Notifications",
    description: "Firebase FCM and Apple APNs",
    icon: <Bell className="w-5 h-5" />,
    category: "push",
  },
];

function isConfigured(settings: AdminSetting[]): boolean {
  return settings.some((s) => {
    if (s.type === "bool") return false;
    try {
      const v = JSON.parse(s.value);
      return typeof v === "string" && v.length > 0;
    } catch {
      return s.value.length > 2;
    }
  });
}

function ProviderCard({
  group,
  settings,
  onUpdate,
}: {
  group: (typeof PROVIDER_GROUPS)[0];
  settings: AdminSetting[];
  onUpdate: (key: string, value: unknown) => Promise<void>;
}) {
  const [open, setOpen] = useState(false);
  const configured = useMemo(() => isConfigured(settings), [settings]);

  return (
    <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
      {/* Header */}
      <button
        onClick={() => setOpen((o) => !o)}
        className="w-full flex items-center justify-between px-5 py-4 hover:bg-slate-50 transition-colors text-left"
      >
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-slate-100 text-slate-600">
            {group.icon}
          </div>
          <div>
            <div className="flex items-center gap-2">
              <p className="font-semibold text-slate-900">{group.title}</p>
              {configured ? (
                <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-green-50 text-green-700 font-medium">
                  <CheckCircle2 className="w-3 h-3" />
                  Configured
                </span>
              ) : (
                <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-amber-50 text-amber-700 font-medium">
                  <AlertCircle className="w-3 h-3" />
                  Not configured
                </span>
              )}
            </div>
            <p className="text-sm text-slate-500 mt-0.5">{group.description}</p>
          </div>
        </div>
        {open ? (
          <ChevronUp className="w-4 h-4 text-slate-400 shrink-0" />
        ) : (
          <ChevronDown className="w-4 h-4 text-slate-400 shrink-0" />
        )}
      </button>

      {/* Settings fields */}
      {open && (
        <div className="border-t border-slate-100 px-5 py-4 space-y-1">
          {settings.length === 0 ? (
            <p className="text-sm text-slate-400 text-center py-4">
              No settings found for this provider.
            </p>
          ) : (
            settings.map((s) => (
              <SettingField key={s.key} setting={s} onUpdate={onUpdate} />
            ))
          )}
        </div>
      )}
    </div>
  );
}

const PROVIDER_CATEGORIES = ["oauth", "payments", "email", "storage", "maps", "sms", "push"];

export default function AdminProvidersPage() {
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);

  // Fetch all settings for provider categories in one shot
  const { data: allSettings = [], isLoading } = useQuery<AdminSetting[]>({
    queryKey: ["admin-settings-providers"],
    queryFn: async () => {
      const results = await Promise.all(
        PROVIDER_CATEGORIES.map((cat) => settingsApi.getByCategory(cat))
      );
      return results.flat();
    },
  });

  const handleUpdate = async (key: string, value: unknown) => {
    try {
      await settingsApi.update(key, value);
      qc.invalidateQueries({ queryKey: ["admin-settings-providers"] });
      qc.invalidateQueries({ queryKey: ["admin-settings"] });
      showToast({
        type: "success",
        title: "Saved",
        message: `${key} updated successfully.`,
      });
    } catch (error) {
      const message =
        (error as { message?: string } | null)?.message ?? "Could not save setting.";
      showToast({ type: "error", title: "Update failed", message });
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">External Providers</h1>
        <p className="text-sm text-slate-500 mt-0.5">
          Configure credentials for all third-party services — OAuth, payments, storage, maps, SMS, and push notifications
        </p>
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {PROVIDER_GROUPS.map((g) => (
            <div key={g.id} className="bg-white rounded-xl border border-slate-200 h-16 animate-pulse" />
          ))}
        </div>
      ) : (
        <div className="space-y-3">
          {PROVIDER_GROUPS.map((g) => (
            <ProviderCard
              key={g.id}
              group={g}
              settings={allSettings.filter((s) => s.category === g.category)}
              onUpdate={handleUpdate}
            />
          ))}
        </div>
      )}
    </div>
  );
}
