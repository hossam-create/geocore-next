"use client";

import Link from "next/link";
import PageHeader from "@/components/shared/PageHeader";

const SECTIONS = [
  { href: "/admin/settings/general", title: "General", desc: "Site identity, locale, defaults" },
  { href: "/admin/settings/listings", title: "Listings", desc: "Publishing, moderation, limits" },
  { href: "/admin/settings/auctions", title: "Auctions", desc: "Durations, extensions, rules" },
  { href: "/admin/settings/search", title: "Search Ranking", desc: "Cassini-style weights, banned keywords" },
  { href: "/admin/settings/pricing", title: "Dynamic Pricing", desc: "Fee schedule, live calculator, overrides" },
  { href: "/admin/settings/shipping", title: "Shipping", desc: "Carrier on/off, credentials, rate testing" },
  { href: "/admin/settings/payments", title: "Payments", desc: "Gateways, escrow, fees" },
  { href: "/admin/settings/email", title: "Email", desc: "SMTP, templates, alerts" },
  { href: "/admin/settings/trust", title: "Trust & Safety", desc: "Auto-ban, fraud, verification rules" },
  { href: "/admin/settings/aws", title: "AWS", desc: "CloudWatch, S3, CloudFront" },
  { href: "/admin/settings/seo", title: "SEO", desc: "Meta defaults and indexing" },
  { href: "/admin/settings/providers", title: "External Providers", desc: "OAuth, storage, maps, SMS, push" },
];

export default function AdminSettingsPage() {
  return (
    <div>
      <PageHeader title="Settings" description="System-wide configuration hubs" />
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {SECTIONS.map((s) => (
          <Link key={s.href} href={s.href} className="surface p-4 surface-hover transition-all">
            <p className="font-semibold" style={{ color: "var(--text-primary)" }}>{s.title}</p>
            <p className="text-sm mt-1" style={{ color: "var(--text-tertiary)" }}>{s.desc}</p>
          </Link>
        ))}
      </div>
    </div>
  );
}
