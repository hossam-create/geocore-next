import { Link } from "wouter";

const CATEGORIES = [
  { emoji: "🚗", label: "Vehicles", slug: "vehicles" },
  { emoji: "🏠", label: "Real Estate", slug: "real-estate" },
  { emoji: "📱", label: "Electronics", slug: "electronics" },
  { emoji: "👕", label: "Fashion", slug: "clothing" },
  { emoji: "🛋️", label: "Furniture", slug: "furniture" },
  { emoji: "💎", label: "Jewelry", slug: "jewelry" },
  { emoji: "🔧", label: "Tools", slug: "tools" },
  { emoji: "🎮", label: "Gaming", slug: "gaming" },
  { emoji: "📚", label: "Books", slug: "books" },
  { emoji: "🏋️", label: "Sports", slug: "sports" },
];

export function CategorySection() {
  return (
    <section>
      <h2 className="text-2xl font-bold text-gray-900 mb-5">Shop by Category</h2>
      <div className="grid grid-cols-5 md:grid-cols-10 gap-3">
        {CATEGORIES.map((cat) => (
          <Link
            key={cat.slug}
            href={`/listings?category=${cat.slug}`}
            className="flex flex-col items-center gap-2 p-3 bg-white rounded-xl shadow-sm hover:shadow-md hover:-translate-y-0.5 transition-all group cursor-pointer"
          >
            <span className="text-3xl group-hover:scale-110 transition-transform">{cat.emoji}</span>
            <span className="text-xs text-gray-600 font-medium text-center leading-tight">{cat.label}</span>
          </Link>
        ))}
      </div>
    </section>
  );
}
