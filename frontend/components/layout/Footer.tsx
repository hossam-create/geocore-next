'use client'
import Link from 'next/link';
import { Facebook, Twitter, Instagram, Youtube } from "lucide-react";
import { useTranslations } from "next-intl";

const FOOTER_COLS: { titleKey: string; links: { labelKey: string; href: string }[] }[] = [
  {
    titleKey: "buy",
    links: [
      { labelKey: "registration", href: "/register" },
      { labelKey: "biddingHelp", href: "/auctions" },
      { labelKey: "buyerProtection", href: "/buyer-protection" },
      { labelKey: "brandOutlet", href: "/brand-outlet" },
      { labelKey: "seasonalSales", href: "/seasonal-sales" },
      { labelKey: "giftCards", href: "/gift-cards" },
    ],
  },
  {
    titleKey: "sell",
    links: [
      { labelKey: "startSelling", href: "/sell" },
      { labelKey: "howToSell", href: "/how-to-sell" },
      { labelKey: "sellerProtection", href: "/seller-protection" },
      { labelKey: "businessSellers", href: "/business-sellers" },
      { labelKey: "sellerCenter", href: "/seller-center" },
      { labelKey: "affiliates", href: "/affiliates" },
    ],
  },
  {
    titleKey: "toolsApps",
    links: [
      { labelKey: "aiSearch", href: "/search" },
      { labelKey: "mobileApp", href: "/mobile-app" },
      { labelKey: "securityCenter", href: "/security-center" },
      { labelKey: "siteMap", href: "/sitemap-page" },
    ],
  },
  {
    titleKey: "aboutMnbarh",
    links: [
      { labelKey: "companyInfo", href: "/about" },
      { labelKey: "news", href: "/news" },
      { labelKey: "investors", href: "/investors" },
      { labelKey: "careers", href: "/careers" },
      { labelKey: "diversity", href: "/diversity" },
      { labelKey: "advertise", href: "/advertise" },
      { labelKey: "termsOfService", href: "/legal/terms" },
      { labelKey: "privacyPolicy", href: "/legal/privacy" },
      { labelKey: "cookiePolicy", href: "/legal/cookies" },
    ],
  },
  {
    titleKey: "helpContact",
    links: [
      { labelKey: "helpCenter", href: "/help" },
      { labelKey: "faq", href: "/help/faq" },
      { labelKey: "sellerCenter", href: "/seller-center" },
      { labelKey: "contactUs", href: "/contact" },
      { labelKey: "returns", href: "/returns" },
      { labelKey: "refundPolicy", href: "/refund-policy" },
    ],
  },
  {
    titleKey: "community",
    links: [
      { labelKey: "announcements", href: "/announcements" },
      { labelKey: "communityForum", href: "/community" },
      { labelKey: "mnbarhBusiness", href: "/business" },
    ],
  },
];

const BOTTOM_LINKS: { labelKey: string; href: string }[] = [
  { labelKey: "allDepartments", href: "/listings" },
  { labelKey: "howItWorks", href: "/how-it-works" },
  { labelKey: "helpCenter", href: "/help" },
  { labelKey: "about", href: "/about" },
  { labelKey: "shipping", href: "/shipping" },
  { labelKey: "feeCalculator", href: "/fees" },
  { labelKey: "accessibility", href: "/accessibility" },
  { labelKey: "userAgreement", href: "/legal/terms" },
  { labelKey: "privacyNotice", href: "/legal/privacy" },
  { labelKey: "cookies", href: "/legal/cookies" },
  { labelKey: "refundPolicy", href: "/refund-policy" },
  { labelKey: "contactUs", href: "/contact" },
  { labelKey: "brandOutlet", href: "/brand-outlet" },
  { labelKey: "sellOnMnbarh", href: "/sell" },
  { labelKey: "safetyCenter", href: "/safety-center" },
  { labelKey: "siteMap", href: "/sitemap-page" },
  { labelKey: "deleteAccount", href: "/delete-account" },
];

export function Footer() {
  const t = useTranslations("footer");
  return (
    <footer className="bg-[#0071CE] mt-16">

      {/* Main columns */}
      <div className="max-w-7xl mx-auto px-4 py-10">
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-8 mb-10">
          {FOOTER_COLS.map((col) => (
            <div key={col.titleKey}>
              <h4 className="text-sm font-semibold text-white mb-3">{t(col.titleKey)}</h4>
              <ul className="space-y-2">
                {col.links.map((link) => (
                  <li key={link.labelKey}>
                    <Link
                      href={link.href}
                      className="text-sm text-white hover:text-[#FFC220] transition-colors"
                    >
                      {t(link.labelKey)}
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
            <h4 className="text-sm font-semibold text-white mb-3">{t("stayConnected")}</h4>
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
            <span className="text-sm font-semibold text-white">{t("mnbarhSites")}</span>
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
            {BOTTOM_LINKS.map((link) => (
              <Link key={link.labelKey} href={link.href} className="text-xs text-white hover:text-[#FFC220] transition-colors">
                {t(link.labelKey)}
              </Link>
            ))}
          </div>
          <p className="text-center text-xs text-white/90">
            {t("copyright", { year: new Date().getFullYear() })}
          </p>
        </div>
      </div>
    </footer>
  );
}
