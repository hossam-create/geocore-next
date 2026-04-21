"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { settingsApi } from "@/lib/api";
import SettingField from "@/components/settings/SettingField";
import PageHeader from "@/components/shared/PageHeader";
import { useToastStore } from "@/lib/toast";
import type { AdminSetting } from "@/lib/types";

export default function TrustSettingsPage() {
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const { data: settings = [], isLoading } = useQuery<AdminSetting[]>({
    queryKey: ["admin-settings", "trust"],
    queryFn: () => settingsApi.getByCategory("trust"),
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
      <PageHeader
        title="Trust & Safety"
        description="Auto-ban thresholds, fraud detection, verification requirements"
      />
      <div className="surface rounded-xl p-5">
        {isLoading ? (
          <p className="text-sm py-8 text-center" style={{ color: "var(--text-tertiary)" }}>Loading settings...</p>
        ) : settings.length === 0 ? (
          <p className="text-sm py-8 text-center" style={{ color: "var(--text-tertiary)" }}>No trust settings found. Run backend seed first.</p>
        ) : (
          settings.map((s) => <SettingField key={s.key} setting={s} onUpdate={handleUpdate} />)
        )}
      </div>
    </div>
  );
}
