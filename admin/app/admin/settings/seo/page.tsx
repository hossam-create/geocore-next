"use client";

export default function AdminSeoSettingsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">SEO Settings</h1>
        <p className="text-sm text-slate-500 mt-0.5">Meta tags, analytics, robots.txt, sitemap</p>
      </div>
      <div className="bg-white rounded-xl border border-slate-200 p-8 text-center text-sm text-slate-400">
        SEO settings will be available after adding an &quot;seo&quot; category to admin_settings.
        <br />Insert rows with category=&quot;seo&quot; and they will appear here automatically.
      </div>
    </div>
  );
}
