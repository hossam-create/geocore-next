"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { emailTemplatesApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import { Mail, Pencil, X, Check, Send, Eye } from "lucide-react";

interface EmailTemplate {
  id: number;
  slug: string;
  event_type: string;
  name: string;
  subject: string;
  body_html: string;
  body_text: string;
  variables: string;
  is_active: boolean;
  updated_by: string;
}

interface PreviewData {
  subject: string;
  body_html: string;
  variables: Record<string, string>;
}

export default function EmailTemplatesPage() {
  const qc = useQueryClient();
  const [editing, setEditing] = useState<EmailTemplate | null>(null);
  const [testEmail, setTestEmail] = useState("");
  const [preview, setPreview] = useState<PreviewData | null>(null);

  const { data = [], isLoading } = useQuery({ queryKey: ["email-templates"], queryFn: emailTemplatesApi.list });

  const updateMut = useMutation({
    mutationFn: ({ slug, ...d }: { slug: string } & Record<string, unknown>) => emailTemplatesApi.update(slug, d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["email-templates"] }); setEditing(null); },
  });

  const testMut = useMutation({
    mutationFn: ({ slug, email }: { slug: string; email: string }) => emailTemplatesApi.test(slug, email),
    onSuccess: () => { setTestEmail(""); alert("Test email queued!"); },
  });

  const previewMut = useMutation({
    mutationFn: (slug: string) => emailTemplatesApi.preview(slug),
    onSuccess: (data) => { setPreview(data?.data ?? data ?? null); },
  });

  const templates: EmailTemplate[] = Array.isArray(data) ? data : [];

  return (
    <div>
      <PageHeader title="Email Templates" description="Manage transactional email templates" />

      {/* Preview Modal */}
      {preview && (
        <div className="p-4 mb-4 rounded-xl border border-slate-200 bg-white">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold text-sm">Preview</h3>
            <button onClick={() => setPreview(null)}><X className="w-4 h-4 text-slate-400" /></button>
          </div>
          <div className="mb-2 text-xs text-slate-500"><strong>Subject:</strong> {preview.subject}</div>
          <div className="border rounded-lg p-3 bg-slate-50 text-sm overflow-auto max-h-64" dangerouslySetInnerHTML={{ __html: preview.body_html }} />
          {preview.variables && Object.keys(preview.variables).length > 0 && (
            <div className="mt-2 text-xs text-slate-400">Sample data: {Object.entries(preview.variables).map(([k, v]) => `${k}=${v}`).join(", ")}</div>
          )}
        </div>
      )}

      {editing && (
        <div className="p-4 mb-4 rounded-xl border border-slate-200 bg-white">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold text-sm">Edit: {editing.name}</h3>
            <button onClick={() => setEditing(null)}><X className="w-4 h-4 text-slate-400" /></button>
          </div>
          <div className="space-y-3">
            <input placeholder="Subject" value={editing.subject} onChange={(e) => setEditing({ ...editing, subject: e.target.value })} className="w-full border rounded-lg px-3 py-2 text-sm" />
            <textarea rows={10} placeholder="HTML Body" value={editing.body_html} onChange={(e) => setEditing({ ...editing, body_html: e.target.value })} className="w-full border rounded-lg px-3 py-2 text-sm font-mono" />
            <div className="text-xs text-slate-400">Variables: {editing.variables}</div>
            <div className="flex items-center gap-2">
              <input placeholder="Test email address" value={testEmail} onChange={(e) => setTestEmail(e.target.value)} className="border rounded-lg px-3 py-2 text-sm flex-1" />
              <button onClick={() => testMut.mutate({ slug: editing.slug, email: testEmail })} disabled={!testEmail} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white disabled:opacity-50" style={{ background: "var(--color-info)" }}><Send className="w-3.5 h-3.5" /> Test</button>
            </div>
          </div>
          <div className="mt-3 flex gap-2">
            <button onClick={() => updateMut.mutate({ slug: editing.slug, subject: editing.subject, body_html: editing.body_html })} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Check className="w-4 h-4" /> Save</button>
            <button onClick={() => setEditing(null)} className="px-3 py-1.5 rounded-lg text-sm border">Cancel</button>
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="text-center py-12 text-sm text-slate-400">Loading templates...</div>
      ) : (
        <div className="grid gap-3">
          {templates.map((t) => (
            <div key={t.slug} className="flex items-center justify-between p-4 rounded-xl border border-slate-200 bg-white hover:border-slate-300 transition-colors">
              <div className="flex items-center gap-3">
                <div className="w-9 h-9 rounded-lg flex items-center justify-center" style={{ background: "var(--color-brand-light)" }}>
                  <Mail className="w-4 h-4" style={{ color: "var(--color-brand)" }} />
                </div>
                <div>
                  <p className="text-sm font-medium text-slate-800">{t.name}</p>
                  <p className="text-xs text-slate-400">{t.slug} — {t.subject}</p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <StatusBadge status={t.is_active ? "active" : "inactive"} />
                <button onClick={() => previewMut.mutate(t.slug)} disabled={previewMut.isPending} className="p-1.5 hover:bg-slate-100 rounded" title="Preview"><Eye className="w-3.5 h-3.5 text-slate-500" /></button>
                <button onClick={() => setEditing(t)} className="p-1.5 hover:bg-slate-100 rounded"><Pencil className="w-3.5 h-3.5 text-slate-500" /></button>
              </div>
            </div>
          ))}
          {templates.length === 0 && <p className="text-sm text-slate-400 text-center py-8">No email templates found.</p>}
        </div>
      )}
    </div>
  );
}
