"use client";

import { useState, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { cmsApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import { Image, Type, Settings, Menu, Upload, Plus, Pencil, X, Trash2, GripVertical, Save, ExternalLink } from "lucide-react";

type Tab = "slides" | "blocks" | "media" | "settings" | "nav";

interface HeroSlide {
  id: string; title: string; subtitle: string; image_url: string;
  link_url: string; link_label: string; badge: string;
  position: number; is_active: boolean; start_date?: string; end_date?: string;
}

interface ContentBlock {
  id: string; slug: string; title: string; type: string; content: string;
  content2: string; image_url: string; link_url: string; metadata: string;
  position: number; is_active: boolean; page: string; section: string;
}

interface MediaFile {
  id: string; file_name: string; url: string; mime_type: string;
  size_bytes: number; type: string; alt: string; folder: string; created_at: string;
}

interface SiteSetting {
  id: string; key: string; value: string; group: string; label: string; type: string;
}

interface NavItem {
  id: string; location: string; label: string; url: string; icon: string;
  parent_id: string | null; position: number; is_external: boolean; is_active: boolean;
}

const TABS: { key: Tab; label: string; icon: typeof Image }[] = [
  { key: "slides", label: "Hero Slider", icon: Image },
  { key: "blocks", label: "Content Blocks", icon: Type },
  { key: "media", label: "Media Library", icon: Upload },
  { key: "settings", label: "Site Settings", icon: Settings },
  { key: "nav", label: "Navigation", icon: Menu },
];

export default function CMSPage() {
  const qc = useQueryClient();
  const [tab, setTab] = useState<Tab>("slides");
  const [editing, setEditing] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null!);

  return (
    <div>
      <PageHeader title="Content Management" description="Manage your site without code — sliders, content, media, settings, navigation" />

      {/* Tab Bar */}
      <div className="flex gap-1 mb-6 p-1 bg-slate-100 rounded-xl">
        {TABS.map((t) => (
          <button key={t.key} onClick={() => { setTab(t.key); setEditing(null); }}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${tab === t.key ? "bg-white shadow text-slate-800" : "text-slate-500 hover:text-slate-700"}`}>
            <t.icon className="w-4 h-4" /> {t.label}
          </button>
        ))}
      </div>

      {tab === "slides" && <SlidesTab qc={qc} editing={editing} setEditing={setEditing} />}
      {tab === "blocks" && <BlocksTab qc={qc} editing={editing} setEditing={setEditing} />}
      {tab === "media" && <MediaTab qc={qc} fileRef={fileRef} />}
      {tab === "settings" && <SettingsTab qc={qc} />}
      {tab === "nav" && <NavTab qc={qc} editing={editing} setEditing={setEditing} />}
    </div>
  );
}

// ════════════════════════════════════════════════════════════════════════════
// Hero Slides Tab
// ════════════════════════════════════════════════════════════════════════════

function SlidesTab({ qc, editing, setEditing }: { qc: QueryClient; editing: string | null; setEditing: (v: string | null) => void }) {
  const [form, setForm] = useState<Partial<HeroSlide>>({});

  const { data = [] } = useQuery({ queryKey: ["cms-slides"], queryFn: cmsApi.slides.list });
  const slides: HeroSlide[] = Array.isArray(data) ? data : [];

  const createMut = useMutation({
    mutationFn: (d: Record<string, unknown>) => cmsApi.slides.create(d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["cms-slides"] }); setForm({}); },
  });
  const updateMut = useMutation({
    mutationFn: ({ id, ...d }: { id: string } & Record<string, unknown>) => cmsApi.slides.update(id, d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["cms-slides"] }); setEditing(null); },
  });
  const deleteMut = useMutation({
    mutationFn: cmsApi.slides.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cms-slides"] }),
  });

  const saveForm = () => {
    if (editing) {
      updateMut.mutate({ id: editing, ...form });
    } else {
      createMut.mutate(form as Record<string, unknown>);
    }
  };

  return (
    <div>
      {/* Form */}
      <div className="p-4 mb-4 rounded-xl border border-slate-200 bg-white">
        <div className="flex items-center justify-between mb-3">
          <h3 className="font-semibold text-sm">{editing ? "Edit Slide" : "Add New Slide"}</h3>
          {editing && <button onClick={() => { setEditing(null); setForm({}); }}><X className="w-4 h-4 text-slate-400" /></button>}
        </div>
        <div className="grid grid-cols-2 gap-3">
          <input placeholder="Title" value={form.title ?? ""} onChange={(e) => setForm({ ...form, title: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
          <input placeholder="Subtitle" value={form.subtitle ?? ""} onChange={(e) => setForm({ ...form, subtitle: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
          <input placeholder="Image URL" value={form.image_url ?? ""} onChange={(e) => setForm({ ...form, image_url: e.target.value })} className="border rounded-lg px-3 py-2 text-sm col-span-2" />
          <input placeholder="Link URL" value={form.link_url ?? ""} onChange={(e) => setForm({ ...form, link_url: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
          <input placeholder="Link Label" value={form.link_label ?? ""} onChange={(e) => setForm({ ...form, link_label: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
          <input placeholder="Badge (SALE, NEW, HOT)" value={form.badge ?? ""} onChange={(e) => setForm({ ...form, badge: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
          <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={form.is_active ?? true} onChange={(e) => setForm({ ...form, is_active: e.target.checked })} /> Active</label>
        </div>
        <div className="mt-3 flex gap-2">
          <button onClick={saveForm} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Save className="w-4 h-4" /> {editing ? "Update" : "Create"}</button>
        </div>
      </div>

      {/* List */}
      <div className="grid gap-3">
        {slides.map((s) => (
          <div key={s.id} className="flex items-center gap-4 p-4 rounded-xl border border-slate-200 bg-white">
            <GripVertical className="w-4 h-4 text-slate-300 cursor-grab" />
            {s.image_url && <img src={s.image_url} alt={s.title} className="w-24 h-14 object-cover rounded-lg" />}
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <p className="text-sm font-medium text-slate-800 truncate">{s.title || "Untitled"}</p>
                {s.badge && <span className="px-1.5 py-0.5 rounded text-xs font-bold bg-amber-100 text-amber-700">{s.badge}</span>}
              </div>
              <p className="text-xs text-slate-400 truncate">{s.subtitle}</p>
            </div>
            <StatusBadge status={s.is_active ? "active" : "inactive"} />
            <button onClick={() => { setEditing(s.id); setForm(s); }} className="p-1.5 hover:bg-slate-100 rounded"><Pencil className="w-3.5 h-3.5 text-slate-500" /></button>
            <button onClick={() => deleteMut.mutate(s.id)} className="p-1.5 hover:bg-red-50 rounded"><Trash2 className="w-3.5 h-3.5 text-red-400" /></button>
          </div>
        ))}
      </div>
    </div>
  );
}

// ════════════════════════════════════════════════════════════════════════════
// Content Blocks Tab
// ════════════════════════════════════════════════════════════════════════════

function BlocksTab({ qc, editing, setEditing }: { qc: QueryClient; editing: string | null; setEditing: (v: string | null) => void }) {
  const [form, setForm] = useState<Partial<ContentBlock>>({});

  const { data = [] } = useQuery({ queryKey: ["cms-blocks"], queryFn: () => cmsApi.blocks.list() });
  const blocks: ContentBlock[] = Array.isArray(data) ? data : [];

  const createMut = useMutation({
    mutationFn: (d: Record<string, unknown>) => cmsApi.blocks.create(d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["cms-blocks"] }); setForm({}); },
  });
  const updateMut = useMutation({
    mutationFn: ({ slug, ...d }: { slug: string } & Record<string, unknown>) => cmsApi.blocks.update(slug, d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["cms-blocks"] }); setEditing(null); },
  });
  const deleteMut = useMutation({
    mutationFn: cmsApi.blocks.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cms-blocks"] }),
  });

  const saveForm = () => {
    if (editing) {
      updateMut.mutate({ slug: form.slug ?? editing, ...form });
    } else {
      createMut.mutate(form as Record<string, unknown>);
    }
  };

  const TYPE_COLORS: Record<string, string> = {
    html: "bg-slate-100 text-slate-600", hero: "bg-purple-100 text-purple-700",
    cta: "bg-green-100 text-green-700", faq: "bg-blue-100 text-blue-700",
    features: "bg-indigo-100 text-indigo-700", testimonial: "bg-pink-100 text-pink-700",
    markdown: "bg-amber-100 text-amber-700", image: "bg-cyan-100 text-cyan-700",
  };

  return (
    <div>
      <div className="p-4 mb-4 rounded-xl border border-slate-200 bg-white">
        <div className="flex items-center justify-between mb-3">
          <h3 className="font-semibold text-sm">{editing ? `Edit: ${form.title}` : "Add Content Block"}</h3>
          {editing && <button onClick={() => { setEditing(null); setForm({}); }}><X className="w-4 h-4 text-slate-400" /></button>}
        </div>
        <div className="grid grid-cols-3 gap-3">
          <input placeholder="Slug (e.g. homepage_hero)" value={form.slug ?? ""} onChange={(e) => setForm({ ...form, slug: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" disabled={!!editing} />
          <input placeholder="Title" value={form.title ?? ""} onChange={(e) => setForm({ ...form, title: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
          <select value={form.type ?? "html"} onChange={(e) => setForm({ ...form, type: e.target.value })} className="border rounded-lg px-3 py-2 text-sm">
            <option value="html">HTML</option><option value="hero">Hero</option><option value="cta">CTA</option>
            <option value="features">Features</option><option value="faq">FAQ</option><option value="testimonial">Testimonial</option>
            <option value="markdown">Markdown</option><option value="image">Image</option>
          </select>
          <input placeholder="Page (home, about, etc.)" value={form.page ?? ""} onChange={(e) => setForm({ ...form, page: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
          <input placeholder="Section (hero, features, footer)" value={form.section ?? ""} onChange={(e) => setForm({ ...form, section: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
          <input placeholder="Image URL" value={form.image_url ?? ""} onChange={(e) => setForm({ ...form, image_url: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
        </div>
        <textarea rows={6} placeholder="Content (HTML, JSON, or text depending on type)" value={form.content ?? ""} onChange={(e) => setForm({ ...form, content: e.target.value })} className="w-full mt-3 border rounded-lg px-3 py-2 text-sm font-mono" />
        <div className="mt-3 flex gap-2">
          <button onClick={saveForm} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Save className="w-4 h-4" /> {editing ? "Update" : "Create"}</button>
        </div>
      </div>

      <div className="grid gap-3">
        {blocks.map((b) => (
          <div key={b.id} className="flex items-center gap-3 p-4 rounded-xl border border-slate-200 bg-white">
            <span className={`px-2 py-0.5 rounded text-xs font-medium ${TYPE_COLORS[b.type] ?? "bg-slate-100 text-slate-600"}`}>{b.type}</span>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-slate-800 truncate">{b.title}</p>
              <p className="text-xs text-slate-400">{b.slug} · {b.page}/{b.section}</p>
            </div>
            <StatusBadge status={b.is_active ? "active" : "inactive"} />
            <button onClick={() => { setEditing(b.id); setForm(b); }} className="p-1.5 hover:bg-slate-100 rounded"><Pencil className="w-3.5 h-3.5 text-slate-500" /></button>
            <button onClick={() => deleteMut.mutate(b.slug)} className="p-1.5 hover:bg-red-50 rounded"><Trash2 className="w-3.5 h-3.5 text-red-400" /></button>
          </div>
        ))}
      </div>
    </div>
  );
}

// ════════════════════════════════════════════════════════════════════════════
// Media Library Tab
// ════════════════════════════════════════════════════════════════════════════

function MediaTab({ qc, fileRef }: { qc: QueryClient; fileRef: React.RefObject<HTMLInputElement | null> }) {
  const [folder, setFolder] = useState("");

  const { data = [] } = useQuery({ queryKey: ["cms-media", folder], queryFn: () => cmsApi.media.list(folder ? { folder } : undefined) });
  const files: MediaFile[] = Array.isArray(data) ? data : [];

  const uploadMut = useMutation({
    mutationFn: (fd: FormData) => cmsApi.media.upload(fd),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["cms-media"] }); if (fileRef.current) fileRef.current.value = ""; },
  });
  const deleteMut = useMutation({
    mutationFn: cmsApi.media.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cms-media"] }),
  });

  const handleUpload = () => {
    const file = fileRef.current?.files?.[0];
    if (!file) return;
    const fd = new FormData();
    fd.append("file", file);
    if (folder) fd.append("folder", folder);
    uploadMut.mutate(fd);
  };

  const formatSize = (bytes: number) => bytes > 1024 * 1024 ? `${(bytes / 1024 / 1024).toFixed(1)} MB` : `${(bytes / 1024).toFixed(0)} KB`;

  return (
    <div>
      {/* Upload */}
      <div className="p-4 mb-4 rounded-xl border border-slate-200 bg-white">
        <div className="flex items-center gap-3">
          <input ref={fileRef} type="file" accept="image/*,video/*,.pdf,.doc,.docx" className="flex-1 text-sm" />
          <input placeholder="Folder" value={folder} onChange={(e) => setFolder(e.target.value)} className="border rounded-lg px-3 py-2 text-sm w-32" />
          <button onClick={handleUpload} disabled={uploadMut.isPending} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white disabled:opacity-50" style={{ background: "var(--color-brand)" }}>
            <Upload className="w-3.5 h-3.5" /> Upload
          </button>
        </div>
      </div>

      {/* Grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-3">
        {files.map((f) => (
          <div key={f.id} className="group relative rounded-xl border border-slate-200 bg-white overflow-hidden">
            {f.type === "image" ? (
              <img src={f.url} alt={f.alt || f.file_name} className="w-full h-32 object-cover" />
            ) : (
              <div className="w-full h-32 flex items-center justify-center bg-slate-50">
                <Type className="w-8 h-8 text-slate-300" />
              </div>
            )}
            <div className="p-2">
              <p className="text-xs text-slate-700 truncate">{f.file_name}</p>
              <p className="text-xs text-slate-400">{formatSize(f.size_bytes)}</p>
            </div>
            <div className="absolute top-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity">
              <button onClick={() => deleteMut.mutate(f.id)} className="p-1 bg-red-500 text-white rounded"><Trash2 className="w-3 h-3" /></button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// ════════════════════════════════════════════════════════════════════════════
// Site Settings Tab
// ════════════════════════════════════════════════════════════════════════════

function SettingsTab({ qc }: { qc: QueryClient }) {
  const [group, setGroup] = useState("branding");
  const [edits, setEdits] = useState<Record<string, string>>({});

  const { data = [] } = useQuery({ queryKey: ["cms-settings", group], queryFn: () => cmsApi.settings.list({ group }) });
  const settings: SiteSetting[] = Array.isArray(data) ? data : [];

  const bulkMut = useMutation({
    mutationFn: (data: Record<string, string>) => cmsApi.settings.bulkUpdate(data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["cms-settings"] }); setEdits({}); alert("Settings saved!"); },
  });

  const saveAll = () => {
    if (Object.keys(edits).length === 0) return;
    bulkMut.mutate(edits);
  };

  const GROUPS = [
    { key: "branding", label: "🎨 Branding" },
    { key: "contact", label: "📞 Contact" },
    { key: "social", label: "🌐 Social Media" },
    { key: "seo", label: "🔍 SEO" },
    { key: "general", label: "⚙️ General" },
  ];

  const renderInput = (s: SiteSetting) => {
    const val = edits[s.key] ?? s.value;
    const onChange = (v: string) => setEdits({ ...edits, [s.key]: v });

    switch (s.type) {
      case "color":
        return <div className="flex items-center gap-2"><input type="color" value={val} onChange={(e) => onChange(e.target.value)} className="w-10 h-10 rounded cursor-pointer" /><input value={val} onChange={(e) => onChange(e.target.value)} className="border rounded-lg px-3 py-2 text-sm flex-1" /></div>;
      case "textarea":
        return <textarea value={val} onChange={(e) => onChange(e.target.value)} rows={3} className="w-full border rounded-lg px-3 py-2 text-sm" />;
      case "boolean":
        return <select value={val} onChange={(e) => onChange(e.target.value)} className="border rounded-lg px-3 py-2 text-sm w-full"><option value="true">Yes</option><option value="false">No</option></select>;
      case "image":
        return <input value={val} onChange={(e) => onChange(e.target.value)} placeholder="Image URL" className="w-full border rounded-lg px-3 py-2 text-sm" />;
      case "url":
      case "email":
        return <input type={s.type} value={val} onChange={(e) => onChange(e.target.value)} className="w-full border rounded-lg px-3 py-2 text-sm" />;
      default:
        return <input value={val} onChange={(e) => onChange(e.target.value)} className="w-full border rounded-lg px-3 py-2 text-sm" />;
    }
  };

  return (
    <div>
      {/* Group Tabs */}
      <div className="flex gap-2 mb-4">
        {GROUPS.map((g) => (
          <button key={g.key} onClick={() => setGroup(g.key)}
            className={`px-3 py-2 rounded-lg text-sm font-medium ${group === g.key ? "bg-white shadow" : "text-slate-500 hover:bg-slate-50"}`}>
            {g.label}
          </button>
        ))}
      </div>

      {/* Settings Form */}
      <div className="p-4 rounded-xl border border-slate-200 bg-white">
        <div className="space-y-4">
          {settings.map((s) => (
            <div key={s.key}>
              <label className="text-sm font-medium text-slate-700 mb-1 block">{s.label}</label>
              {renderInput(s)}
            </div>
          ))}
        </div>
        <div className="mt-4 flex gap-2">
          <button onClick={saveAll} disabled={bulkMut.isPending || Object.keys(edits).length === 0}
            className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium text-white disabled:opacity-50"
            style={{ background: "var(--color-brand)" }}>
            <Save className="w-4 h-4" /> Save Changes {Object.keys(edits).length > 0 && `(${Object.keys(edits).length})`}
          </button>
        </div>
      </div>
    </div>
  );
}

// ════════════════════════════════════════════════════════════════════════════
// Navigation Tab
// ════════════════════════════════════════════════════════════════════════════

function NavTab({ qc, editing, setEditing }: { qc: QueryClient; editing: string | null; setEditing: (v: string | null) => void }) {
  const [location, setLocation] = useState("header");
  const [form, setForm] = useState<Partial<NavItem>>({ location: "header" });

  const { data = [] } = useQuery({ queryKey: ["cms-nav", location], queryFn: () => cmsApi.nav.list(location) });
  const items: NavItem[] = Array.isArray(data) ? data : [];

  const createMut = useMutation({
    mutationFn: (d: Record<string, unknown>) => cmsApi.nav.create(d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["cms-nav"] }); setForm({ location }); },
  });
  const updateMut = useMutation({
    mutationFn: ({ id, ...d }: { id: string } & Record<string, unknown>) => cmsApi.nav.update(id, d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["cms-nav"] }); setEditing(null); },
  });
  const deleteMut = useMutation({
    mutationFn: cmsApi.nav.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cms-nav"] }),
  });

  const saveForm = () => {
    if (editing) {
      updateMut.mutate({ id: editing, ...form });
    } else {
      createMut.mutate(form as Record<string, unknown>);
    }
  };

  return (
    <div>
      {/* Location Tabs */}
      <div className="flex gap-2 mb-4">
        {["header", "footer", "mobile", "sidebar"].map((loc) => (
          <button key={loc} onClick={() => setLocation(loc)}
            className={`px-3 py-2 rounded-lg text-sm font-medium capitalize ${location === loc ? "bg-white shadow" : "text-slate-500 hover:bg-slate-50"}`}>
            {loc}
          </button>
        ))}
      </div>

      {/* Form */}
      <div className="p-4 mb-4 rounded-xl border border-slate-200 bg-white">
        <div className="flex items-center justify-between mb-3">
          <h3 className="font-semibold text-sm">{editing ? "Edit Item" : "Add Menu Item"}</h3>
          {editing && <button onClick={() => { setEditing(null); setForm({ location }); }}><X className="w-4 h-4 text-slate-400" /></button>}
        </div>
        <div className="grid grid-cols-3 gap-3">
          <input placeholder="Label" value={form.label ?? ""} onChange={(e) => setForm({ ...form, label: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
          <input placeholder="URL" value={form.url ?? ""} onChange={(e) => setForm({ ...form, url: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
          <input placeholder="Icon (lucide name)" value={form.icon ?? ""} onChange={(e) => setForm({ ...form, icon: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
        </div>
        <div className="mt-3 flex items-center gap-4">
          <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={form.is_external ?? false} onChange={(e) => setForm({ ...form, is_external: e.target.checked })} /> Open in new tab</label>
          <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={form.is_active ?? true} onChange={(e) => setForm({ ...form, is_active: e.target.checked })} /> Active</label>
          <button onClick={saveForm} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white ml-auto" style={{ background: "var(--color-brand)" }}>
            <Save className="w-4 h-4" /> {editing ? "Update" : "Add"}
          </button>
        </div>
      </div>

      {/* List */}
      <div className="grid gap-2">
        {items.map((item) => (
          <div key={item.id} className="flex items-center gap-3 p-3 rounded-xl border border-slate-200 bg-white">
            <GripVertical className="w-4 h-4 text-slate-300 cursor-grab" />
            {item.is_external && <ExternalLink className="w-3 h-3 text-slate-400" />}
            <span className="text-sm font-medium text-slate-800">{item.label}</span>
            <span className="text-xs text-slate-400">{item.url}</span>
            <div className="ml-auto flex items-center gap-2">
              <StatusBadge status={item.is_active ? "active" : "inactive"} />
              <button onClick={() => { setEditing(item.id); setForm(item); }} className="p-1.5 hover:bg-slate-100 rounded"><Pencil className="w-3.5 h-3.5 text-slate-500" /></button>
              <button onClick={() => deleteMut.mutate(item.id)} className="p-1.5 hover:bg-red-50 rounded"><Trash2 className="w-3.5 h-3.5 text-red-400" /></button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

import type { QueryClient } from "@tanstack/react-query";
