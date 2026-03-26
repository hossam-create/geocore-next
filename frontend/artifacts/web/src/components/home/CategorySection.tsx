import { Link } from "wouter";
import { ChevronRight } from "lucide-react";

const CATEGORIES = [
  { seed: "laptop-notebook-dell", label: "Laptops & Computers", slug: "electronics", count: "12K+" },
  { seed: "iphone15-promax-gcc", label: "Smartphones", slug: "electronics", count: "8K+" },
  { seed: "car-vehicle-suv", label: "Vehicles", slug: "vehicles", count: "45K+" },
  { seed: "apartment-dubai-luxury", label: "Real Estate", slug: "real-estate", count: "30K+" },
  { seed: "fashion-dress-abaya", label: "Fashion & Clothing", slug: "clothing", count: "20K+" },
  { seed: "rolex-luxury-watch", label: "Jewelry & Watches", slug: "jewelry", count: "5K+" },
  { seed: "sofa-living-room", label: "Furniture & Home", slug: "furniture", count: "10K+" },
  { seed: "gaming-ps5-controller", label: "Gaming", slug: "gaming", count: "4K+" },
];

export function CategorySection() {
  return (
    <section>
      <div className="flex items-center justify-between mb-5">
        <h2 className="text-xl font-bold text-gray-900">Shop by Department</h2>
        <Link href="/listings" className="text-[#0071CE] text-sm font-semibold hover:underline flex items-center gap-1">
          All departments <ChevronRight size={14} />
        </Link>
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-8 gap-3">
        {CATEGORIES.map((cat) => (
          <Link
            key={cat.slug + cat.seed}
            href={`/listings?category=${cat.slug}`}
            className="group bg-white rounded-2xl overflow-hidden shadow-sm hover:shadow-lg transition-all duration-200 hover:-translate-y-0.5 border border-gray-100 hover:border-[#0071CE]/20 cursor-pointer"
          >
            <div className="aspect-square overflow-hidden bg-gray-50">
              <img
                src={`https://picsum.photos/seed/${cat.seed}/200/200`}
                alt={cat.label}
                className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
              />
            </div>
            <div className="p-2.5">
              <p className="text-xs font-semibold text-gray-800 group-hover:text-[#0071CE] transition-colors leading-tight line-clamp-2">
                {cat.label}
              </p>
              <p className="text-[10px] text-gray-400 mt-0.5">{cat.count} listings</p>
            </div>
          </Link>
        ))}
      </div>
    </section>
  );
}
