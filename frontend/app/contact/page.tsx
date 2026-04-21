'use client';
import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useAuthStore } from '@/store/auth';
import api from '@/lib/api';
import {
  MessageSquare, Send, CheckCircle, Loader2, Mail,
  User, HelpCircle, ChevronDown, ArrowLeft
} from 'lucide-react';

interface SubjectOption {
  value: string;
  label: string;
}

export default function ContactPage() {
  const { user, isAuthenticated } = useAuthStore();
  const [mounted, setMounted] = useState(false);
  const [subjects, setSubjects] = useState<SubjectOption[]>([]);
  const [form, setForm] = useState({
    name: '',
    email: '',
    subject: '',
    message: '',
  });
  const [loading, setLoading] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    // Pre-fill form if user is logged in
    if (isAuthenticated && user) {
      setForm((f) => ({
        ...f,
        name: user.name || f.name,
        email: user.email || f.email,
      }));
    }
  }, [isAuthenticated, user]);

  useEffect(() => {
    const fetchSubjects = async () => {
      try {
        const res = await api.get('/support/subjects');
        setSubjects(res.data || []);
      } catch {
        // Use defaults
        setSubjects([
          { value: 'general', label: 'General Inquiry' },
          { value: 'order', label: 'Order Issue' },
          { value: 'payment', label: 'Payment Problem' },
          { value: 'account', label: 'Account Help' },
          { value: 'selling', label: 'Selling Question' },
          { value: 'technical', label: 'Technical Support' },
          { value: 'feedback', label: 'Feedback' },
          { value: 'other', label: 'Other' },
        ]);
      }
    };
    fetchSubjects();
  }, []);

  if (!mounted) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (form.message.length < 20) {
      setError('Message must be at least 20 characters');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await api.post('/support/contact', form);
      setSubmitted(true);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to submit message');
    } finally {
      setLoading(false);
    }
  };

  if (submitted) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center px-4">
        <div className="max-w-md w-full bg-white rounded-2xl shadow-sm p-8 text-center">
          <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <CheckCircle size={32} className="text-green-600" />
          </div>
          <h1 className="text-2xl font-bold text-gray-900 mb-2">Message Sent!</h1>
          <p className="text-gray-500 mb-6">
            Thank you for reaching out. Our support team will get back to you within 24-48 hours.
          </p>
          <Link
            href="/"
            className="inline-flex items-center gap-2 bg-[#0071CE] text-white px-6 py-3 rounded-xl font-semibold hover:bg-[#005ba3] transition-colors"
          >
            Back to Home
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-[#0071CE] text-white py-10 px-4">
        <div className="max-w-2xl mx-auto text-center">
          <div className="inline-flex items-center gap-2 bg-white/20 rounded-full px-4 py-1.5 text-sm mb-4">
            <HelpCircle size={16} />
            <span>Support</span>
          </div>
          <h1 className="text-3xl font-bold mb-2">Contact Us</h1>
          <p className="text-blue-100">
            Have a question or need help? We're here for you.
          </p>
        </div>
      </div>

      <div className="max-w-2xl mx-auto px-4 py-8">
        {/* Back link */}
        <Link href="/" className="inline-flex items-center gap-2 text-sm text-gray-500 hover:text-gray-700 mb-6">
          <ArrowLeft size={16} />
          Back to Home
        </Link>

        {/* Form */}
        <div className="bg-white rounded-2xl shadow-sm p-6 md:p-8">
          <form onSubmit={handleSubmit} className="space-y-5">
            {/* Name */}
            <div>
              <label className="text-sm font-medium text-gray-700 block mb-1.5">
                Your Name
              </label>
              <div className="relative">
                <User size={18} className="absolute left-4 top-1/2 -translate-y-1/2 text-gray-400" />
                <input
                  type="text"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  required
                  placeholder="John Doe"
                  className="w-full border border-gray-200 rounded-xl pl-11 pr-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
                />
              </div>
            </div>

            {/* Email */}
            <div>
              <label className="text-sm font-medium text-gray-700 block mb-1.5">
                Email Address
              </label>
              <div className="relative">
                <Mail size={18} className="absolute left-4 top-1/2 -translate-y-1/2 text-gray-400" />
                <input
                  type="email"
                  value={form.email}
                  onChange={(e) => setForm({ ...form, email: e.target.value })}
                  required
                  placeholder="john@example.com"
                  className="w-full border border-gray-200 rounded-xl pl-11 pr-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
                />
              </div>
            </div>

            {/* Subject */}
            <div>
              <label className="text-sm font-medium text-gray-700 block mb-1.5">
                Subject
              </label>
              <div className="relative">
                <select
                  value={form.subject}
                  onChange={(e) => setForm({ ...form, subject: e.target.value })}
                  required
                  className="w-full border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE] appearance-none bg-white"
                >
                  <option value="">Select a topic</option>
                  {subjects.map((s) => (
                    <option key={s.value} value={s.value}>{s.label}</option>
                  ))}
                </select>
                <ChevronDown size={18} className="absolute right-4 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none" />
              </div>
            </div>

            {/* Message */}
            <div>
              <label className="text-sm font-medium text-gray-700 block mb-1.5">
                Message
              </label>
              <div className="relative">
                <textarea
                  value={form.message}
                  onChange={(e) => setForm({ ...form, message: e.target.value })}
                  required
                  minLength={20}
                  rows={6}
                  placeholder="Describe your issue or question in detail (minimum 20 characters)..."
                  className="w-full border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE] resize-none"
                />
                <span className={`absolute bottom-3 right-3 text-xs ${form.message.length >= 20 ? 'text-green-600' : 'text-gray-400'}`}>
                  {form.message.length}/20 min
                </span>
              </div>
            </div>

            {error && (
              <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
                {error}
              </div>
            )}

            {/* Submit */}
            <button
              type="submit"
              disabled={loading}
              className="w-full flex items-center justify-center gap-2 bg-[#0071CE] text-white py-3.5 rounded-xl font-semibold hover:bg-[#005ba3] disabled:bg-gray-300 transition-colors"
            >
              {loading ? (
                <>
                  <Loader2 size={18} className="animate-spin" />
                  Sending...
                </>
              ) : (
                <>
                  <Send size={18} />
                  Send Message
                </>
              )}
            </button>
          </form>
        </div>

        {/* Help links */}
        <div className="mt-8 grid gap-4 sm:grid-cols-2">
          <Link href="/help" className="flex items-center gap-3 bg-white rounded-xl p-4 border border-gray-200 hover:shadow-sm transition-shadow">
            <div className="w-10 h-10 bg-blue-50 rounded-lg flex items-center justify-center text-[#0071CE]">
              <HelpCircle size={20} />
            </div>
            <div>
              <p className="text-sm font-semibold text-gray-900">Help Center</p>
              <p className="text-xs text-gray-500">Browse FAQs and guides</p>
            </div>
          </Link>
          <Link href="/help/faq" className="flex items-center gap-3 bg-white rounded-xl p-4 border border-gray-200 hover:shadow-sm transition-shadow">
            <div className="w-10 h-10 bg-purple-50 rounded-lg flex items-center justify-center text-purple-600">
              <MessageSquare size={20} />
            </div>
            <div>
              <p className="text-sm font-semibold text-gray-900">FAQ</p>
              <p className="text-xs text-gray-500">Quick answers to common questions</p>
            </div>
          </Link>
        </div>
      </div>
    </div>
  );
}
