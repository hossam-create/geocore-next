"use client";

import PageHeader from "@/components/shared/PageHeader";

const NOTIFICATIONS = [
  { id: 1, text: "Escrow ESC-001 awaiting decision", time: "1 min ago" },
  { id: 2, text: "High risk alert for user U-005", time: "5 min ago" },
  { id: 3, text: "Daily revenue report is ready", time: "12 min ago" },
];

export default function NotificationsPage() {
  return (
    <div>
      <PageHeader title="Notifications" description="System and workflow alerts" />
      <div className="space-y-3">
        {NOTIFICATIONS.map((n) => (
          <div key={n.id} className="surface p-4">
            <p style={{ color: "var(--text-primary)" }}>{n.text}</p>
            <p className="text-xs mt-1" style={{ color: "var(--text-tertiary)" }}>{n.time}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
