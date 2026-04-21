'use client';
import { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { useAuthStore } from '@/store/auth';
import api from '@/lib/api';
import {
  ArrowLeft, MessageSquare, DollarSign, Clock, User,
  Loader2, PackageSearch, Send, CheckCircle, Tag
} from 'lucide-react';

interface Response {
  id: string;
  seller_id: string;
  seller_name: string;
  listing_id?: string;
  message: string;
  created_at: string;
}

interface ProductRequest {
  id: string;
  user_id: string;
  user_name: string;
  title: string;
  description?: string;
  category_name?: string;
  budget?: number;
  currency: string;
  status: string;
  response_count: number;
  created_at: string;
  expires_at?: string;
}

export default function ProductRequestDetailPage() {
  const params = useParams();
  const router = useRouter();
  const { user, isAuthenticated } = useAuthStore();
  const id = params?.id as string;

  const [request, setRequest] = useState<ProductRequest | null>(null);
  const [responses, setResponses] = useState<Response[]>([]);
  const [loading, setLoading] = useState(true);
  const [respondMessage, setRespondMessage] = useState('');
  const [listingId, setListingId] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!id) return;
    api.get(`/api/v1/requests/${id}`)
      .then(res => {
        setRequest(res.data.data);
        setResponses(res.data.responses ?? []);
      })
      .catch(() => router.push('/requests'))
      .finally(() => setLoading(false));
  }, [id, router]);

  const handleRespond = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!respondMessage.trim()) return;
    setSubmitting(true);
    setError('');
    try {
      const payload: Record<string, unknown> = { message: respondMessage.trim() };
      if (listingId.trim()) payload.listing_id = listingId.trim();
      await api.post(`/api/v1/requests/${id}/respond`, payload);
      setSubmitted(true);
      // Reload responses
      const res = await api.get(`/api/v1/requests/${id}`);
      setResponses(res.data.responses ?? []);
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } };
      setError(e?.response?.data?.error ?? 'Failed to submit response');
    } finally {
      setSubmitting(false);
    }
  };

  const formatDate = (d: string) =>
    new Date(d).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });

  const isOwner = user?.id === request?.user_id;
  const alreadyResponded = responses.some(r => r.seller_id === user?.id);

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
      </div>
    );
  }

  if (!request) return null;

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-2xl mx-auto px-4 py-8">
        <Link href="/requests" className="inline-flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 mb-6">
          <ArrowLeft className="w-4 h-4" />
          Back to Requests
        </Link>

        {/* Request card */}
        <div className="bg-white rounded-2xl border border-gray-200 p-6 mb-6">
          <div className="flex items-start justify-between gap-3 mb-4">
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-2">
                <PackageSearch className="w-5 h-5 text-blue-600 flex-shrink-0" />
                <h1 className="text-xl font-bold text-gray-900">{request.title}</h1>
              </div>
              {request.description && (
                <p className="text-sm text-gray-600 mt-2">{request.description}</p>
              )}
            </div>
            <span className={`px-3 py-1 rounded-full text-xs font-semibold flex-shrink-0 ${
              request.status === 'open' ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600'
            }`}>
              {request.status}
            </span>
          </div>

          <div className="flex flex-wrap gap-3 text-xs text-gray-500 pt-4 border-t border-gray-100">
            <span className="flex items-center gap-1">
              <User className="w-3.5 h-3.5" />
              {request.user_name}
            </span>
            {request.category_name && (
              <span className="flex items-center gap-1">
                <Tag className="w-3.5 h-3.5" />
                {request.category_name}
              </span>
            )}
            {request.budget && (
              <span className="flex items-center gap-1 text-green-700 font-medium">
                <DollarSign className="w-3.5 h-3.5" />
                Budget: {new Intl.NumberFormat('en-US', { style: 'currency', currency: request.currency }).format(request.budget)}
              </span>
            )}
            <span className="flex items-center gap-1">
              <Clock className="w-3.5 h-3.5" />
              Posted {formatDate(request.created_at)}
            </span>
            <span className="flex items-center gap-1">
              <MessageSquare className="w-3.5 h-3.5" />
              {request.response_count} {request.response_count === 1 ? 'response' : 'responses'}
            </span>
          </div>
        </div>

        {/* Seller response form */}
        {isAuthenticated && !isOwner && request.status === 'open' && (
          <div className="bg-white rounded-2xl border border-gray-200 p-6 mb-6">
            <h2 className="font-semibold text-gray-900 mb-4">
              {alreadyResponded ? 'Your Response' : 'Respond as a Seller'}
            </h2>

            {submitted || alreadyResponded ? (
              <div className="flex items-center gap-2 p-3 bg-green-50 border border-green-200 rounded-xl text-green-700 text-sm">
                <CheckCircle className="w-4 h-4 flex-shrink-0" />
                {alreadyResponded && !submitted ? 'You have already responded to this request.' : 'Response submitted!'}
              </div>
            ) : (
              <form onSubmit={handleRespond} className="space-y-4">
                {error && (
                  <p className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-xl p-3">{error}</p>
                )}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1.5">Message <span className="text-red-500">*</span></label>
                  <textarea
                    value={respondMessage}
                    onChange={e => setRespondMessage(e.target.value)}
                    placeholder="Describe how you can fulfill this request…"
                    rows={4}
                    className="w-full px-4 py-2.5 border border-gray-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1.5">
                    Listing ID <span className="text-gray-400 font-normal">(optional — link a matching listing)</span>
                  </label>
                  <input
                    value={listingId}
                    onChange={e => setListingId(e.target.value)}
                    placeholder="Paste listing UUID if you have a matching item listed"
                    className="w-full px-4 py-2.5 border border-gray-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <button
                  type="submit"
                  disabled={submitting}
                  className="w-full py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-60 text-white rounded-xl font-semibold text-sm transition-colors flex items-center justify-center gap-2"
                >
                  {submitting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
                  {submitting ? 'Sending…' : 'Send Response'}
                </button>
              </form>
            )}
          </div>
        )}

        {!isAuthenticated && request.status === 'open' && (
          <div className="bg-blue-50 border border-blue-200 rounded-2xl p-5 mb-6 text-center">
            <p className="text-sm text-blue-700 mb-3">Sign in to respond as a seller</p>
            <Link
              href={`/login?redirect=/requests/${id}`}
              className="inline-block px-5 py-2 bg-blue-600 text-white rounded-xl text-sm font-semibold hover:bg-blue-700 transition-colors"
            >
              Sign In
            </Link>
          </div>
        )}

        {/* Responses list */}
        <div className="space-y-3">
          <h2 className="font-semibold text-gray-900">
            Responses ({responses.length})
          </h2>

          {responses.length === 0 ? (
            <div className="bg-white rounded-2xl border border-gray-200 p-8 text-center text-gray-400">
              <MessageSquare className="w-10 h-10 mx-auto mb-3 text-gray-200" />
              <p className="text-sm">No responses yet — be the first seller to reply!</p>
            </div>
          ) : (
            responses.map(r => (
              <div key={r.id} className="bg-white rounded-2xl border border-gray-200 p-5">
                <div className="flex items-center justify-between mb-2">
                  <span className="font-semibold text-sm text-gray-900">{r.seller_name}</span>
                  <span className="text-xs text-gray-400">{formatDate(r.created_at)}</span>
                </div>
                <p className="text-sm text-gray-700">{r.message}</p>
                {r.listing_id && (
                  <Link
                    href={`/listings/${r.listing_id}`}
                    className="inline-flex items-center gap-1.5 mt-3 text-xs text-blue-600 hover:underline font-medium"
                  >
                    <Tag className="w-3 h-3" />
                    View matching listing →
                  </Link>
                )}
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
