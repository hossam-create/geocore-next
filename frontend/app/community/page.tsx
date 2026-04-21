import Link from 'next/link';
import { Users, MessageCircle, Star, Plane, Wrench, Lightbulb, MessageSquare, ArrowRight } from 'lucide-react';

const CATEGORIES = [
  { icon: Users, title: 'New Member Introductions', desc: 'Say hello and tell us about yourself.', count: '1.2k posts' },
  { icon: Star, title: 'Seller Tips & Tricks', desc: 'Share what works for your store.', count: '3.4k posts' },
  { icon: MessageCircle, title: 'Buyer Stories', desc: 'Great deals, amazing finds, and shopping wins.', count: '2.1k posts' },
  { icon: Plane, title: 'Crowdshipping Experiences', desc: 'Traveler stories and delivery tips.', count: '890 posts' },
  { icon: Wrench, title: 'Technical Help', desc: 'Bugs, features, and platform questions.', count: '1.5k posts' },
  { icon: Lightbulb, title: 'Feature Requests', desc: 'Suggest new features and vote on ideas.', count: '670 posts' },
  { icon: MessageSquare, title: 'Off-Topic / General', desc: 'Anything goes — within reason.', count: '4.2k posts' },
];

export default function CommunityPage() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <div className="mb-8 text-center">
        <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-violet-100 px-4 py-1.5 text-sm font-medium text-violet-700">
          <Users size={16} /> Community
        </div>
        <h1 className="text-3xl font-extrabold text-gray-900">The Mnbarh Community</h1>
        <p className="mt-2 text-sm text-gray-500">Connect with buyers, sellers, and travelers from across the Arab world.</p>
      </div>

      <section className="mb-10">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {CATEGORIES.map((c) => (
            <div key={c.title} className="rounded-2xl border border-gray-200 bg-white p-5 hover:border-violet-300 transition-colors cursor-pointer">
              <c.icon size={20} className="mb-2 text-violet-600" />
              <h3 className="text-sm font-bold text-gray-900">{c.title}</h3>
              <p className="mt-1 text-xs text-gray-600">{c.desc}</p>
              <span className="mt-2 inline-block text-xs text-gray-400">{c.count}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="mb-10 rounded-2xl border border-violet-200 bg-violet-50 p-6 text-center">
        <h3 className="text-sm font-bold text-violet-800">Join the Conversation</h3>
        <p className="mt-1 text-sm text-violet-700">Sign in with your Mnbarh account to post and reply.</p>
        <Link href="/register" className="mt-3 inline-flex items-center gap-2 rounded-full bg-violet-600 px-6 py-2.5 text-xs font-bold text-white hover:bg-violet-700">
          Create Account <ArrowRight size={14} />
        </Link>
      </section>
    </div>
  );
}
