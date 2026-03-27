import { Link } from "wouter";

export function Footer() {
  return (
    <footer className="bg-gray-900 text-gray-400 mt-16">
      <div className="max-w-7xl mx-auto px-4 py-12 grid grid-cols-2 md:grid-cols-4 gap-8">
        <div>
          <h3 className="text-white font-bold text-lg mb-3">
            <span className="text-[#FFC220]">Geo</span>Core
          </h3>
          <p className="text-sm leading-relaxed">
            The GCC region's premier marketplace for buying, selling, and bidding.
          </p>
        </div>
        <div>
          <h4 className="text-white font-semibold mb-3">Marketplace</h4>
          <ul className="space-y-2 text-sm">
            <li><Link href="/listings" className="hover:text-white transition-colors">Browse Listings</Link></li>
            <li><Link href="/auctions" className="hover:text-white transition-colors">Live Auctions</Link></li>
            <li><Link href="/sell" className="hover:text-white transition-colors">Sell an Item</Link></li>
          </ul>
        </div>
        <div>
          <h4 className="text-white font-semibold mb-3">Account</h4>
          <ul className="space-y-2 text-sm">
            <li><Link href="/login" className="hover:text-white transition-colors">Sign In</Link></li>
            <li><Link href="/register" className="hover:text-white transition-colors">Register</Link></li>
            <li><Link href="/profile" className="hover:text-white transition-colors">My Profile</Link></li>
          </ul>
        </div>
        <div>
          <h4 className="text-white font-semibold mb-3">Support</h4>
          <ul className="space-y-2 text-sm">
            <li><a href="#" className="hover:text-white transition-colors">Help Center</a></li>
            <li><a href="#" className="hover:text-white transition-colors">Privacy Policy</a></li>
            <li><a href="#" className="hover:text-white transition-colors">Contact Us</a></li>
          </ul>
        </div>
      </div>
      <div className="border-t border-gray-800 py-4 text-center text-xs">
        © {new Date().getFullYear()} GeoCore. All rights reserved. · GCC Region
      </div>
    </footer>
  );
}
