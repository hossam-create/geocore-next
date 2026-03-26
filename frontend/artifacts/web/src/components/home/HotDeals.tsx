import { Link } from "wouter";
import { Flame, Clock } from "lucide-react";

const DEALS = [
  {
    seed: "deal-electronics",
    label: "Electronics",
    headline: "Up to 40% Off",
    sub: "Phones, Laptops & More",
    from: "#6366F1",
    to: "#8B5CF6",
    href: "/listings?category=electronics",
  },
  {
    seed: "deal-vehicles",
    label: "Vehicles",
    headline: "Best Car Deals",
    sub: "New & Used · UAE & KSA",
    from: "#0071CE",
    to: "#0284C7",
    href: "/listings?category=vehicles",
  },
  {
    seed: "deal-fashion",
    label: "Fashion",
    headline: "Luxury & Brands",
    sub: "Designer Labels at Less",
    from: "#EC4899",
    to: "#DB2777",
    href: "/listings?category=clothing",
  },
  {
    seed: "deal-realestate",
    label: "Real Estate",
    headline: "Prime Locations",
    sub: "Dubai · Riyadh · Doha",
    from: "#10B981",
    to: "#059669",
    href: "/listings?category=real-estate",
  },
];

export function HotDeals() {
  return (
    <section>
      <div className="flex items-center justify-between mb-5">
        <h2 className="text-xl font-bold text-gray-900 flex items-center gap-2">
          <Flame size={20} className="text-orange-500" />
          Today's Deals
        </h2>
        <Link href="/listings?sort=price_asc" className="text-[#0071CE] text-sm font-semibold hover:underline flex items-center gap-1">
          <Clock size={13} />
          See all deals →
        </Link>
      </div>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {DEALS.map((deal) => (
          <Link
            key={deal.seed}
            href={deal.href}
            className="group relative rounded-2xl overflow-hidden shadow-sm hover:shadow-xl transition-all duration-300 hover:-translate-y-1 aspect-[4/3]"
            style={{ background: `linear-gradient(135deg, ${deal.from}, ${deal.to})` }}
          >
            <img
              src={`https://picsum.photos/seed/${deal.seed}/400/300`}
              alt={deal.label}
              className="absolute inset-0 w-full h-full object-cover opacity-20 group-hover:opacity-30 group-hover:scale-105 transition-all duration-500"
            />
            <div className="relative z-10 p-4 h-full flex flex-col justify-between">
              <span className="text-white/80 text-[11px] font-bold uppercase tracking-widest">
                {deal.label}
              </span>
              <div>
                <p className="text-white text-xl font-black leading-tight">{deal.headline}</p>
                <p className="text-white/70 text-xs mt-1 font-medium">{deal.sub}</p>
                <span className="mt-3 inline-block bg-white text-gray-900 text-xs font-bold px-3 py-1.5 rounded-full group-hover:bg-[#FFC220] transition-colors">
                  Shop Now →
                </span>
              </div>
            </div>
          </Link>
        ))}
      </div>
    </section>
  );
}
