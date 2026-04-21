"use client";

import PageHeader from "@/components/shared/PageHeader";

const CATEGORIES = ["Electronics", "Vehicles", "Fashion", "Collectibles", "Home", "Sports"];

export default function CategoriesPage() {
  return (
    <div>
      <PageHeader title="Categories" description="Manage marketplace taxonomy" />
      <div className="surface p-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {CATEGORIES.map((c) => (
            <div key={c} className="px-3 py-2 rounded-lg flex items-center justify-between" style={{ background: "var(--bg-surface-active)" }}>
              <span style={{ color: "var(--text-primary)" }}>{c}</span>
              <button className="text-xs" style={{ color: "var(--color-danger)" }}>Delete</button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
