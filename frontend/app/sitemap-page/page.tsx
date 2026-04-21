import Link from 'next/link';
import { ShoppingBag, Tag, HelpCircle, Building2, Scale, Plane } from 'lucide-react';

const SECTIONS = [
  {
    icon: ShoppingBag,
    title: 'Buy',
    links: [
      { label: 'Listings', href: '/listings' },
      { label: 'Auctions', href: '/auctions' },
      { label: 'Live Auctions', href: '/auctions/live' },
      { label: 'Brand Outlet', href: '/brand-outlet' },
      { label: 'Deals', href: '/deals' },
      { label: 'Seasonal Sales', href: '/seasonal-sales' },
      { label: 'Gift Cards', href: '/gift-cards' },
      { label: 'Buyer Protection', href: '/buyer-protection' },
    ],
  },
  {
    icon: Tag,
    title: 'Sell',
    links: [
      { label: 'Start Selling', href: '/sell' },
      { label: 'How to Sell', href: '/how-to-sell' },
      { label: 'Seller Protection', href: '/seller-protection' },
      { label: 'Seller Center', href: '/seller-center' },
      { label: 'Business Sellers', href: '/business-sellers' },
      { label: 'Affiliates', href: '/affiliates' },
      { label: 'Fee Calculator', href: '/fees' },
    ],
  },
  {
    icon: Plane,
    title: 'Crowdshipping',
    links: [
      { label: 'Buy via Traveler', href: '/buy-via-traveler' },
      { label: 'How It Works', href: '/crowdshipping' },
      { label: 'For Travelers', href: '/crowdshipping/for-travelers' },
      { label: 'Safety & Trust', href: '/crowdshipping/trust' },
      { label: 'Insurance', href: '/crowdshipping/insurance' },
      { label: 'Secure Payment', href: '/crowdshipping/payment' },
      { label: 'Transport Contract', href: '/crowdshipping/contract' },
    ],
  },
  {
    icon: HelpCircle,
    title: 'Help & Contact',
    links: [
      { label: 'Help Center', href: '/help' },
      { label: 'FAQ', href: '/help/faq' },
      { label: 'Contact Us', href: '/contact' },
      { label: 'Returns', href: '/returns' },
      { label: 'Refund Policy', href: '/refund-policy' },
      { label: 'Security Center', href: '/security-center' },
      { label: 'Safety Center', href: '/safety-center' },
    ],
  },
  {
    icon: Building2,
    title: 'About',
    links: [
      { label: 'About Mnbarh', href: '/about' },
      { label: 'Careers', href: '/careers' },
      { label: 'News', href: '/news' },
      { label: 'Investors', href: '/investors' },
      { label: 'Diversity & Inclusion', href: '/diversity' },
      { label: 'Advertise with Us', href: '/advertise' },
      { label: 'Mnbarh for Business', href: '/business' },
      { label: 'Community', href: '/community' },
      { label: 'Announcements', href: '/announcements' },
    ],
  },
  {
    icon: Scale,
    title: 'Legal',
    links: [
      { label: 'Terms of Service', href: '/legal/terms' },
      { label: 'Privacy Policy', href: '/legal/privacy' },
      { label: 'Cookie Policy', href: '/legal/cookies' },
      { label: 'Accessibility', href: '/accessibility' },
      { label: 'Delete Account', href: '/delete-account' },
    ],
  },
];

export default function SitemapPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10">
      <div className="mb-8 text-center">
        <h1 className="text-3xl font-extrabold text-gray-900">Site Map</h1>
        <p className="mt-2 text-sm text-gray-500">All pages on Mnbarh, organized by category.</p>
      </div>

      <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-3">
        {SECTIONS.map((sec) => (
          <div key={sec.title}>
            <div className="mb-3 flex items-center gap-2">
              <sec.icon size={18} className="text-[#0071CE]" />
              <h2 className="text-sm font-bold text-gray-900">{sec.title}</h2>
            </div>
            <ul className="space-y-1.5">
              {sec.links.map((link) => (
                <li key={link.href}>
                  <Link href={link.href} className="text-sm text-[#0071CE] hover:underline">{link.label}</Link>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>
    </div>
  );
}
