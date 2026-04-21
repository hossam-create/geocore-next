'use client';

import { useEffect, useState, useMemo } from 'react';
import { useParams, useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { ChevronRight, Filter, MapPin, Tag, Home } from 'lucide-react';
import api from '@/lib/api';

// ─── Types ─────────────────────────────────────────────────────────────────
interface BreadcrumbNode {
  id: string;
  slug: string;
  name_en: string;
  name_ar?: string;
  level: number;
}

interface CategoryChild {
  id: string;
  slug: string;
  name_en: string;
  name_ar?: string;
  icon?: string;
  level: number;
}

interface Category {
  id: string;
  slug: string;
  name_en: string;
  name_ar?: string;
  icon?: string;
  level: number;
  path: string;
  children?: CategoryChild[];
}

interface BackendListing {
  id: string;
  title: string;
  price: number | null;
  currency: string;
  condition: string;
  city: string;
  country: string;
  images?: { url: string }[];
  category?: { name_en?: string } | null;
}

interface Facets {
  categories: { id: string; name: string; count: number }[];
  conditions: { value: string; count: number }[];
  price_ranges: { min: number; max: number; label: string; count: number }[];
}

interface SearchData {
  results: BackendListing[];
  total: number;
  page: number;
  per_page: number;
  pages: number;
  facets: Facets;
}

// ─── Page ──────────────────────────────────────────────────────────────────
export default function CategoryPage() {
  const params = useParams<{ slug: string }>();
  const slug = params?.slug;
  const router = useRouter();
  const qs = useSearchParams();

  const [category, setCategory] = useState<Category | null>(null);
  const [breadcrumb, setBreadcrumb] = useState<BreadcrumbNode[]>([]);
  const [data, setData] = useState<SearchData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filters driven by URL
  const filters = useMemo(() => ({
    condition: qs.get('condition') ?? '',
    min_price: qs.get('min_price') ?? '',
    max_price: qs.get('max_price') ?? '',
    city: qs.get('city') ?? '',
    sort_by: qs.get('sort_by') ?? 'relevance',
    page: Number(qs.get('page') ?? '1'),
  }), [qs]);

  // Load category + breadcrumb once per slug
  useEffect(() => {
    if (!slug) return;
    let cancelled = false;
    (async () => {
      try {
        const [catRes, bcRes] = await Promise.all([
          api.get(`/category/${slug}`),
          api.get(`/category/${slug}/breadcrumb`),
        ]);
        if (cancelled) return;
        setCategory(catRes.data?.data?.category ?? null);
        setBreadcrumb(bcRes.data?.data?.breadcrumb ?? []);
      } catch (e) {
        if (!cancelled) setError('Category not found');
      }
    })();
    return () => { cancelled = true; };
  }, [slug]);

  // Load listings whenever filters or slug change
  useEffect(() => {
    if (!slug) return;
    let cancelled = false;
    setLoading(true);
    const p = new URLSearchParams({ per_page: '24', page: String(filters.page) });
    if (filters.condition) p.set('condition', filters.condition);
    if (filters.min_price) p.set('min_price', filters.min_price);
    if (filters.max_price) p.set('max_price', filters.max_price);
    if (filters.city) p.set('city', filters.city);
    if (filters.sort_by) p.set('sort_by', filters.sort_by);
    api
      .get(`/category/${slug}/listings?${p.toString()}`)
      .then((res) => { if (!cancelled) setData(res.data?.data ?? null); })
      .catch(() => { if (!cancelled) setError('Failed to load listings'); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [slug, filters]);

  const updateFilter = (key: string, value: string) => {
    const next = new URLSearchParams(qs.toString());
    if (value) next.set(key, value);
    else next.delete(key);
    next.delete('page'); // reset pagination on filter change
    router.push(`/category/${slug}?${next.toString()}`);
  };

  const goToPage = (p: number) => {
    const next = new URLSearchParams(qs.toString());
    next.set('page', String(p));
    router.push(`/category/${slug}?${next.toString()}`);
  };

  if (error && !category) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <p className="text-xl font-semibold text-gray-800">Category not found</p>
          <Link href="/" className="text-[#0071CE] hover:underline mt-2 inline-block">Back to home</Link>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Breadcrumb */}
      <div className="bg-white border-b border-gray-100">
        <div className="max-w-7xl mx-auto px-4 py-3 flex items-center gap-1 text-sm text-gray-500 overflow-x-auto">
          <Link href="/" className="flex items-center gap-1 hover:text-[#0071CE]">
            <Home className="w-4 h-4" /> Home
          </Link>
          {breadcrumb.map((b, i) => (
            <span key={b.id} className="flex items-center gap-1 whitespace-nowrap">
              <ChevronRight className="w-4 h-4 text-gray-300" />
              {i === breadcrumb.length - 1 ? (
                <span className="text-gray-800 font-medium">{b.name_en}</span>
              ) : (
                <Link href={`/category/${b.slug}`} className="hover:text-[#0071CE]">{b.name_en}</Link>
              )}
            </span>
          ))}
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 py-6 grid grid-cols-1 lg:grid-cols-[260px_1fr] gap-6">
        {/* ── Sidebar filters ── */}
        <aside className="bg-white rounded-2xl shadow-sm border border-gray-100 p-5 h-fit sticky top-4">
          <div className="flex items-center gap-2 mb-4 text-gray-800 font-semibold">
            <Filter className="w-4 h-4" /> Filters
          </div>

          {/* Subcategories */}
          {category?.children && category.children.length > 0 && (
            <div className="mb-5">
              <h3 className="text-xs font-semibold uppercase text-gray-400 mb-2">Subcategories</h3>
              <ul className="space-y-1">
                {category.children.map((c) => (
                  <li key={c.id}>
                    <Link
                      href={`/category/${c.slug}`}
                      className="flex items-center gap-2 text-sm text-gray-700 hover:text-[#0071CE] py-1"
                    >
                      {c.icon && <span>{c.icon}</span>}
                      <span>{c.name_en}</span>
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          )}

          <FilterGroup
            label="Condition"
            value={filters.condition}
            options={(data?.facets?.conditions ?? []).map(c => ({ value: c.value, label: `${c.value} (${c.count})` }))}
            onChange={(v) => updateFilter('condition', v)}
          />

          <div className="mb-5">
            <h3 className="text-xs font-semibold uppercase text-gray-400 mb-2">Price (AED)</h3>
            <div className="flex gap-2">
              <input
                type="number" placeholder="Min" value={filters.min_price}
                onChange={(e) => updateFilter('min_price', e.target.value)}
                className="w-full px-2 py-1.5 text-sm border border-gray-200 rounded-lg focus:outline-none focus:border-[#0071CE]"
              />
              <input
                type="number" placeholder="Max" value={filters.max_price}
                onChange={(e) => updateFilter('max_price', e.target.value)}
                className="w-full px-2 py-1.5 text-sm border border-gray-200 rounded-lg focus:outline-none focus:border-[#0071CE]"
              />
            </div>
          </div>

          <div className="mb-5">
            <h3 className="text-xs font-semibold uppercase text-gray-400 mb-2">Location</h3>
            <input
              placeholder="City" value={filters.city}
              onChange={(e) => updateFilter('city', e.target.value)}
              className="w-full px-2 py-1.5 text-sm border border-gray-200 rounded-lg focus:outline-none focus:border-[#0071CE]"
            />
          </div>

          <div className="mb-2">
            <h3 className="text-xs font-semibold uppercase text-gray-400 mb-2">Sort</h3>
            <select
              value={filters.sort_by}
              onChange={(e) => updateFilter('sort_by', e.target.value)}
              className="w-full px-2 py-1.5 text-sm border border-gray-200 rounded-lg focus:outline-none focus:border-[#0071CE]"
            >
              <option value="relevance">Relevance</option>
              <option value="date">Newest</option>
              <option value="price_asc">Price: Low → High</option>
              <option value="price_desc">Price: High → Low</option>
            </select>
          </div>
        </aside>

        {/* ── Main results column ── */}
        <main>
          <div className="flex items-end justify-between mb-5">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">{category?.name_en ?? '—'}</h1>
              {category?.name_ar && <p className="text-sm text-gray-500">{category.name_ar}</p>}
              {data && <p className="text-xs text-gray-400 mt-1">{data.total.toLocaleString()} listings</p>}
            </div>
          </div>

          {loading && (
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
              {Array.from({ length: 8 }).map((_, i) => (
                <div key={i} className="bg-white rounded-2xl overflow-hidden animate-pulse">
                  <div className="h-40 bg-gray-100" />
                  <div className="p-3 space-y-2">
                    <div className="h-3 bg-gray-100 rounded w-3/4" />
                    <div className="h-4 bg-gray-100 rounded w-1/2" />
                  </div>
                </div>
              ))}
            </div>
          )}

          {!loading && data && data.results.length === 0 && (
            <div className="bg-white rounded-2xl shadow-sm p-12 text-center text-gray-500">
              No listings in this category yet.
            </div>
          )}

          {!loading && data && data.results.length > 0 && (
            <>
              <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
                {data.results.map((l) => (
                  <button
                    key={l.id}
                    onClick={() => router.push(`/listings/${l.id}`)}
                    className="bg-white rounded-2xl overflow-hidden shadow-sm border border-gray-100 hover:shadow-md hover:border-[#0071CE]/30 transition-all text-left"
                  >
                    <div className="relative">
                      <img
                        src={l.images?.[0]?.url ?? `https://picsum.photos/seed/${l.id}/400/300`}
                        alt={l.title}
                        className="w-full h-40 object-cover"
                      />
                      {l.condition && (
                        <span className="absolute top-2 right-2 bg-white/90 text-[10px] px-2 py-0.5 rounded-full text-gray-600">
                          {l.condition}
                        </span>
                      )}
                    </div>
                    <div className="p-3">
                      <p className="text-sm font-semibold text-gray-800 line-clamp-2 leading-snug">{l.title}</p>
                      {l.price != null && (
                        <p className="text-[#0071CE] font-bold mt-1.5">
                          {l.currency} {l.price.toLocaleString()}
                        </p>
                      )}
                      <div className="flex items-center gap-2 mt-2 text-xs text-gray-400">
                        {(l.city || l.country) && (
                          <span className="flex items-center gap-0.5">
                            <MapPin className="w-3 h-3" /> {l.city || l.country}
                          </span>
                        )}
                        {l.category?.name_en && (
                          <span className="flex items-center gap-0.5">
                            <Tag className="w-3 h-3" /> {l.category.name_en}
                          </span>
                        )}
                      </div>
                    </div>
                  </button>
                ))}
              </div>

              {/* Pagination */}
              {data.pages > 1 && (
                <div className="flex items-center justify-center gap-2 mt-8">
                  <button
                    disabled={filters.page <= 1}
                    onClick={() => goToPage(filters.page - 1)}
                    className="px-3 py-1.5 text-sm bg-white border border-gray-200 rounded-lg disabled:opacity-40"
                  >
                    Previous
                  </button>
                  <span className="text-sm text-gray-500">
                    Page {filters.page} / {data.pages}
                  </span>
                  <button
                    disabled={filters.page >= data.pages}
                    onClick={() => goToPage(filters.page + 1)}
                    className="px-3 py-1.5 text-sm bg-white border border-gray-200 rounded-lg disabled:opacity-40"
                  >
                    Next
                  </button>
                </div>
              )}
            </>
          )}
        </main>
      </div>
    </div>
  );
}

// ─── Sub-component ─────────────────────────────────────────────────────────
function FilterGroup({
  label, value, options, onChange,
}: {
  label: string;
  value: string;
  options: { value: string; label: string }[];
  onChange: (v: string) => void;
}) {
  if (options.length === 0) return null;
  return (
    <div className="mb-5">
      <h3 className="text-xs font-semibold uppercase text-gray-400 mb-2">{label}</h3>
      <ul className="space-y-1">
        <li>
          <button
            onClick={() => onChange('')}
            className={`text-sm ${value === '' ? 'text-[#0071CE] font-semibold' : 'text-gray-700'} hover:text-[#0071CE]`}
          >
            All
          </button>
        </li>
        {options.map((o) => (
          <li key={o.value}>
            <button
              onClick={() => onChange(o.value)}
              className={`text-sm ${value === o.value ? 'text-[#0071CE] font-semibold' : 'text-gray-700'} hover:text-[#0071CE]`}
            >
              {o.label}
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
