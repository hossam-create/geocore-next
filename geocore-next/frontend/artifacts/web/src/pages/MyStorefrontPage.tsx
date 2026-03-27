import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Link, useLocation } from "wouter";
import api from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { formatPrice } from "@/lib/utils";
import {
  Store,
  Edit3,
  Eye,
  TrendingUp,
  Package,
  Plus,
  Trash2,
  CheckCircle,
  Clock,
  XCircle,
  AlertCircle,
} from "lucide-react";

// ── Types ────────────────────────────────────────────────────────────────────

interface Storefront {
  id: string;
  name: string;
  slug: string;
  description?: string;
  welcome_msg?: string;
  logo_url?: string;
  banner_url?: string;
  views?: number;
}

interface StorefrontFormData {
  name: string;
  description: string;
  welcome_msg: string;
  logo_url: string;
  banner_url: string;
  slug: string;
}

interface MyListing {
  id: string;
  title: string;
  price: number | null;
  currency: string;
  status: "active" | "sold" | "inactive" | "pending";
  condition: string;
  view_count?: number;
  created_at: string;
  images?: { url: string }[];
  category?: { name_en?: string; slug?: string } | null;
}

interface ListingsResponse {
  listings: MyListing[];
  total: number;
}

// ── Status helpers ────────────────────────────────────────────────────────────

const STATUS_LABEL: Record<string, string> = {
  active: "Active",
  sold: "Sold",
  inactive: "Inactive",
  pending: "Pending Review",
};

function StatusBadge({ status }: { status: string }) {
  const icons: Record<string, React.ReactNode> = {
    active: <CheckCircle size={12} />,
    sold: <TrendingUp size={12} />,
    inactive: <XCircle size={12} />,
    pending: <Clock size={12} />,
  };
  const colors: Record<string, string> = {
    active: "text-green-600 bg-green-50",
    sold: "text-blue-600 bg-blue-50",
    inactive: "text-gray-500 bg-gray-100",
    pending: "text-amber-600 bg-amber-50",
  };
  const cls = colors[status] ?? "text-gray-500 bg-gray-100";
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${cls}`}>
      {icons[status]} {STATUS_LABEL[status] ?? status}
    </span>
  );
}

// ── Edit listing modal ────────────────────────────────────────────────────────

interface EditModalProps {
  listing: MyListing;
  onClose: () => void;
  onSaved: () => void;
}

function EditListingModal({ listing, onClose, onSaved }: EditModalProps) {
  const qc = useQueryClient();
  const [form, setForm] = useState({
    title: listing.title,
    price: String(listing.price ?? ""),
    condition: listing.condition ?? "",
    status: listing.status,
  });
  const [error, setError] = useState("");

  const mutation = useMutation({
    mutationFn: (data: typeof form) =>
      api.put(`/listings/${listing.id}`, {
        title: data.title,
        price: data.price ? Number(data.price) : undefined,
        condition: data.condition.toLowerCase().replace(/\s+/g, "-"),
        status: data.status,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["my-listings"] });
      onSaved();
    },
    onError: (err: { response?: { data?: { message?: string } } }) => {
      setError(err?.response?.data?.message ?? "Failed to update listing.");
    },
  });

  const handle = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
    setForm((f) => ({ ...f, [e.target.name]: e.target.value }));

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4">
      <div className="bg-white rounded-2xl shadow-xl w-full max-w-md p-6">
        <h3 className="font-bold text-gray-900 text-lg mb-5">Edit Listing</h3>
        <div className="space-y-4">
          <div>
            <label className="text-sm font-medium text-gray-700 block mb-1.5">Title</label>
            <input
              name="title"
              value={form.title}
              onChange={handle}
              className="w-full border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
            />
          </div>
          <div>
            <label className="text-sm font-medium text-gray-700 block mb-1.5">Price (AED)</label>
            <input
              name="price"
              type="number"
              min="0"
              value={form.price}
              onChange={handle}
              className="w-full border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
            />
          </div>
          <div>
            <label className="text-sm font-medium text-gray-700 block mb-1.5">Condition</label>
            <select
              name="condition"
              value={form.condition}
              onChange={handle}
              className="w-full border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
            >
              {["new", "like-new", "good", "fair", "for-parts"].map((c) => (
                <option key={c} value={c}>{c.replace(/-/g, " ").replace(/\b\w/g, (x) => x.toUpperCase())}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-sm font-medium text-gray-700 block mb-1.5">Status</label>
            <select
              name="status"
              value={form.status}
              onChange={handle}
              className="w-full border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
            >
              {["active", "inactive", "sold"].map((s) => (
                <option key={s} value={s}>{STATUS_LABEL[s] ?? s}</option>
              ))}
            </select>
          </div>
          {error && <p className="text-sm text-red-500">{error}</p>}
          <div className="flex gap-3 pt-2">
            <button
              onClick={() => mutation.mutate(form)}
              disabled={mutation.isPending}
              className="flex-1 bg-[#0071CE] hover:bg-[#005BA1] text-white font-bold py-3 rounded-xl transition-colors disabled:opacity-60"
            >
              {mutation.isPending ? "Saving…" : "Save Changes"}
            </button>
            <button
              onClick={onClose}
              className="px-6 py-3 border border-gray-200 rounded-xl text-sm text-gray-600 hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

// ── Main page ─────────────────────────────────────────────────────────────────

export default function MyStorefrontPage() {
  const { isAuthenticated } = useAuthStore();
  const [, navigate] = useLocation();
  const qc = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [form, setForm] = useState<StorefrontFormData>({
    name: "", description: "", welcome_msg: "", logo_url: "", banner_url: "", slug: "",
  });
  const [msg, setMsg] = useState("");
  const [editingListing, setEditingListing] = useState<MyListing | null>(null);
  const [deleteId, setDeleteId] = useState<string | null>(null);

  if (!isAuthenticated) {
    navigate("/login?next=/my-store");
    return null;
  }

  const { data: storefront, isLoading, error } = useQuery<Storefront>({
    queryKey: ["storefront", "mine"],
    queryFn: () => api.get("/stores/me").then((r) => r.data.data as Storefront),
    retry: false,
  });

  const { data: listingsData, isLoading: listingsLoading } = useQuery<ListingsResponse>({
    queryKey: ["my-listings"],
    queryFn: () =>
      api.get("/listings/me").then((r) => {
        const d = r.data.data as MyListing[] | null;
        const listings = Array.isArray(d) ? d : [];
        return { listings, total: listings.length };
      }),
  });

  const myListings = listingsData?.listings ?? [];
  const activeCount = myListings.filter((l) => l.status === "active").length;
  const soldCount = myListings.filter((l) => l.status === "sold").length;

  const createMutation = useMutation({
    mutationFn: (data: StorefrontFormData) => api.post("/stores", data).then((r) => r.data.data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["storefront", "mine"] });
      setMsg("Storefront created!");
    },
    onError: (err: { response?: { data?: { message?: string } } }) => {
      setMsg(err?.response?.data?.message ?? "Failed to create storefront.");
    },
  });

  const updateMutation = useMutation({
    mutationFn: (data: Partial<StorefrontFormData>) => api.put("/stores/me", data).then((r) => r.data.data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["storefront", "mine"] });
      setEditing(false);
      setMsg("Storefront updated!");
    },
    onError: (err: { response?: { data?: { message?: string } } }) => {
      setMsg(err?.response?.data?.message ?? "Failed to update storefront.");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/listings/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["my-listings"] });
      setDeleteId(null);
    },
  });

  const hasStore = !error && storefront;

  return (
    <div className="max-w-3xl mx-auto px-4 py-10">
      <h1 className="text-2xl font-bold text-gray-900 mb-2 flex items-center gap-2">
        <Store size={24} className="text-[#0071CE]" /> My Storefront
      </h1>
      <p className="text-gray-500 text-sm mb-8">Your public seller page where buyers can discover all your listings.</p>

      {isLoading ? (
        <div className="h-40 bg-white rounded-2xl animate-pulse shadow-sm mb-8" />
      ) : hasStore ? (
        <div className="mb-8">
          <div className="bg-white rounded-2xl shadow-sm overflow-hidden">
            {storefront.banner_url ? (
              <img src={storefront.banner_url} alt="banner" className="w-full h-36 object-cover" />
            ) : (
              <div className="w-full h-36 bg-gradient-to-r from-[#0071CE] to-[#003f75]" />
            )}
            <div className="px-6 pb-6">
              <div className="flex items-end justify-between -mt-8 mb-4">
                <div className="w-16 h-16 rounded-2xl border-4 border-white shadow bg-[#FFC220] flex items-center justify-center text-2xl font-extrabold text-gray-900 overflow-hidden">
                  {storefront.logo_url ? (
                    <img src={storefront.logo_url} alt="logo" className="w-full h-full object-cover" />
                  ) : (
                    storefront.name?.[0]?.toUpperCase()
                  )}
                </div>
                <div className="flex gap-2">
                  <Link
                    href={`/stores/${storefront.slug}`}
                    className="flex items-center gap-1.5 text-sm border border-[#0071CE] text-[#0071CE] px-3 py-1.5 rounded-lg hover:bg-blue-50 transition-colors"
                  >
                    <Eye size={14} /> Preview
                  </Link>
                  <button
                    onClick={() => {
                      setForm({
                        name: storefront.name ?? "",
                        description: storefront.description ?? "",
                        welcome_msg: storefront.welcome_msg ?? "",
                        logo_url: storefront.logo_url ?? "",
                        banner_url: storefront.banner_url ?? "",
                        slug: storefront.slug ?? "",
                      });
                      setEditing(true);
                    }}
                    className="flex items-center gap-1.5 text-sm bg-[#0071CE] text-white px-3 py-1.5 rounded-lg hover:bg-[#005BA1] transition-colors"
                  >
                    <Edit3 size={14} /> Edit
                  </button>
                </div>
              </div>
              <h2 className="text-xl font-bold text-gray-900">{storefront.name}</h2>
              <p className="text-xs text-gray-400 mt-0.5">geocore.com/stores/{storefront.slug}</p>
              {storefront.description && (
                <p className="text-sm text-gray-600 mt-3 leading-relaxed">{storefront.description}</p>
              )}
              {storefront.welcome_msg && (
                <div className="mt-3 bg-blue-50 border border-blue-100 rounded-xl px-4 py-3 text-sm text-blue-700 italic">
                  "{storefront.welcome_msg}"
                </div>
              )}
              <div className="grid grid-cols-3 gap-4 mt-5">
                {[
                  { label: "Total Views", value: storefront.views?.toLocaleString() ?? "0", icon: <Eye size={16} /> },
                  { label: "Active Listings", value: String(activeCount), icon: <Package size={16} /> },
                  { label: "Sold Items", value: String(soldCount), icon: <TrendingUp size={16} /> },
                ].map((s) => (
                  <div key={s.label} className="bg-gray-50 rounded-xl p-3 text-center">
                    <div className="text-[#0071CE] flex justify-center mb-1">{s.icon}</div>
                    <p className="text-lg font-bold text-gray-800">{s.value}</p>
                    <p className="text-xs text-gray-400">{s.label}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {editing && (
            <div className="mt-4">
              <StorefrontForm
                form={form}
                setForm={setForm}
                onSubmit={() => updateMutation.mutate(form)}
                onCancel={() => setEditing(false)}
                loading={updateMutation.isPending}
                submitLabel="Save Changes"
                msg={msg}
              />
            </div>
          )}
        </div>
      ) : (
        <div className="mb-8">
          <div className="bg-white rounded-2xl shadow-sm p-8 text-center mb-4">
            <div className="w-20 h-20 rounded-full bg-blue-50 flex items-center justify-center mx-auto mb-4">
              <Store size={36} className="text-[#0071CE]" />
            </div>
            <h2 className="text-xl font-bold text-gray-800 mb-2">You don't have a storefront yet</h2>
            <p className="text-gray-500 text-sm leading-relaxed max-w-md mx-auto">
              Create your free seller storefront — a dedicated page with your own URL where buyers can browse all your listings.
            </p>
          </div>
          <StorefrontForm
            form={form}
            setForm={setForm}
            onSubmit={() => createMutation.mutate(form)}
            onCancel={() => {}}
            loading={createMutation.isPending}
            submitLabel="Create My Storefront"
            msg={msg}
            isCreate
          />
        </div>
      )}

      {msg && !editing && (
        <p className={`mb-6 text-center text-sm font-medium ${msg.includes("created") || msg.includes("updated") ? "text-green-600" : "text-red-500"}`}>
          {msg}
        </p>
      )}

      {/* ── My Listings Section ─────────────────────────────────────────── */}
      <div className="bg-white rounded-2xl shadow-sm overflow-hidden">
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100">
          <h2 className="text-lg font-bold text-gray-900 flex items-center gap-2">
            <Package size={20} className="text-[#0071CE]" /> My Listings
          </h2>
          <Link
            href="/sell"
            className="flex items-center gap-1.5 text-sm bg-[#0071CE] text-white px-3 py-1.5 rounded-lg hover:bg-[#005BA1] transition-colors"
          >
            <Plus size={14} /> Add Listing
          </Link>
        </div>

        {listingsLoading ? (
          <div className="p-6 space-y-3">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-16 bg-gray-100 rounded-xl animate-pulse" />
            ))}
          </div>
        ) : myListings.length === 0 ? (
          <div className="p-12 text-center">
            <Package size={40} className="text-gray-300 mx-auto mb-3" />
            <p className="text-gray-500 text-sm">No listings yet</p>
            <Link href="/sell" className="mt-3 inline-block text-[#0071CE] text-sm hover:underline">
              Post your first listing
            </Link>
          </div>
        ) : (
          <ul className="divide-y divide-gray-50">
            {myListings.map((listing) => {
              const thumb = listing.images?.[0]?.url ?? `https://picsum.photos/seed/${listing.id}/80/80`;
              const cat = listing.category?.name_en ?? listing.category?.slug ?? "";
              return (
                <li key={listing.id} className="flex items-center gap-4 px-6 py-4 hover:bg-gray-50 transition-colors">
                  <img
                    src={thumb}
                    alt={listing.title}
                    className="w-14 h-14 rounded-xl object-cover flex-shrink-0 bg-gray-100"
                  />
                  <div className="flex-1 min-w-0">
                    <p className="font-semibold text-gray-900 text-sm truncate">{listing.title}</p>
                    <div className="flex items-center gap-2 mt-0.5 flex-wrap">
                      <StatusBadge status={listing.status} />
                      {cat && <span className="text-xs text-gray-400">{cat}</span>}
                      <span className="text-xs text-gray-400 flex items-center gap-0.5">
                        <Eye size={11} /> {listing.view_count ?? 0}
                      </span>
                    </div>
                  </div>
                  <div className="text-right flex-shrink-0">
                    <p className="font-bold text-gray-900 text-sm">
                      {listing.price != null
                        ? formatPrice(listing.price, listing.currency ?? "AED")
                        : "Contact"}
                    </p>
                  </div>
                  <div className="flex gap-1.5 flex-shrink-0">
                    <Link
                      href={`/listings/${listing.id}`}
                      className="p-2 rounded-lg border border-gray-200 hover:bg-gray-50 text-gray-500 transition-colors"
                      title="View"
                    >
                      <Eye size={15} />
                    </Link>
                    <button
                      onClick={() => setEditingListing(listing)}
                      className="p-2 rounded-lg border border-gray-200 hover:bg-blue-50 text-[#0071CE] transition-colors"
                      title="Edit"
                    >
                      <Edit3 size={15} />
                    </button>
                    <button
                      onClick={() => setDeleteId(listing.id)}
                      className="p-2 rounded-lg border border-red-100 hover:bg-red-50 text-red-500 transition-colors"
                      title="Delete"
                    >
                      <Trash2 size={15} />
                    </button>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </div>

      {/* ── Edit listing modal ───────────────────────────────────────────── */}
      {editingListing && (
        <EditListingModal
          listing={editingListing}
          onClose={() => setEditingListing(null)}
          onSaved={() => setEditingListing(null)}
        />
      )}

      {/* ── Delete confirmation ──────────────────────────────────────────── */}
      {deleteId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4">
          <div className="bg-white rounded-2xl shadow-xl w-full max-w-sm p-6 text-center">
            <AlertCircle size={40} className="text-red-500 mx-auto mb-3" />
            <h3 className="font-bold text-gray-900 text-lg mb-2">Delete Listing?</h3>
            <p className="text-gray-500 text-sm mb-6">This action cannot be undone. The listing will be permanently removed.</p>
            <div className="flex gap-3">
              <button
                onClick={() => setDeleteId(null)}
                className="flex-1 py-3 border border-gray-200 rounded-xl text-sm text-gray-600 hover:bg-gray-50 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => deleteMutation.mutate(deleteId)}
                disabled={deleteMutation.isPending}
                className="flex-1 py-3 bg-red-500 hover:bg-red-600 text-white font-bold rounded-xl transition-colors disabled:opacity-60"
              >
                {deleteMutation.isPending ? "Deleting…" : "Delete"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// ── Storefront form ────────────────────────────────────────────────────────────

interface StorefrontFormProps {
  form: StorefrontFormData;
  setForm: (f: StorefrontFormData) => void;
  onSubmit: () => void;
  onCancel: () => void;
  loading: boolean;
  submitLabel: string;
  msg?: string;
  isCreate?: boolean;
}

function StorefrontForm({ form, setForm, onSubmit, onCancel, loading, submitLabel, msg, isCreate }: StorefrontFormProps) {
  const handle = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
    setForm({ ...form, [e.target.name]: e.target.value });

  return (
    <div className="bg-white rounded-2xl shadow-sm p-6">
      <h3 className="font-bold text-gray-800 mb-5">{isCreate ? "Set up your storefront" : "Edit Storefront"}</h3>
      <div className="space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="text-sm font-medium text-gray-700 block mb-1.5">Store Name *</label>
            <input
              name="name"
              value={form.name}
              onChange={handle}
              required
              placeholder="Ahmed Phones"
              className="w-full border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
            />
          </div>
          <div>
            <label className="text-sm font-medium text-gray-700 block mb-1.5">URL Slug</label>
            <div className="flex items-center border border-gray-200 rounded-xl overflow-hidden">
              <span className="px-3 py-3 text-xs text-gray-400 bg-gray-50 border-r border-gray-200">stores/</span>
              <input
                name="slug"
                value={form.slug}
                onChange={handle}
                placeholder="ahmed-phones"
                className="flex-1 px-3 py-3 text-sm outline-none"
              />
            </div>
          </div>
        </div>

        <div>
          <label className="text-sm font-medium text-gray-700 block mb-1.5">Description</label>
          <textarea
            name="description"
            value={form.description}
            onChange={handle}
            rows={3}
            placeholder="Tell buyers about your store — what you sell, your experience, etc."
            className="w-full border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE] resize-none"
          />
        </div>

        <div>
          <label className="text-sm font-medium text-gray-700 block mb-1.5">Welcome Message</label>
          <input
            name="welcome_msg"
            value={form.welcome_msg}
            onChange={handle}
            placeholder="Welcome to my store! All items come with a 7-day return guarantee."
            className="w-full border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
          />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="text-sm font-medium text-gray-700 block mb-1.5">Logo URL</label>
            <input
              name="logo_url"
              value={form.logo_url}
              onChange={handle}
              placeholder="https://..."
              className="w-full border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
            />
          </div>
          <div>
            <label className="text-sm font-medium text-gray-700 block mb-1.5">Banner URL</label>
            <input
              name="banner_url"
              value={form.banner_url}
              onChange={handle}
              placeholder="https://..."
              className="w-full border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
            />
          </div>
        </div>

        {msg && (
          <p className={`text-sm font-medium ${msg.includes("created") || msg.includes("updated") ? "text-green-600" : "text-red-500"}`}>
            {msg}
          </p>
        )}

        <div className="flex gap-3 pt-2">
          <button
            onClick={onSubmit}
            disabled={loading || !form.name}
            className="flex-1 bg-[#0071CE] hover:bg-[#005BA1] text-white font-bold py-3 rounded-xl transition-colors disabled:opacity-60"
          >
            {loading ? "Saving…" : submitLabel}
          </button>
          {!isCreate && (
            <button
              onClick={onCancel}
              className="px-6 py-3 border border-gray-200 rounded-xl text-sm text-gray-600 hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
