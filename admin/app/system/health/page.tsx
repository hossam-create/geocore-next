"use client";

import { useQuery } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";

const CHECKS = [
  { name: "API", status: "healthy", latency: "44ms" },
  { name: "Database", status: "healthy", latency: "12ms" },
  { name: "Redis", status: "healthy", latency: "3ms" },
  { name: "Queue Worker", status: "degraded", latency: "—" },
];

export default function HealthPage() {
  const { data: externalSignal, isLoading: externalLoading } = useQuery({
    queryKey: ["system", "external-signal"],
    queryFn: async () => {
      // Open-Meteo is listed in public-apis/public-apis and does not require an API key.
      const res = await fetch(
        "https://api.open-meteo.com/v1/forecast?latitude=30.0444&longitude=31.2357&current=temperature_2m,wind_speed_10m"
      );
      if (!res.ok) throw new Error("External API unavailable");
      return (await res.json()) as {
        current?: { temperature_2m?: number; wind_speed_10m?: number };
      };
    },
    retry: 1,
    staleTime: 300_000,
  });

  const externalCard = {
    name: "External Signal (Open-Meteo)",
    status: externalLoading ? "loading" : externalSignal ? "healthy" : "degraded",
    latency: externalLoading
      ? "checking..."
      : externalSignal?.current
      ? `${externalSignal.current.temperature_2m ?? "-"}°C / ${externalSignal.current.wind_speed_10m ?? "-"}km/h`
      : "unavailable",
  };

  const checks = [...CHECKS, externalCard];

  return (
    <div>
      <PageHeader title="System Health" description="Infrastructure checks and runtime status" />
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {checks.map((c) => (
          <div key={c.name} className="surface p-4">
            <p className="font-medium" style={{ color: "var(--text-primary)" }}>{c.name}</p>
            <p className="text-sm mt-1" style={{ color: c.status === "healthy" ? "var(--color-success)" : c.status === "loading" ? "var(--color-brand)" : "var(--color-warning)" }}>{c.status}</p>
            <p className="text-xs mt-1" style={{ color: "var(--text-tertiary)" }}>Latency: {c.latency}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
