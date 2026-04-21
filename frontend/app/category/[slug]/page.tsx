// Sprint 18.1 — Server wrapper for /category/[slug]
// Responsibilities:
//   • generateMetadata → dynamic <title> + description for SEO
//   • Render JSON-LD (BreadcrumbList + CollectionPage) structured data
//   • Defer UI to CategoryClient (client component)
//
// Backend fetch uses NEXT_PUBLIC_API_URL when present; otherwise falls back to
// the Next proxy base, and finally degrades to generic metadata on failure.

import type { Metadata } from 'next';
import CategoryClient from './CategoryClient';

interface Params {
  slug: string;
}

interface BreadcrumbNode {
  id: string;
  slug: string;
  name_en: string;
  name_ar?: string;
  level: number;
}

interface CategoryData {
  id: string;
  slug: string;
  name_en: string;
  name_ar?: string;
  icon?: string;
  level: number;
  path: string;
  children?: { id: string; slug: string; name_en: string }[];
}

const API_BASE =
  process.env.NEXT_PUBLIC_API_URL ||
  process.env.API_URL ||
  'http://localhost:8080/api/v1';

async function fetchJSON<T>(path: string, revalidate = 600): Promise<T | null> {
  try {
    const r = await fetch(`${API_BASE}${path}`, {
      next: { revalidate },
      headers: { Accept: 'application/json' },
    });
    if (!r.ok) return null;
    const body = await r.json();
    return (body?.data ?? null) as T | null;
  } catch {
    return null;
  }
}

export async function generateMetadata(
  { params }: { params: Promise<Params> },
): Promise<Metadata> {
  const { slug } = await params;
  const data = await fetchJSON<{ category: CategoryData }>(`/category/${slug}`);
  const cat = data?.category;
  if (!cat) {
    return {
      title: 'Category — Mnbarh',
      description: 'Browse listings on Mnbarh marketplace.',
    };
  }
  const title = `${cat.name_en} for sale — Mnbarh`;
  const description = cat.name_ar
    ? `Browse ${cat.name_en} (${cat.name_ar}) listings on Mnbarh. Buy and sell across the GCC with escrow protection.`
    : `Browse ${cat.name_en} listings on Mnbarh. Buy and sell across the GCC with escrow protection.`;
  const canonical = `/category/${cat.slug}`;
  return {
    title,
    description,
    alternates: { canonical },
    openGraph: {
      title,
      description,
      type: 'website',
      url: canonical,
    },
    twitter: {
      card: 'summary_large_image',
      title,
      description,
    },
  };
}

export default async function CategoryPage(
  { params }: { params: Promise<Params> },
) {
  const { slug } = await params;

  // Fetch data in parallel for JSON-LD. Failures degrade silently — the
  // client component re-fetches its own data on mount so UX is unaffected.
  const [catRes, bcRes] = await Promise.all([
    fetchJSON<{ category: CategoryData }>(`/category/${slug}`),
    fetchJSON<{ breadcrumb: BreadcrumbNode[] }>(`/category/${slug}/breadcrumb`),
  ]);
  const category = catRes?.category ?? null;
  const breadcrumb = bcRes?.breadcrumb ?? [];

  const siteUrl =
    process.env.NEXT_PUBLIC_SITE_URL || 'https://mnbarh.com';

  const jsonLd: Record<string, unknown>[] = [];

  if (breadcrumb.length > 0) {
    jsonLd.push({
      '@context': 'https://schema.org',
      '@type': 'BreadcrumbList',
      itemListElement: [
        {
          '@type': 'ListItem',
          position: 1,
          name: 'Home',
          item: siteUrl,
        },
        ...breadcrumb.map((b, i) => ({
          '@type': 'ListItem',
          position: i + 2,
          name: b.name_en,
          item: `${siteUrl}/category/${b.slug}`,
        })),
      ],
    });
  }

  if (category) {
    jsonLd.push({
      '@context': 'https://schema.org',
      '@type': 'CollectionPage',
      name: category.name_en,
      description: `Listings in the ${category.name_en} category on Mnbarh.`,
      url: `${siteUrl}/category/${category.slug}`,
      inLanguage: ['en', 'ar'],
    });
  }

  return (
    <>
      {jsonLd.map((ld, i) => (
        <script
          key={i}
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(ld) }}
        />
      ))}
      <CategoryClient />
    </>
  );
}
