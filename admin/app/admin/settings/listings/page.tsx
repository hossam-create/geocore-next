"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { settingsApi } from "@/lib/api";
import SettingField from "@/components/settings/SettingField";
import { useToastStore } from "@/lib/toast";
import type { AdminSetting } from "@/lib/types";

export default function AdminListingsSettingsPage() {
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const { data: settings = [], isLoading } = useQuery<AdminSetting[]>({
    queryKey: ["admin-settings", "listings"],
    queryFn: () => settingsApi.getByCategory("listings"),
  });

  const handleUpdate = async (key: string, value: unknown) => {
    try {
      await settingsApi.update(key, value);
      qc.invalidateQueries({ queryKey: ["admin-settings"] });
      showToast({ type: "success", title: "Setting updated", message: `${key} has been saved.` });
    } catch (error) {
      const message = (error as { message?: string } | null)?.message ?? "Could not save setting.";
      showToast({ type: "error", title: "Update failed", message });
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Listings Settings</h1>
        <p className="text-sm text-slate-500 mt-0.5">Listing rules, approval requirements, and limits</p>
      </div>
      <div className="bg-white rounded-xl border border-slate-200 p-5">
        {isLoading ? (
          <p className="text-sm text-slate-400 py-8 text-center">Loading...</p>
        ) : (
          settings.map((s) => <SettingField key={s.key} setting={s} onUpdate={handleUpdate} />)
        )}
      </div>
    </div>
  );
}
