'use client'
import Link from 'next/link';
import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";
import { Store, Eye } from "lucide-react";

const MOCK_STORES = [
  { id: "s1", slug: "ahmed-phones", name: "Ahmed Phones", description: "Premium smartphones and accessories. All items authentic and warranty-included.", views: 4821, logo_url: "", created_at: "2024-01-15" },
  { id: "s2", slug: "dubai-wheels", name: "Dubai Wheels", description: "Certified pre-owned vehicles. Toyota, BMW, Mercedes. Trade-ins welcome.", views: 12440, logo_url: "", created_at: "2023-09-05" },
  { id: "s3", slug: "gulf-luxury", name: "Gulf Luxury", description: "Authentic luxury watches and jewelry. Rolex, Patek Philippe, Cartier.", views: 8900, logo_url: "", created_at: "2023-11-20" },
  { id: "s4", slug: "riyadh-realty", name: "Riyadh Realty", description: "Residential and commercial properties across Riyadh. 10+ years experience.", views: 6350, logo_url: "", created_at: "2024-02-01" },
  { id: "s5", slug: "techzone-kw", name: "TechZone Kuwait", description: "Electronics, gaming, and gadgets. Best prices in Kuwait.", views: 3200, logo_url: "", created_at: "2024-03-10" },
  { id: "s6", slug: "fashion-forward-ae", name: "Fashion Forward", description: "Designer clothing and accessories. New arrivals weekly.", views: 5670, logo_url: "", created_at: "2023-12-01" },
];

function StoreCard({ store }: { store: any }) {
  const initial = store.name?.[0]?.toUpperCase() || "S";
  return (
    <Link href={`/stores/${store.slug}`}>
      <div className="bg-white rounded-2xl shadow-sm hover:shadow-md hover:-translate-y-0.5 transition-all overflow-hidden cursor-pointer group">
        <div className="h-24 bg-gradient-to-br from-[#0071CE] to-[#003f75]" />
        <div className="px-5 pb-5">
          <div className="w-14 h-14 rounded-xl border-4 border-white shadow bg-[#FFC220] flex items-center justify-center text-xl font-extrabold text-gray-900 -mt-7 mb-3 overflow-hidden">
            {store.logo_url ? (
              <img src={store.logo_url} alt={store.name} className="w-full h-full object-cover" />
            ) : (
              initial
            )}
          </div>
          <h3 className="font-bold text-gray-900 group-hover:text-[#0071CE] transition-colors">{store.name}</h3>
          <p className="text-xs text-gray-500 mt-1 line-clamp-2 leading-relaxed">{store.description}</p>
          <div className="flex items-center gap-1 mt-3 text-xs text-gray-400">
            <Eye size={12} />
            <span>{store.views?.toLocaleString()} views</span>
          </div>
        </div>
      </div>
    </Link>
  );
}

export default function StoreListPage() {
  const { data: stores, isLoading } = useQuery({
    queryKey: ["stores"],
    queryFn: () => api.get("/stores?limit=50").then((r) => r.data.data),
    retry: false,
  });

  const displayStores = stores?.length ? stores : MOCK_STORES;

  return (
    <div className="max-w-7xl mx-auto px-4 py-10">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 flex items-center gap-2">
            <Store size={24} className="text-[#0071CE]" /> Seller Storefronts
          </h1>
          <p className="text-gray-500 text-sm mt-1">Discover trusted sellers across the GCC marketplace</p>
        </div>
        <Link
          href="/my-store"
          className="bg-[#0071CE] text-white text-sm font-bold px-4 py-2.5 rounded-xl hover:bg-[#005BA1] transition-colors flex items-center gap-1.5"
        >
          <Store size={14} /> My Storefront
        </Link>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="bg-white rounded-2xl overflow-hidden shadow-sm">
              <div className="h-24 bg-gray-100 animate-pulse" />
              <div className="p-4 space-y-2">
                <div className="h-5 bg-gray-100 rounded animate-pulse w-3/4" />
                <div className="h-4 bg-gray-100 rounded animate-pulse" />
                <div className="h-4 bg-gray-100 rounded animate-pulse w-2/3" />
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
          {displayStores.map((store: any) => (
            <StoreCard key={store.id} store={store} />
          ))}
        </div>
      )}
    </div>
  );
}
