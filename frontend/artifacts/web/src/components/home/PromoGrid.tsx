import { Link } from "wouter";
import { Tag, Zap, Trophy, Clock } from "lucide-react";

const PROMOS = [
  {
    id: 1,
    title: "Flash Deals",
    subtitle: "Up to 70% off",
    icon: Zap,
    href: "/listings?sort=price_asc",
    bg: "from-blue-600 to-blue-500",
    imgSeed: "flash-deal-sale-blue",
  },
  {
    id: 2,
    title: "New Arrivals",
    subtitle: "Fresh listings daily",
    icon: Tag,
    href: "/listings?sort=newest",
    bg: "from-violet-600 to-violet-500",
    imgSeed: "new-arrival-fresh",
  },
  {
    id: 3,
    title: "Top Rated",
    subtitle: "Trusted sellers",
    icon: Trophy,
    href: "/listings?is_featured=true",
    bg: "from-amber-500 to-yellow-400",
    imgSeed: "top-rated-winner",
  },
  {
    id: 4,
    title: "Ending Soon",
    subtitle: "Auctions closing soon",
    icon: Clock,
    href: "/auctions",
    bg: "from-rose-600 to-red-500",
    imgSeed: "auction-ending-soon",
  },
];

export function PromoGrid() {
  return (
    <section>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {PROMOS.map((promo) => {
          const Icon = promo.icon;
          return (
            <Link
              key={promo.id}
              href={promo.href}
              className={`relative overflow-hidden rounded-2xl bg-gradient-to-br ${promo.bg} text-white p-5 flex flex-col gap-2 group min-h-[130px] shadow-md hover:shadow-xl transition-all duration-200 hover:-translate-y-0.5`}
            >
              <Icon size={26} className="opacity-90" />
              <div>
                <p className="font-extrabold text-lg leading-tight">{promo.title}</p>
                <p className="text-sm opacity-85 mt-0.5">{promo.subtitle}</p>
              </div>
              <div className="absolute -bottom-4 -right-4 w-24 h-24 rounded-full bg-white/10 group-hover:scale-125 transition-transform duration-500" />
              <div className="absolute -top-3 -right-3 w-16 h-16 rounded-full bg-white/10 group-hover:scale-110 transition-transform duration-700" />
            </Link>
          );
        })}
      </div>
    </section>
  );
}
