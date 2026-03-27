import { Link } from "wouter";
import { Facebook, Twitter, Instagram, Youtube } from "lucide-react";

const FOOTER_COLS = [
  {
    title: "Buy",
    links: [
      { label: "Registration", href: "/register" },
      { label: "Bidding & buying help", href: "/auctions" },
      { label: "Brand Outlet", href: "/brand-outlet" },
      { label: "Seasonal sales & events", href: "/listings" },
      { label: "GeoCore Gift Cards", href: "#" },
    ],
  },
  {
    title: "Sell",
    links: [
      { label: "Start selling", href: "/sell" },
      { label: "How to sell", href: "/sell" },
      { label: "Business sellers", href: "/stores" },
      { label: "Seller center", href: "/dashboard" },
      { label: "Affiliates", href: "#" },
    ],
  },
  {
    title: "Tools & apps",
    links: [
      { label: "AI Search", href: "/search" },
      { label: "Mobile app", href: "#" },
      { label: "Security center", href: "#" },
      { label: "Site map", href: "/listings" },
    ],
  },
  {
    title: "About GeoCore",
    links: [
      { label: "Company info", href: "#" },
      { label: "News", href: "#" },
      { label: "Investors", href: "#" },
      { label: "Careers", href: "#" },
      { label: "Diversity & Inclusion", href: "#" },
      { label: "Advertise with us", href: "#" },
      { label: "Policies", href: "#" },
    ],
  },
  {
    title: "Help & Contact",
    links: [
      { label: "Seller Center", href: "/dashboard" },
      { label: "Contact Us", href: "#" },
      { label: "Returns", href: "#" },
      { label: "Money Back Guarantee", href: "#" },
    ],
  },
  {
    title: "Community",
    links: [
      { label: "Announcements", href: "#" },
      { label: "Community forum", href: "#" },
      { label: "GeoCore for Business", href: "#" },
    ],
  },
];

const BOTTOM_LINKS = [
  "All Departments", "Accessibility", "User Agreement", "Privacy Notice",
  "Cookies", "Contact Us", "Brand Outlet", "Sell on GeoCore",
  "Safety Center", "Site Map", "Delete Account",
];

export function Footer() {
  return (
    <footer className="bg-[#0071CE] mt-16">

      {/* Main columns */}
      <div className="max-w-7xl mx-auto px-4 py-10">
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-8 mb-10">
          {FOOTER_COLS.map((col) => (
            <div key={col.title}>
              <h4 className="text-sm font-semibold text-white mb-3">{col.title}</h4>
              <ul className="space-y-2">
                {col.links.map((link) => (
                  <li key={link.label}>
                    <Link
                      href={link.href}
                      className="text-sm text-white hover:text-[#FFC220] transition-colors"
                    >
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        {/* Stay connected + Sites row */}
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-6 pt-6 border-t border-white/10">
          <div>
            <h4 className="text-sm font-semibold text-white mb-3">Stay connected</h4>
            <div className="flex gap-3">
              {/* Facebook */}
              <a href="#" title="Facebook"
                className="w-9 h-9 rounded-full flex items-center justify-center bg-[#1877F2] hover:opacity-90 transition-opacity shadow-sm">
                <Facebook size={17} className="text-white" />
              </a>
              {/* X / Twitter */}
              <a href="#" title="X"
                className="w-9 h-9 rounded-full flex items-center justify-center bg-black hover:opacity-80 transition-opacity shadow-sm">
                <Twitter size={17} className="text-white" />
              </a>
              {/* Instagram — gradient background */}
              <a href="#" title="Instagram"
                className="w-9 h-9 rounded-full flex items-center justify-center hover:opacity-90 transition-opacity shadow-sm"
                style={{ background: "radial-gradient(circle at 30% 107%, #fdf497 0%, #fdf497 5%, #fd5949 45%, #d6249f 60%, #285AEB 90%)" }}>
                <Instagram size={17} className="text-white" />
              </a>
              {/* YouTube */}
              <a href="#" title="YouTube"
                className="w-9 h-9 rounded-full flex items-center justify-center bg-[#FF0000] hover:opacity-90 transition-opacity shadow-sm">
                <Youtube size={17} className="text-white" />
              </a>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <span className="text-sm font-semibold text-white">GeoCore Sites:</span>
            <select className="text-sm border border-white/20 rounded-lg px-3 py-1.5 text-white bg-white/10 focus:outline-none focus:ring-2 focus:ring-[#FFC220]">
              <option className="text-gray-900">🇦🇪 United Arab Emirates</option>
              <option className="text-gray-900">🇸🇦 Saudi Arabia</option>
              <option className="text-gray-900">🇰🇼 Kuwait</option>
              <option className="text-gray-900">🇶🇦 Qatar</option>
              <option className="text-gray-900">🇧🇭 Bahrain</option>
              <option className="text-gray-900">🇴🇲 Oman</option>
            </select>
          </div>
        </div>
      </div>

      {/* Bottom bar */}
      <div className="border-t border-white/20 bg-[#005aab]">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex flex-wrap justify-center gap-x-4 gap-y-1 mb-3">
            {BOTTOM_LINKS.map((t) => (
              <a key={t} href="#" className="text-xs text-white hover:text-[#FFC220] transition-colors">
                {t}
              </a>
            ))}
          </div>
          <p className="text-center text-xs text-white/90">
            © {new Date().getFullYear()} GeoCore Inc. All rights reserved. · GCC Marketplace · The GeoCore name, logo, and related marks are trademarks of GeoCore Inc.
          </p>
        </div>
      </div>
    </footer>
  );
}
