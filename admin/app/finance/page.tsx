"use client";

import { useState } from "react";
import PageHeader from "@/components/shared/PageHeader";
import { Download, FileText, FileSpreadsheet } from "lucide-react";

export default function FinancePage() {
  const [from, setFrom] = useState(() => {
    const d = new Date();
    d.setMonth(d.getMonth() - 1);
    return d.toISOString().slice(0, 10);
  });
  const [to, setTo] = useState(() => new Date().toISOString().slice(0, 10));
  const [exporting, setExporting] = useState<string | null>(null);

  const handleExport = async (format: "csv" | "pdf") => {
    setExporting(format);
    try {
      const token = localStorage.getItem("admin_token");
      const res = await fetch(
        `/api/v1/admin/finance/report?format=${format}&from=${from}&to=${to}`,
        { headers: { Authorization: `Bearer ${token}` } }
      );
      if (!res.ok) throw new Error("Export failed");
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `finance_report_${from}_to_${to}.${format}`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch (err) {
      console.error("Export error:", err);
      alert("Export failed. Please try again.");
    } finally {
      setExporting(null);
    }
  };

  return (
    <div>
      <PageHeader
        title="Finance"
        description="Financial reports & export"
      />

      <div className="p-6 rounded-xl border border-slate-200 bg-white mb-6">
        <h3 className="font-semibold text-sm mb-4">Export Financial Report</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
          <div>
            <label className="block text-xs text-slate-500 mb-1">From</label>
            <input
              type="date"
              value={from}
              onChange={(e) => setFrom(e.target.value)}
              className="border border-slate-200 rounded-lg px-3 py-2 text-sm w-full"
            />
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">To</label>
            <input
              type="date"
              value={to}
              onChange={(e) => setTo(e.target.value)}
              className="border border-slate-200 rounded-lg px-3 py-2 text-sm w-full"
            />
          </div>
        </div>
        <p className="text-xs text-slate-400 mb-4">
          Report includes: Revenue, Fees Collected, Refunds, Escrow Held, Payouts
        </p>
        <div className="flex gap-3">
          <button
            onClick={() => handleExport("csv")}
            disabled={!!exporting}
            className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium border border-slate-200 hover:bg-slate-50 disabled:opacity-50"
          >
            <FileSpreadsheet className="w-4 h-4" />
            {exporting === "csv" ? "Exporting..." : "Export CSV"}
          </button>
          <button
            onClick={() => handleExport("pdf")}
            disabled={!!exporting}
            className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium text-white disabled:opacity-50"
            style={{ background: "var(--color-brand)" }}
          >
            <FileText className="w-4 h-4" />
            {exporting === "pdf" ? "Exporting..." : "Export PDF"}
          </button>
        </div>
      </div>
    </div>
  );
}
