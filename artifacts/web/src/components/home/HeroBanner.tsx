import { Link } from "wouter";

export function HeroBanner() {
  return (
    <div className="bg-gradient-to-br from-[#0071CE] via-[#005BA1] to-[#003f75] text-white py-14 px-4">
      <div className="max-w-7xl mx-auto flex flex-col md:flex-row items-center justify-between gap-10">
        <div className="max-w-xl">
          <h1 className="text-4xl md:text-5xl font-extrabold leading-tight">
            Buy. Sell.{" "}
            <span className="text-[#FFC220]">Bid.</span>
          </h1>
          <p className="text-lg text-blue-100 mt-4 leading-relaxed">
            Millions of listings across the GCC region. Real-time auctions. Instant deals.
          </p>
          <div className="flex gap-3 mt-8 flex-wrap">
            <Link
              href="/listings"
              className="bg-[#FFC220] text-gray-900 px-7 py-3.5 rounded-xl font-bold text-sm hover:bg-yellow-400 transition-colors shadow-lg"
            >
              Browse Listings
            </Link>
            <Link
              href="/auctions"
              className="border-2 border-white text-white px-7 py-3.5 rounded-xl font-bold text-sm hover:bg-white hover:text-[#0071CE] transition-colors"
            >
              ⚡ Live Auctions
            </Link>
            <Link
              href="/sell"
              className="border-2 border-[#FFC220] text-[#FFC220] px-7 py-3.5 rounded-xl font-bold text-sm hover:bg-[#FFC220] hover:text-gray-900 transition-colors"
            >
              + Post Listing
            </Link>
          </div>

          <div className="flex gap-8 mt-10">
            {[
              { label: "Active Listings", value: "2.4M+" },
              { label: "Live Auctions", value: "18K+" },
              { label: "GCC Cities", value: "50+" },
            ].map((s) => (
              <div key={s.label}>
                <p className="text-2xl font-extrabold text-[#FFC220]">{s.value}</p>
                <p className="text-xs text-blue-200">{s.label}</p>
              </div>
            ))}
          </div>
        </div>

        <div className="text-9xl hidden md:block select-none drop-shadow-2xl">🌍</div>
      </div>
    </div>
  );
}
