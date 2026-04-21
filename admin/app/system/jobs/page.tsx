"use client";

import PageHeader from "@/components/shared/PageHeader";

const JOBS = [
  { name: "sync:listings", status: "running", last: "2m ago" },
  { name: "alerts:processor", status: "running", last: "just now" },
  { name: "reports:daily", status: "scheduled", last: "today 01:00" },
  { name: "cleanup:sessions", status: "completed", last: "5m ago" },
];

export default function JobsPage() {
  return (
    <div>
      <PageHeader title="Jobs" description="Background workers and scheduled tasks" />
      <div className="space-y-3">
        {JOBS.map((j) => (
          <div key={j.name} className="surface p-4 flex items-center justify-between">
            <div>
              <p className="font-medium" style={{ color: "var(--text-primary)" }}>{j.name}</p>
              <p className="text-xs" style={{ color: "var(--text-tertiary)" }}>Last run: {j.last}</p>
            </div>
            <span className="text-xs px-2 py-1 rounded" style={{ background: "var(--bg-surface-active)", color: "var(--text-secondary)" }}>{j.status}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
