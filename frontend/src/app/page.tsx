import Link from 'next/link'
import { Search, Tag, Gavel, MessageCircle, Shield, Globe } from 'lucide-react'

export default function HomePage() {
  return (
    <main className="min-h-screen bg-gray-950 text-white">
      {/* Hero */}
      <section className="relative overflow-hidden bg-gradient-to-br from-gray-900 via-blue-950 to-gray-900 py-24">
        <div className="container mx-auto px-4 text-center">
          <h1 className="text-5xl md:text-7xl font-black mb-6 bg-gradient-to-r from-white via-blue-400 to-cyan-400 bg-clip-text text-transparent">
            GeoCore Next
          </h1>
          <p className="text-xl text-gray-400 mb-10 max-w-2xl mx-auto">
            The modern global marketplace for classifieds and real-time auctions.
            Buy, sell, bid — anywhere in the world.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Link href="/listings" className="px-8 py-4 bg-blue-600 hover:bg-blue-500 rounded-xl font-bold text-lg transition-all hover:scale-105">
              Browse Listings
            </Link>
            <Link href="/auctions" className="px-8 py-4 bg-white/10 hover:bg-white/20 border border-white/20 rounded-xl font-bold text-lg transition-all hover:scale-105">
              Live Auctions
            </Link>
          </div>
        </div>
      </section>

      {/* Search Bar */}
      <section className="bg-gray-900 py-8 border-b border-gray-800">
        <div className="container mx-auto px-4">
          <div className="flex gap-3 max-w-3xl mx-auto">
            <div className="flex-1 flex items-center gap-3 bg-gray-800 border border-gray-700 rounded-xl px-4">
              <Search className="text-gray-400 w-5 h-5" />
              <input
                type="text"
                placeholder="Search listings, auctions..."
                className="flex-1 bg-transparent py-4 outline-none text-white placeholder-gray-500"
              />
            </div>
            <button className="px-6 py-4 bg-blue-600 hover:bg-blue-500 rounded-xl font-semibold transition-colors">
              Search
            </button>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="py-20 bg-gray-950">
        <div className="container mx-auto px-4">
          <h2 className="text-3xl font-bold text-center mb-12">Why GeoCore Next?</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {[
              { icon: Tag, title: 'Free Listings', desc: 'Post your items for free. Reach buyers worldwide instantly.' },
              { icon: Gavel, title: 'Live Auctions', desc: 'Real-time bidding with instant notifications and auto-bid.' },
              { icon: MessageCircle, title: 'Secure Chat', desc: 'Message buyers and sellers directly, safely.' },
              { icon: Shield, title: 'Verified Users', desc: 'Phone and ID verification for trusted transactions.' },
              { icon: Globe, title: '100+ Countries', desc: 'Multi-currency, multi-language global marketplace.' },
              { icon: Search, title: 'Smart Search', desc: 'AI-powered search understands what you are looking for.' },
            ].map((f) => (
              <div key={f.title} className="bg-gray-900 border border-gray-800 rounded-2xl p-6 hover:border-blue-500/40 transition-colors">
                <f.icon className="w-10 h-10 text-blue-400 mb-4" />
                <h3 className="text-lg font-bold mb-2">{f.title}</h3>
                <p className="text-gray-400 text-sm leading-relaxed">{f.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="py-20 bg-gradient-to-r from-blue-900 to-cyan-900">
        <div className="container mx-auto px-4 text-center">
          <h2 className="text-4xl font-black mb-4">Ready to sell?</h2>
          <p className="text-gray-300 mb-8 text-lg">Post your first listing in under 2 minutes.</p>
          <Link href="/listings/create" className="px-10 py-5 bg-white text-gray-900 hover:bg-gray-100 rounded-xl font-bold text-lg transition-all hover:scale-105 inline-block">
            Post Free Ad
          </Link>
        </div>
      </section>
    </main>
  )
}
