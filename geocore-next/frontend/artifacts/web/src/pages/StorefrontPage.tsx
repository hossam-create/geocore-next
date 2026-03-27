import { useParams, Link } from "wouter";
import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";
import { ListingCard } from "@/components/listings/ListingCard";
import { LoadingGrid } from "@/components/ui/LoadingGrid";
import { Eye, Store, MessageCircle } from "lucide-react";

export default function StorefrontPage() {
  const params = useParams<{ slug: string }>();
  const slug = params.slug;

  const { data, isLoading, error } = useQuery({
    queryKey: ["storefront", slug],
    queryFn: () => api.get(`/stores/${slug}`).then((r) => r.data.data),
    retry: false,
  });

  if (isLoading) {
    return (
      <div className="max-w-7xl mx-auto px-4 py-10">
        <div className="h-48 bg-gray-100 rounded-2xl animate-pulse mb-6" />
        <LoadingGrid count={6} />
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="text-center py-20 text-gray-400">
        <p className="text-5xl mb-4">🏪</p>
        <p className="text-lg font-semibold">Storefront not found</p>
        <Link href="/stores" className="mt-4 text-[#0071CE] hover:underline block text-sm">
          ← Browse storefronts
        </Link>
      </div>
    );
  }

  const storefront = data?.storefront ?? data;
  const listings: any[] = data?.listings ?? [];

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      <div className="bg-white rounded-2xl shadow-sm overflow-hidden mb-8">
        <div className="h-48 relative overflow-hidden">
          {storefront.banner_url ? (
            <img src={storefront.banner_url} alt="banner" className="w-full h-full object-cover" />
          ) : (
            <div className="w-full h-full bg-gradient-to-r from-[#0071CE] to-[#003f75]" />
          )}
        </div>

        <div className="px-6 pb-6">
          <div className="flex items-end justify-between -mt-10 mb-4 flex-wrap gap-3">
            <div className="w-20 h-20 rounded-2xl border-4 border-white shadow-md bg-[#FFC220] flex items-center justify-center text-3xl font-extrabold text-gray-900 overflow-hidden">
              {storefront.logo_url ? (
                <img src={storefront.logo_url} alt="logo" className="w-full h-full object-cover" />
              ) : (
                storefront.name?.[0]?.toUpperCase()
              )}
            </div>
            <div className="flex gap-2 flex-wrap">
              <div className="flex items-center gap-1.5 text-xs text-gray-400 bg-gray-50 rounded-full px-3 py-1.5">
                <Eye size={12} /> {storefront.views?.toLocaleString() ?? 0} views
              </div>
              <button className="flex items-center gap-1.5 text-sm bg-[#0071CE] text-white px-4 py-2 rounded-xl hover:bg-[#005BA1] transition-colors font-medium">
                <MessageCircle size={14} /> Message Seller
              </button>
            </div>
          </div>

          <h1 className="text-2xl font-extrabold text-gray-900">{storefront.name}</h1>
          <p className="text-xs text-gray-400 mt-0.5 flex items-center gap-1">
            <Store size={11} /> geocore.com/stores/{storefront.slug}
          </p>

          {storefront.welcome_msg && (
            <div className="mt-4 bg-blue-50 border border-blue-100 rounded-xl px-5 py-3 text-sm text-blue-700 italic">
              "{storefront.welcome_msg}"
            </div>
          )}

          {storefront.description && (
            <p className="text-sm text-gray-600 mt-4 leading-relaxed max-w-3xl">
              {storefront.description}
            </p>
          )}
        </div>
      </div>

      <div className="flex items-center justify-between mb-5">
        <h2 className="text-xl font-bold text-gray-900">
          Active Listings <span className="text-gray-400 font-normal text-base">({listings.length})</span>
        </h2>
      </div>

      {listings.length === 0 ? (
        <div className="text-center py-16 text-gray-400">
          <p className="text-4xl mb-3">📦</p>
          <p className="font-semibold">No active listings</p>
          <p className="text-sm mt-1">This seller hasn't posted any listings yet</p>
        </div>
      ) : (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
          {listings.map((listing: any) => (
            <ListingCard key={listing.id} listing={listing} />
          ))}
        </div>
      )}
    </div>
  );
}
