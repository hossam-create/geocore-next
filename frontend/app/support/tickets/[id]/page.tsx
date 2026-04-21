'use client';
import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useRouter, useParams } from 'next/navigation';
import { useAuthStore } from '@/store/auth';
import api from '@/lib/api';
import {
  ArrowLeft, Loader2, Send, User, Headphones, Clock,
  AlertCircle
} from 'lucide-react';

interface TicketMessage {
  id: string;
  sender_id: string;
  is_admin: boolean;
  message: string;
  created_at: string;
  sender?: { id: string; name: string };
}

interface Ticket {
  id: string;
  user_id: string;
  subject: string;
  status: 'open' | 'in_progress' | 'resolved' | 'closed';
  priority: 'low' | 'normal' | 'high' | 'urgent';
  created_at: string;
  updated_at: string;
  messages: TicketMessage[];
}

const STATUS_CONFIG = {
  open: { label: 'Open', color: 'bg-blue-100 text-blue-700' },
  in_progress: { label: 'In Progress', color: 'bg-yellow-100 text-yellow-700' },
  resolved: { label: 'Resolved', color: 'bg-green-100 text-green-700' },
  closed: { label: 'Closed', color: 'bg-gray-100 text-gray-600' },
};

export default function TicketDetailPage() {
  const router = useRouter();
  const params = useParams();
  const ticketId = params?.id as string;
  const { isAuthenticated, user } = useAuthStore();
  const [mounted, setMounted] = useState(false);
  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [loading, setLoading] = useState(true);
  const [reply, setReply] = useState('');
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (mounted && !isAuthenticated) {
      router.push('/login?redirect=/support/tickets/' + ticketId);
    }
  }, [mounted, isAuthenticated, router, ticketId]);

  useEffect(() => {
    if (!isAuthenticated || !ticketId) return;

    const fetchTicket = async () => {
      setLoading(true);
      try {
        const res = await api.get(`/support/tickets/${ticketId}`);
        setTicket(res.data);
      } catch (err: any) {
        setError(err.response?.data?.error || 'Failed to load ticket');
      } finally {
        setLoading(false);
      }
    };

    fetchTicket();
  }, [isAuthenticated, ticketId]);

  if (!mounted || !isAuthenticated) return null;

  const handleReply = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!reply.trim() || reply.length < 10) return;

    setSending(true);
    try {
      const res = await api.post(`/support/tickets/${ticketId}/messages`, { message: reply });
      setTicket((t) => t ? { ...t, messages: [...t.messages, res.data] } : t);
      setReply('');
    } catch (err: any) {
      alert(err.response?.data?.error || 'Failed to send reply');
    } finally {
      setSending(false);
    }
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const statusConfig = ticket ? STATUS_CONFIG[ticket.status] : STATUS_CONFIG.open;

  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      {/* Header */}
      <div className="mb-6 flex items-center gap-3">
        <Link href="/support/tickets" className="rounded-lg p-2 hover:bg-gray-100">
          <ArrowLeft size={20} className="text-gray-600" />
        </Link>
        <div className="flex-1">
          <h1 className="text-xl font-bold text-gray-900">{ticket?.subject || 'Ticket'}</h1>
          <div className="flex items-center gap-2 mt-1">
            <span className={`inline-flex px-2 py-0.5 rounded-full text-xs font-medium ${statusConfig.color}`}>
              {statusConfig.label}
            </span>
            <span className="text-xs text-gray-400">
              Created {ticket ? formatDate(ticket.created_at) : ''}
            </span>
          </div>
        </div>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 size={32} className="animate-spin text-[#0071CE]" />
        </div>
      ) : error ? (
        <div className="text-center py-20 text-gray-500">{error}</div>
      ) : ticket ? (
        <>
          {/* Messages */}
          <div className="bg-white rounded-xl border border-gray-200 overflow-hidden mb-4">
            <div className="divide-y divide-gray-100 max-h-[60vh] overflow-y-auto">
              {ticket.messages.map((msg) => (
                <div
                  key={msg.id}
                  className={`p-4 ${msg.is_admin ? 'bg-purple-50' : 'bg-white'}`}
                >
                  <div className="flex items-start gap-3">
                    <div
                      className={`w-8 h-8 rounded-full flex items-center justify-center shrink-0 ${
                        msg.is_admin ? 'bg-purple-200 text-purple-700' : 'bg-[#0071CE] text-white'
                      }`}
                    >
                      {msg.is_admin ? <Headphones size={16} /> : <User size={16} />}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="text-sm font-semibold text-gray-900">
                          {msg.is_admin ? 'Support Team' : (msg.sender?.name || user?.name || 'You')}
                        </span>
                        <span className="text-xs text-gray-400">{formatDate(msg.created_at)}</span>
                      </div>
                      <p className="text-sm text-gray-700 whitespace-pre-wrap">{msg.message}</p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Reply Form */}
          {ticket.status !== 'closed' ? (
            <form onSubmit={handleReply} className="bg-white rounded-xl border border-gray-200 p-4">
              <label className="text-sm font-medium text-gray-700 block mb-2">Add Reply</label>
              <textarea
                value={reply}
                onChange={(e) => setReply(e.target.value)}
                rows={3}
                placeholder="Type your reply here (minimum 10 characters)..."
                className="w-full border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE] resize-none mb-3"
              />
              <div className="flex items-center justify-between">
                <span className={`text-xs ${reply.length >= 10 ? 'text-green-600' : 'text-gray-400'}`}>
                  {reply.length}/10 min
                </span>
                <button
                  type="submit"
                  disabled={sending || reply.length < 10}
                  className="flex items-center gap-2 bg-[#0071CE] text-white px-4 py-2 rounded-xl text-sm font-semibold hover:bg-[#005ba3] disabled:bg-gray-300 transition-colors"
                >
                  {sending ? (
                    <>
                      <Loader2 size={14} className="animate-spin" />
                      Sending...
                    </>
                  ) : (
                    <>
                      <Send size={14} />
                      Send Reply
                    </>
                  )}
                </button>
              </div>
            </form>
          ) : (
            <div className="bg-gray-50 rounded-xl border border-gray-200 p-4 text-center">
              <AlertCircle size={20} className="text-gray-400 mx-auto mb-2" />
              <p className="text-sm text-gray-500">This ticket is closed. You cannot add more replies.</p>
            </div>
          )}
        </>
      ) : null}
    </div>
  );
}
