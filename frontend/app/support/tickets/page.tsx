'use client';
import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/store/auth';
import api from '@/lib/api';
import {
  Ticket, Plus, MessageSquare, Clock, ChevronRight, Loader2,
  ArrowLeft, AlertCircle
} from 'lucide-react';

interface Ticket {
  id: string;
  subject: string;
  status: 'open' | 'in_progress' | 'resolved' | 'closed';
  priority: 'low' | 'normal' | 'high' | 'urgent';
  created_at: string;
  updated_at: string;
}

const STATUS_CONFIG = {
  open: { label: 'Open', color: 'bg-blue-100 text-blue-700', dot: 'bg-blue-500' },
  in_progress: { label: 'In Progress', color: 'bg-yellow-100 text-yellow-700', dot: 'bg-yellow-500' },
  resolved: { label: 'Resolved', color: 'bg-green-100 text-green-700', dot: 'bg-green-500' },
  closed: { label: 'Closed', color: 'bg-gray-100 text-gray-600', dot: 'bg-gray-400' },
};

const PRIORITY_CONFIG = {
  low: { label: 'Low', color: 'text-gray-500' },
  normal: { label: 'Normal', color: 'text-blue-600' },
  high: { label: 'High', color: 'text-orange-600' },
  urgent: { label: 'Urgent', color: 'text-red-600' },
};

export default function SupportTicketsPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [mounted, setMounted] = useState(false);
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [newTicket, setNewTicket] = useState({ subject: '', message: '' });
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (mounted && !isAuthenticated) {
      router.push('/login?redirect=/support/tickets');
    }
  }, [mounted, isAuthenticated, router]);

  useEffect(() => {
    if (!isAuthenticated) return;

    const fetchTickets = async () => {
      setLoading(true);
      try {
        const res = await api.get('/support/tickets');
        setTickets(res.data || []);
      } catch {
        // Handle error
      } finally {
        setLoading(false);
      }
    };

    fetchTickets();
  }, [isAuthenticated]);

  if (!mounted || !isAuthenticated) return null;

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);
    try {
      const res = await api.post('/support/tickets', newTicket);
      setTickets([res.data, ...tickets]);
      setShowCreate(false);
      setNewTicket({ subject: '', message: '' });
    } catch (err: any) {
      alert(err.response?.data?.error || 'Failed to create ticket');
    } finally {
      setCreating(false);
    }
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
  };

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link href="/dashboard" className="rounded-lg p-2 hover:bg-gray-100">
            <ArrowLeft size={20} className="text-gray-600" />
          </Link>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Support Tickets</h1>
            <p className="text-sm text-gray-500">Track and manage your support requests</p>
          </div>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-2 bg-[#0071CE] text-white px-4 py-2.5 rounded-xl text-sm font-semibold hover:bg-[#005ba3] transition-colors"
        >
          <Plus size={16} />
          New Ticket
        </button>
      </div>

      {/* Create Modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl w-full max-w-lg p-6">
            <h2 className="text-lg font-bold text-gray-900 mb-4">Create New Ticket</h2>
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="text-sm font-medium text-gray-700 block mb-1">Subject</label>
                <input
                  type="text"
                  value={newTicket.subject}
                  onChange={(e) => setNewTicket({ ...newTicket, subject: e.target.value })}
                  required
                  minLength={5}
                  placeholder="Brief description of your issue"
                  className="w-full border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
                />
              </div>
              <div>
                <label className="text-sm font-medium text-gray-700 block mb-1">Message</label>
                <textarea
                  value={newTicket.message}
                  onChange={(e) => setNewTicket({ ...newTicket, message: e.target.value })}
                  required
                  minLength={20}
                  rows={5}
                  placeholder="Describe your issue in detail..."
                  className="w-full border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE] resize-none"
                />
              </div>
              <div className="flex justify-end gap-3">
                <button
                  type="button"
                  onClick={() => setShowCreate(false)}
                  className="px-4 py-2 text-sm font-medium text-gray-600 hover:bg-gray-100 rounded-xl"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={creating}
                  className="flex items-center gap-2 bg-[#0071CE] text-white px-4 py-2 rounded-xl text-sm font-semibold hover:bg-[#005ba3] disabled:bg-gray-300"
                >
                  {creating && <Loader2 size={14} className="animate-spin" />}
                  Create Ticket
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {loading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 size={32} className="animate-spin text-[#0071CE]" />
        </div>
      ) : tickets.length === 0 ? (
        <div className="text-center py-20">
          <Ticket size={48} className="text-gray-300 mx-auto mb-4" />
          <h2 className="text-xl font-semibold text-gray-700 mb-2">No Tickets Yet</h2>
          <p className="text-gray-400 text-sm mb-6">You haven't created any support tickets.</p>
          <button
            onClick={() => setShowCreate(true)}
            className="inline-flex items-center gap-2 bg-[#0071CE] text-white px-6 py-3 rounded-xl font-semibold hover:bg-[#005ba3]"
          >
            <Plus size={16} />
            Create Your First Ticket
          </button>
        </div>
      ) : (
        <div className="space-y-3">
          {tickets.map((ticket) => {
            const statusConfig = STATUS_CONFIG[ticket.status];
            const priorityConfig = PRIORITY_CONFIG[ticket.priority];

            return (
              <Link
                key={ticket.id}
                href={`/support/tickets/${ticket.id}`}
                className="block bg-white rounded-xl border border-gray-200 p-4 hover:shadow-sm transition-shadow"
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${statusConfig.color}`}>
                        <span className={`w-1.5 h-1.5 rounded-full ${statusConfig.dot}`} />
                        {statusConfig.label}
                      </span>
                      <span className={`text-xs font-medium ${priorityConfig.color}`}>
                        {priorityConfig.label} Priority
                      </span>
                    </div>
                    <h3 className="text-sm font-semibold text-gray-900 truncate">{ticket.subject}</h3>
                    <div className="flex items-center gap-3 mt-2 text-xs text-gray-400">
                      <span className="flex items-center gap-1">
                        <Clock size={12} />
                        Created {formatDate(ticket.created_at)}
                      </span>
                    </div>
                  </div>
                  <ChevronRight size={18} className="text-gray-300 shrink-0" />
                </div>
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}
