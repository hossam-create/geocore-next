"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import axios from "axios";
import { Shield, AlertTriangle, CheckCircle, Eye, XCircle, BarChart3 } from "lucide-react";

interface FraudAlert {
  id: string;
  target_type: string;
  target_id: string;
  alert_type: string;
  severity: string;
  risk_score: number;
  detected_by: string;
  status: string;
  resolution: string;
  created_at: string;
}

interface FraudRule {
  id: string;
  name: string;
  description: string;
  rule_type: string;
  severity: string;
  is_active: boolean;
}

interface Stats {
  total: number;
  last_24h: number;
  by_status: { status: string; count: number }[];
  by_severity: { severity: string; count: number }[];
}

const sevColor: Record<string, string> = {
  low: "bg-blue-100 text-blue-700",
  medium: "bg-yellow-100 text-yellow-700",
  high: "bg-orange-100 text-orange-700",
  critical: "bg-red-100 text-red-700",
};

const statusColor: Record<string, string> = {
  pending: "bg-orange-100 text-orange-700",
  investigating: "bg-blue-100 text-blue-700",
  confirmed: "bg-red-100 text-red-700",
  false_positive: "bg-gray-100 text-gray-600",
  resolved: "bg-green-100 text-green-700",
};

export default function FraudDashboardPage() {
  const [tab, setTab] = useState<"alerts" | "rules">("alerts");
  const [statusFilter, setStatusFilter] = useState("pending");
  const qc = useQueryClient();

  const { data: stats } = useQuery<Stats>({
    queryKey: ["fraud-stats"],
    queryFn: async () => {
      const { data } = await axios.get("/api/v1/admin/fraud/stats");
      return data.data;
    },
  });

  const { data: alerts = [] } = useQuery<FraudAlert[]>({
    queryKey: ["fraud-alerts", statusFilter],
    queryFn: async () => {
      const params = statusFilter ? `?status=${statusFilter}` : "";
      const { data } = await axios.get(`/api/v1/admin/fraud/alerts${params}`);
      return data.data ?? [];
    },
  });

  const { data: rules = [] } = useQuery<FraudRule[]>({
    queryKey: ["fraud-rules"],
    queryFn: async () => {
      const { data } = await axios.get("/api/v1/admin/fraud/rules");
      return data.data ?? [];
    },
    enabled: tab === "rules",
  });

  const updateAlert = useMutation({
    mutationFn: ({ id, status, resolution }: { id: string; status: string; resolution?: string }) =>
      axios.patch(`/api/v1/admin/fraud/alerts/${id}`, { status, resolution }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["fraud-alerts"] });
      qc.invalidateQueries({ queryKey: ["fraud-stats"] });
    },
  });

  const toggleRule = useMutation({
    mutationFn: ({ id, is_active }: { id: string; is_active: boolean }) =>
      axios.patch(`/api/v1/admin/fraud/rules/${id}`, { is_active }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["fraud-rules"] }),
  });

  return (
    <div className="max-w-6xl mx-auto px-4 py-8">
      <div className="flex items-center gap-3 mb-8">
        <Shield className="w-8 h-8 text-red-600" />
        <div>
          <h1 className="text-3xl font-bold">Fraud Detection</h1>
          <p className="text-gray-500">Monitor alerts, manage rules, and review risk profiles</p>
        </div>
      </div>

      {/* Stats */}
      {stats && (
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-8">
          <div className="bg-white rounded-xl border p-5">
            <div className="flex items-center gap-2 mb-1">
              <BarChart3 className="w-4 h-4 text-gray-400" />
              <span className="text-sm text-gray-500">Total Alerts</span>
            </div>
            <p className="text-2xl font-bold">{stats.total}</p>
          </div>
          <div className="bg-white rounded-xl border p-5">
            <div className="flex items-center gap-2 mb-1">
              <AlertTriangle className="w-4 h-4 text-orange-500" />
              <span className="text-sm text-gray-500">Last 24h</span>
            </div>
            <p className="text-2xl font-bold">{stats.last_24h}</p>
          </div>
          <div className="bg-white rounded-xl border p-5">
            <div className="flex items-center gap-2 mb-1">
              <Eye className="w-4 h-4 text-blue-500" />
              <span className="text-sm text-gray-500">Pending</span>
            </div>
            <p className="text-2xl font-bold">
              {stats.by_status?.find((s) => s.status === "pending")?.count ?? 0}
            </p>
          </div>
          <div className="bg-white rounded-xl border p-5">
            <div className="flex items-center gap-2 mb-1">
              <XCircle className="w-4 h-4 text-red-500" />
              <span className="text-sm text-gray-500">Confirmed</span>
            </div>
            <p className="text-2xl font-bold">
              {stats.by_status?.find((s) => s.status === "confirmed")?.count ?? 0}
            </p>
          </div>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 mb-6 bg-gray-100 rounded-lg p-1 w-fit">
        <button onClick={() => setTab("alerts")} className={`px-4 py-2 rounded-md text-sm font-medium transition ${tab === "alerts" ? "bg-white shadow text-red-600" : "text-gray-600"}`}>
          Alerts
        </button>
        <button onClick={() => setTab("rules")} className={`px-4 py-2 rounded-md text-sm font-medium transition ${tab === "rules" ? "bg-white shadow text-red-600" : "text-gray-600"}`}>
          Rules
        </button>
      </div>

      {/* Alerts Tab */}
      {tab === "alerts" && (
        <>
          <div className="flex gap-2 mb-4 flex-wrap">
            {["pending", "investigating", "confirmed", "false_positive", "resolved", ""].map((s) => (
              <button
                key={s}
                onClick={() => setStatusFilter(s)}
                className={`text-xs px-3 py-1.5 rounded-full font-medium transition ${statusFilter === s ? "bg-red-600 text-white" : "bg-gray-100 text-gray-600 hover:bg-gray-200"}`}
              >
                {s || "All"}
              </button>
            ))}
          </div>

          {alerts.length === 0 && (
            <div className="text-center py-16 text-gray-400">
              <CheckCircle className="w-12 h-12 mx-auto mb-3 opacity-40" />
              <p>No alerts matching filter</p>
            </div>
          )}

          <div className="space-y-3">
            {alerts.map((alert) => (
              <div key={alert.id} className="bg-white rounded-xl border p-5">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-3">
                    <span className={`text-xs px-2 py-1 rounded-full font-medium ${sevColor[alert.severity] ?? "bg-gray-100"}`}>
                      {alert.severity}
                    </span>
                    <span className="font-medium text-sm">{alert.alert_type}</span>
                    <span className="text-xs text-gray-400">{alert.target_type}:{alert.target_id.slice(0, 8)}</span>
                  </div>
                  <span className={`text-xs px-2 py-1 rounded-full font-medium ${statusColor[alert.status] ?? "bg-gray-100"}`}>
                    {alert.status}
                  </span>
                </div>
                <div className="flex items-center gap-4 text-sm text-gray-500 mb-3">
                  <span>Score: <strong className="text-gray-800">{alert.risk_score.toFixed(1)}</strong>/100</span>
                  <span>By: {alert.detected_by}</span>
                  <span>{new Date(alert.created_at).toLocaleString()}</span>
                </div>
                {alert.status === "pending" && (
                  <div className="flex gap-2">
                    <button onClick={() => updateAlert.mutate({ id: alert.id, status: "investigating" })} className="text-xs bg-blue-50 text-blue-600 px-3 py-1 rounded-lg hover:bg-blue-100">
                      Investigate
                    </button>
                    <button onClick={() => updateAlert.mutate({ id: alert.id, status: "confirmed", resolution: "Confirmed fraud" })} className="text-xs bg-red-50 text-red-600 px-3 py-1 rounded-lg hover:bg-red-100">
                      Confirm Fraud
                    </button>
                    <button onClick={() => updateAlert.mutate({ id: alert.id, status: "false_positive", resolution: "False positive" })} className="text-xs bg-gray-50 text-gray-600 px-3 py-1 rounded-lg hover:bg-gray-100">
                      False Positive
                    </button>
                  </div>
                )}
                {alert.status === "investigating" && (
                  <div className="flex gap-2">
                    <button onClick={() => updateAlert.mutate({ id: alert.id, status: "confirmed", resolution: "Confirmed after investigation" })} className="text-xs bg-red-50 text-red-600 px-3 py-1 rounded-lg hover:bg-red-100">
                      Confirm Fraud
                    </button>
                    <button onClick={() => updateAlert.mutate({ id: alert.id, status: "resolved", resolution: "Resolved" })} className="text-xs bg-green-50 text-green-600 px-3 py-1 rounded-lg hover:bg-green-100">
                      Resolve
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>
        </>
      )}

      {/* Rules Tab */}
      {tab === "rules" && (
        <div className="space-y-3">
          {rules.map((rule) => (
            <div key={rule.id} className="bg-white rounded-xl border p-5 flex items-center justify-between">
              <div>
                <div className="flex items-center gap-3 mb-1">
                  <span className="font-medium text-sm">{rule.name}</span>
                  <span className={`text-xs px-2 py-0.5 rounded-full ${sevColor[rule.severity] ?? "bg-gray-100"}`}>{rule.severity}</span>
                  <span className="text-xs text-gray-400">{rule.rule_type}</span>
                </div>
                {rule.description && <p className="text-sm text-gray-500">{rule.description}</p>}
              </div>
              <button
                onClick={() => toggleRule.mutate({ id: rule.id, is_active: !rule.is_active })}
                className={`relative w-12 h-6 rounded-full transition ${rule.is_active ? "bg-green-500" : "bg-gray-300"}`}
              >
                <span className={`absolute top-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform ${rule.is_active ? "left-6" : "left-0.5"}`} />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
