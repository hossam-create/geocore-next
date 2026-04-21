"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import { mockOpsHealth } from "@/lib/mockData";
import { opsApi } from "@/lib/api";
import { useToastStore } from "@/lib/toast";
import { Activity, Database, Server, Wifi, Clock, AlertTriangle, Zap, HardDrive, Play } from "lucide-react";

type HealthMetric = {
  label: string;
  value: string | number;
  unit?: string;
  icon: React.ComponentType<{ className?: string; style?: React.CSSProperties }>;
  color: string;
  status: "healthy" | "warning" | "critical";
};

function getStatus(value: number, warnThreshold: number, critThreshold: number): "healthy" | "warning" | "critical" {
  if (value >= critThreshold) return "critical";
  if (value >= warnThreshold) return "warning";
  return "healthy";
}

function statusColor(status: string) {
  switch (status) {
    case "critical": return "var(--color-danger)";
    case "warning": return "var(--color-warning)";
    default: return "var(--color-success)";
  }
}

export default function OpsHealthPage() {
  const showToast = useToastStore((s) => s.showToast);
  const [triggering, setTriggering] = useState<string | null>(null);

  const { data: liveHealth, isLoading } = useQuery({
    queryKey: ["ops", "health"],
    queryFn: () => opsApi.health(),
    refetchInterval: 30000,
    retry: 1,
  });

  const triggerJob = async (name: string) => {
    setTriggering(name);
    try {
      await opsApi.triggerJob(name);
      showToast({ type: "success", title: "Job triggered", message: `${name} has been queued.` });
    } catch {
      showToast({ type: "error", title: "Trigger failed", message: `Could not trigger ${name}.` });
    } finally {
      setTriggering(null);
    }
  };

  const h = liveHealth ?? mockOpsHealth;

  const metrics: HealthMetric[] = [
    { label: "API Latency P95", value: h.api_latency_p95_ms, unit: "ms", icon: Zap, color: "#3b82f6", status: getStatus(h.api_latency_p95_ms, 300, 1000) },
    { label: "DB Connections", value: `${h.db_connections}/${h.db_max_connections}`, icon: Database, color: "#8b5cf6", status: getStatus((h.db_connections / h.db_max_connections) * 100, 70, 90) },
    { label: "Redis Memory", value: `${h.redis_memory_mb}/${h.redis_max_memory_mb}`, unit: "MB", icon: HardDrive, color: "#ef4444", status: getStatus((h.redis_memory_mb / h.redis_max_memory_mb) * 100, 70, 90) },
    { label: "Active Auctions", value: h.active_auctions, icon: Activity, color: "#f59e0b", status: "healthy" },
    { label: "WebSocket Connections", value: h.active_websocket_connections, icon: Wifi, color: "#06b6d4", status: "healthy" },
    { label: "Job Queue Depth", value: h.job_queue_depth, icon: Clock, color: "#6366f1", status: getStatus(h.job_queue_depth, 50, 200) },
    { label: "Error Rate (1h)", value: `${(h.error_rate_1h * 100).toFixed(2)}%`, icon: AlertTriangle, color: "#ef4444", status: getStatus(h.error_rate_1h * 100, 1, 5) },
    { label: "Uptime", value: formatUptime(h.uptime_seconds), icon: Server, color: "#22c55e", status: "healthy" },
  ];

  return (
    <div>
      <PageHeader title="Operations Health" description="Real-time infrastructure and service monitoring" />

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
        {metrics.map((m) => {
          const Icon = m.icon;
          return (
            <div key={m.label} className="surface p-4 rounded-lg">
              <div className="flex items-center justify-between mb-2">
                <Icon className="w-4 h-4" style={{ color: m.color }} />
                <span className="w-2 h-2 rounded-full" style={{ background: statusColor(m.status) }} />
              </div>
              <p className="text-xs font-medium mb-1" style={{ color: "var(--text-tertiary)" }}>{m.label}</p>
              <p className="text-lg font-bold" style={{ color: "var(--text-primary)" }}>
                {m.value}{m.unit && <span className="text-xs font-normal ml-0.5" style={{ color: "var(--text-tertiary)" }}>{m.unit}</span>}
              </p>
            </div>
          );
        })}
      </div>

      <div className="surface p-5 rounded-lg">
        <h3 className="text-sm font-semibold mb-4" style={{ color: "var(--text-primary)" }}>Service Status</h3>
        <div className="space-y-2">
          {[
            { name: "API Server", status: "operational" },
            { name: "PostgreSQL", status: "operational" },
            { name: "Redis Cache", status: "operational" },
            { name: "WebSocket Hub", status: "operational" },
            { name: "Background Jobs", status: h.job_queue_depth > 50 ? "degraded" : "operational" },
            { name: "Payment Gateway", status: h.last_payment_at ? "operational" : "unknown" },
          ].map((svc) => (
            <div key={svc.name} className="flex items-center justify-between py-2 px-3 rounded-lg" style={{ background: "var(--bg-inset)" }}>
              <span className="text-sm" style={{ color: "var(--text-primary)" }}>{svc.name}</span>
              <span className="flex items-center gap-1.5 text-xs font-medium" style={{ color: svc.status === "operational" ? "var(--color-success)" : svc.status === "degraded" ? "var(--color-warning)" : "var(--text-tertiary)" }}>
                <span className="w-1.5 h-1.5 rounded-full" style={{ background: svc.status === "operational" ? "var(--color-success)" : svc.status === "degraded" ? "var(--color-warning)" : "var(--text-tertiary)" }} />
                {svc.status === "operational" ? "Operational" : svc.status === "degraded" ? "Degraded" : "Unknown"}
              </span>
            </div>
          ))}
        </div>
        <p className="text-xs mt-3" style={{ color: "var(--text-tertiary)" }}>Auto-refreshes every 30 seconds</p>
      </div>

      {/* Alerts */}
      {(h.job_queue_depth > 100 || h.error_rate_1h > 0.01) && (
        <div className="mt-4 p-4 rounded-lg flex items-center gap-3" style={{ background: "rgba(239,68,68,0.1)", border: "1px solid rgba(239,68,68,0.3)" }}>
          <AlertTriangle className="w-5 h-5 flex-shrink-0" style={{ color: "var(--color-danger)" }} />
          <div>
            <p className="text-sm font-medium" style={{ color: "var(--color-danger)" }}>System Alert</p>
            <p className="text-xs" style={{ color: "var(--text-secondary)" }}>
              {h.job_queue_depth > 100 && `Job queue depth (${h.job_queue_depth}) exceeds threshold. `}
              {h.error_rate_1h > 0.01 && `Error rate (${(h.error_rate_1h * 100).toFixed(2)}%) exceeds 1%.`}
            </p>
          </div>
        </div>
      )}

      {/* Manual Job Triggers */}
      <div className="surface p-5 rounded-lg mt-4">
        <h3 className="text-sm font-semibold mb-4" style={{ color: "var(--text-primary)" }}>Manual Job Triggers</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          {[
            { name: "expiry_check", label: "Run Expiry Check", description: "Check and expire overdue listings/auctions" },
            { name: "escrow_release", label: "Run Escrow Release", description: "Release held escrow for completed orders" },
            { name: "payout_process", label: "Process Payouts", description: "Execute pending seller payouts" },
          ].map((job) => (
            <button
              key={job.name}
              onClick={() => triggerJob(job.name)}
              disabled={triggering === job.name}
              className="flex items-start gap-3 p-3 rounded-lg text-left transition-colors hover:opacity-80"
              style={{ background: "var(--bg-inset)", border: "1px solid var(--border-default)" }}
            >
              <Play className="w-4 h-4 mt-0.5 flex-shrink-0" style={{ color: "var(--color-brand)" }} />
              <div>
                <p className="text-sm font-medium" style={{ color: "var(--text-primary)" }}>{job.label}</p>
                <p className="text-xs mt-0.5" style={{ color: "var(--text-tertiary)" }}>{job.description}</p>
                {triggering === job.name && <p className="text-xs mt-1 font-medium" style={{ color: "var(--color-brand)" }}>Running...</p>}
              </div>
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  if (days > 0) return `${days}d ${hours}h`;
  const mins = Math.floor((seconds % 3600) / 60);
  return `${hours}h ${mins}m`;
}
