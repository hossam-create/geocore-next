'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '@/lib/api'
import { PERMISSIONS, hasAnyPermission, hasPermission, isInternalRole } from '@/lib/permissions'
import { useAuthStore } from '@/store/auth'
import {
  Activity, Clock, Bell, Settings, Database, RefreshCw, Plus, Trash2,
  Edit3, CheckCircle, XCircle, AlertTriangle, Server, Cpu, Layers,
  ToggleLeft, ToggleRight, Eye, EyeOff, Save, ChevronDown, ChevronUp,
  Play, Zap, Info, Shield,
} from 'lucide-react'

// ─── Types ───────────────────────────────────────────────────────────────────

type Tab = 'status' | 'cron' | 'alerts' | 'config' | 'jobs'

interface SystemStatus {
  status: 'healthy' | 'degraded'
  db: boolean
  redis: boolean
  job_queue: Record<string, number>
  alerts_24h: number
  server_time: string
}

interface CronSchedule {
  id: string
  name: string
  description: string
  schedule: string
  action: string
  enabled: boolean
  last_run_at: string | null
  last_run_ok: boolean | null
  last_run_err: string
  next_run_at: string | null
}

interface AlertRule {
  id: string
  name: string
  metric: string
  condition: string
  threshold: number
  window: string
  enabled: boolean
  last_fired_at: string | null
}

interface AlertHistory {
  id: string
  rule_name: string
  metric: string
  value: number
  threshold: number
  fired_at: string
}

interface OpsConfig {
  id: string
  key: string
  value: string
  is_secret: boolean
  updated_at: string
  updated_by: string
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function fmtDate(d: string | null) {
  if (!d) return '—'
  return new Date(d).toLocaleString('en-GB', { dateStyle: 'short', timeStyle: 'short' })
}

function badge(ok: boolean, trueLabel = 'OK', falseLabel = 'Fail') {
  return ok
    ? <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-700"><CheckCircle className="w-3 h-3" />{trueLabel}</span>
    : <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-600"><XCircle className="w-3 h-3" />{falseLabel}</span>
}

function Card({ children, className = '' }: { children: React.ReactNode; className?: string }) {
  return <div className={`bg-white rounded-2xl border border-gray-100 shadow-sm ${className}`}>{children}</div>
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return <h3 className="text-sm font-semibold text-gray-700 mb-4">{children}</h3>
}

function useSimpleToast() {
  const [msg, setMsg] = useState<{ text: string; type: 'ok' | 'err' } | null>(null)
  useEffect(() => {
    if (!msg) return
    const t = setTimeout(() => setMsg(null), 3000)
    return () => clearTimeout(t)
  }, [msg])
  const toast = (text: string, type: 'ok' | 'err' = 'ok') => setMsg({ text, type })
  return { toast, msg }
}

// ─── Status Tab ──────────────────────────────────────────────────────────────

function StatusTab() {
  const { data, isLoading, refetch } = useQuery<SystemStatus>({
    queryKey: ['ops-status'],
    queryFn: () => api.get('/ops/status').then(r => r.data?.data ?? r.data),
    refetchInterval: 30_000,
  })

  const queueTotal = data?.job_queue
    ? Object.values(data.job_queue).reduce((s: number, v: unknown) => s + (typeof v === 'number' ? v : 0), 0)
    : 0

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className={`w-2.5 h-2.5 rounded-full ${data?.status === 'healthy' ? 'bg-green-500' : 'bg-yellow-400'} animate-pulse`} />
          <span className="text-sm font-semibold text-gray-700">
            System {isLoading ? '…' : data?.status === 'healthy' ? 'Healthy' : 'Degraded'}
          </span>
          {data?.server_time && <span className="text-xs text-gray-400">{fmtDate(data.server_time)}</span>}
        </div>
        <button onClick={() => refetch()} className="flex items-center gap-1 text-xs text-[#0071CE] hover:underline">
          <RefreshCw className="w-3.5 h-3.5" /> Refresh
        </button>
      </div>

      {/* Service health */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {[
          { label: 'Database', ok: data?.db ?? false, icon: Database, color: 'bg-blue-50 text-blue-600' },
          { label: 'Redis', ok: data?.redis ?? false, icon: Zap, color: 'bg-purple-50 text-purple-600' },
          { label: 'Job Queue', ok: queueTotal < 1000, icon: Layers, color: 'bg-indigo-50 text-indigo-600' },
          { label: 'Alerts (24h)', ok: (data?.alerts_24h ?? 0) === 0, icon: Bell, color: 'bg-orange-50 text-orange-600' },
        ].map(({ label, ok, icon: Icon, color }) => (
          <Card key={label} className="p-5">
            <div className="flex items-center justify-between mb-3">
              <span className={`p-2 rounded-xl ${color}`}><Icon className="w-4 h-4" /></span>
              {badge(ok)}
            </div>
            <p className="text-sm font-semibold text-gray-700">{label}</p>
            {label === 'Job Queue' && <p className="text-xs text-gray-400 mt-0.5">{queueTotal} pending</p>}
            {label === 'Alerts (24h)' && <p className="text-xs text-gray-400 mt-0.5">{data?.alerts_24h ?? 0} fired</p>}
          </Card>
        ))}
      </div>

      {/* Queue breakdown */}
      {data?.job_queue && (
        <Card className="p-5">
          <SectionTitle>Job Queue Breakdown</SectionTitle>
          <div className="grid grid-cols-3 md:grid-cols-6 gap-3">
            {Object.entries(data.job_queue).map(([k, v]) => (
              <div key={k} className="text-center">
                <p className="text-lg font-bold text-gray-900">{String(v)}</p>
                <p className="text-xs text-gray-400 capitalize">{k.replace(/_/g, ' ')}</p>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  )
}

// ─── Cron Tab ────────────────────────────────────────────────────────────────

function CronTab() {
  const qc = useQueryClient()
  const { toast, msg } = useSimpleToast()
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({ name: '', description: '', schedule: '', action: '', payload: '' })

  const { data: items = [] } = useQuery<CronSchedule[]>({
    queryKey: ['ops-cron'],
    queryFn: () => api.get('/ops/cron').then(r => r.data?.data ?? r.data ?? []),
  })

  const createMut = useMutation({
    mutationFn: (d: typeof form) => api.post('/ops/cron', { ...d, enabled: true }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['ops-cron'] }); setShowForm(false); setForm({ name: '', description: '', schedule: '', action: '', payload: '' }); toast('Schedule created') },
    onError: () => toast('Failed to create', 'err'),
  })

  const toggleMut = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => api.put(`/ops/cron/${id}`, { enabled }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['ops-cron'] }),
    onError: () => toast('Update failed', 'err'),
  })

  const deleteMut = useMutation({
    mutationFn: (id: string) => api.delete(`/ops/cron/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['ops-cron'] }); toast('Deleted') },
    onError: () => toast('Delete failed', 'err'),
  })

  return (
    <div className="space-y-4">
      {msg && <Toast msg={msg} />}
      <div className="flex justify-between items-center">
        <p className="text-xs text-gray-500">{items.length} schedule{items.length !== 1 ? 's' : ''}</p>
        <button onClick={() => setShowForm(!showForm)}
          className="flex items-center gap-1.5 bg-[#0071CE] text-white px-3 py-2 rounded-xl text-xs font-semibold hover:bg-[#005ba3] transition-colors">
          <Plus className="w-3.5 h-3.5" /> New Schedule
        </button>
      </div>

      {showForm && (
        <Card className="p-5">
          <SectionTitle>New Cron Schedule</SectionTitle>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <input placeholder="Name (unique)" value={form.name} onChange={e => setForm(p => ({ ...p, name: e.target.value }))} className={input} />
            <input placeholder="Schedule (e.g. */5 * * * *)" value={form.schedule} onChange={e => setForm(p => ({ ...p, schedule: e.target.value }))} className={`${input} font-mono`} />
            <input placeholder="Action (e.g. cleanup_sessions)" value={form.action} onChange={e => setForm(p => ({ ...p, action: e.target.value }))} className={`${input} font-mono`} />
            <input placeholder="Description (optional)" value={form.description} onChange={e => setForm(p => ({ ...p, description: e.target.value }))} className={input} />
            <input placeholder='Payload JSON (optional, e.g. {"key":"val"})' value={form.payload} onChange={e => setForm(p => ({ ...p, payload: e.target.value }))} className={`${input} md:col-span-2 font-mono`} />
          </div>
          <div className="flex gap-2 mt-3">
            <button onClick={() => createMut.mutate(form)} disabled={!form.name || !form.schedule || !form.action || createMut.isPending}
              className="bg-[#0071CE] text-white px-4 py-2 rounded-xl text-xs font-semibold disabled:opacity-50 hover:bg-[#005ba3] transition-colors">
              {createMut.isPending ? 'Saving…' : 'Save'}
            </button>
            <button onClick={() => setShowForm(false)} className="text-xs text-gray-400 px-3 py-2 hover:text-gray-600">Cancel</button>
          </div>
        </Card>
      )}

      <Card className="overflow-hidden">
        {items.length === 0 ? (
          <div className="py-12 text-center text-gray-400">
            <Clock className="w-8 h-8 mx-auto mb-2 text-gray-200" />
            <p className="text-sm">No cron schedules</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-50">
            {items.map(item => (
              <div key={item.id} className="flex items-start gap-4 px-5 py-4 hover:bg-gray-50 transition-colors">
                <button onClick={() => toggleMut.mutate({ id: item.id, enabled: !item.enabled })} className="mt-0.5 shrink-0">
                  {item.enabled
                    ? <ToggleRight className="w-5 h-5 text-green-500" />
                    : <ToggleLeft className="w-5 h-5 text-gray-300" />}
                </button>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-sm font-semibold text-gray-900">{item.name}</span>
                    <code className="text-xs bg-gray-100 px-2 py-0.5 rounded font-mono text-gray-600">{item.schedule}</code>
                    <code className="text-xs bg-blue-50 px-2 py-0.5 rounded font-mono text-[#0071CE]">{item.action}</code>
                  </div>
                  {item.description && <p className="text-xs text-gray-400 mt-0.5">{item.description}</p>}
                  <div className="flex items-center gap-3 mt-1 text-xs text-gray-400">
                    <span>Last: {fmtDate(item.last_run_at)}</span>
                    {item.last_run_at && item.last_run_ok !== null && badge(item.last_run_ok ?? false)}
                    {item.last_run_err && <span className="text-red-400 truncate max-w-xs">{item.last_run_err}</span>}
                    <span>Next: {fmtDate(item.next_run_at)}</span>
                  </div>
                </div>
                <button onClick={() => { if (confirm(`Delete "${item.name}"?`)) deleteMut.mutate(item.id) }}
                  className="p-1.5 rounded-lg hover:bg-red-50 text-gray-300 hover:text-red-400 transition-colors shrink-0">
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  )
}

// ─── Alerts Tab ──────────────────────────────────────────────────────────────

const METRICS = ['job_failures', 'queue_depth', 'payment_failures', 'payment_volume', 'new_users', 'active_auctions']
const CONDITIONS = ['gt', 'gte', 'lt', 'lte', 'eq']

function AlertsTab() {
  const qc = useQueryClient()
  const { toast, msg } = useSimpleToast()
  const [showForm, setShowForm] = useState(false)
  const [showHistory, setShowHistory] = useState(false)
  const [form, setForm] = useState({ name: '', metric: 'job_failures', condition: 'gt', threshold: '0', window: '1h' })

  const { data: rules = [] } = useQuery<AlertRule[]>({
    queryKey: ['ops-alerts'],
    queryFn: () => api.get('/ops/alerts').then(r => r.data?.data ?? r.data ?? []),
  })

  const { data: history = [] } = useQuery<AlertHistory[]>({
    queryKey: ['ops-alert-history'],
    queryFn: () => api.get('/ops/alerts/history').then(r => r.data?.data ?? r.data ?? []),
    enabled: showHistory,
  })

  const createMut = useMutation({
    mutationFn: (d: typeof form) => api.post('/ops/alerts', { ...d, threshold: Number(d.threshold), enabled: true }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['ops-alerts'] }); setShowForm(false); toast('Alert rule created') },
    onError: () => toast('Failed to create', 'err'),
  })

  const toggleMut = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => api.put(`/ops/alerts/${id}`, { enabled }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['ops-alerts'] }),
  })

  const deleteMut = useMutation({
    mutationFn: (id: string) => api.delete(`/ops/alerts/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['ops-alerts'] }); toast('Deleted') },
  })

  return (
    <div className="space-y-4">
      {msg && <Toast msg={msg} />}
      <div className="flex justify-between items-center">
        <div className="flex gap-2">
          <button onClick={() => setShowHistory(!showHistory)}
            className={`flex items-center gap-1 text-xs px-3 py-1.5 rounded-xl border transition-colors ${showHistory ? 'bg-gray-900 text-white border-gray-900' : 'text-gray-500 border-gray-200 hover:bg-gray-50'}`}>
            <Clock className="w-3.5 h-3.5" /> History
          </button>
        </div>
        <button onClick={() => setShowForm(!showForm)}
          className="flex items-center gap-1.5 bg-[#0071CE] text-white px-3 py-2 rounded-xl text-xs font-semibold hover:bg-[#005ba3] transition-colors">
          <Plus className="w-3.5 h-3.5" /> New Rule
        </button>
      </div>

      {showForm && (
        <Card className="p-5">
          <SectionTitle>New Alert Rule</SectionTitle>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <input placeholder="Rule name" value={form.name} onChange={e => setForm(p => ({ ...p, name: e.target.value }))} className={`${input} md:col-span-3`} />
            <select value={form.metric} onChange={e => setForm(p => ({ ...p, metric: e.target.value }))} className={select}>
              {METRICS.map(m => <option key={m} value={m}>{m.replace(/_/g, ' ')}</option>)}
            </select>
            <select value={form.condition} onChange={e => setForm(p => ({ ...p, condition: e.target.value }))} className={select}>
              {CONDITIONS.map(c => <option key={c} value={c}>{c}</option>)}
            </select>
            <input type="number" placeholder="Threshold" value={form.threshold} onChange={e => setForm(p => ({ ...p, threshold: e.target.value }))} className={input} />
            <div className="flex items-center gap-2 md:col-span-3">
              <label className="text-xs text-gray-500 w-16 shrink-0">Window</label>
              {['15m', '30m', '1h', '6h', '24h', '7d'].map(w => (
                <button key={w} onClick={() => setForm(p => ({ ...p, window: w }))}
                  className={`px-2.5 py-1 rounded-lg text-xs font-medium transition-colors ${form.window === w ? 'bg-[#0071CE] text-white' : 'bg-gray-100 text-gray-500 hover:bg-gray-200'}`}>
                  {w}
                </button>
              ))}
            </div>
          </div>
          <div className="flex gap-2 mt-3">
            <button onClick={() => createMut.mutate(form)} disabled={!form.name || createMut.isPending}
              className="bg-[#0071CE] text-white px-4 py-2 rounded-xl text-xs font-semibold disabled:opacity-50 hover:bg-[#005ba3]">
              {createMut.isPending ? 'Saving…' : 'Save'}
            </button>
            <button onClick={() => setShowForm(false)} className="text-xs text-gray-400 px-3 py-2 hover:text-gray-600">Cancel</button>
          </div>
        </Card>
      )}

      <Card className="overflow-hidden">
        {rules.length === 0 ? (
          <div className="py-12 text-center text-gray-400">
            <Bell className="w-8 h-8 mx-auto mb-2 text-gray-200" />
            <p className="text-sm">No alert rules</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-50">
            {rules.map(r => (
              <div key={r.id} className="flex items-center gap-4 px-5 py-4 hover:bg-gray-50 transition-colors">
                <button onClick={() => toggleMut.mutate({ id: r.id, enabled: !r.enabled })} className="shrink-0">
                  {r.enabled ? <ToggleRight className="w-5 h-5 text-green-500" /> : <ToggleLeft className="w-5 h-5 text-gray-300" />}
                </button>
                <div className="flex-1 min-w-0">
                  <span className="text-sm font-semibold text-gray-900">{r.name}</span>
                  <div className="flex items-center gap-2 mt-1 text-xs text-gray-400 flex-wrap">
                    <code className="bg-gray-100 px-1.5 py-0.5 rounded font-mono">{r.metric}</code>
                    <span className="font-mono text-orange-500">{r.condition} {r.threshold}</span>
                    <span>in {r.window}</span>
                    {r.last_fired_at && <span className="text-red-400 flex items-center gap-1"><AlertTriangle className="w-3 h-3" />Last fired {fmtDate(r.last_fired_at)}</span>}
                  </div>
                </div>
                <button onClick={() => { if (confirm(`Delete "${r.name}"?`)) deleteMut.mutate(r.id) }}
                  className="p-1.5 rounded-lg hover:bg-red-50 text-gray-300 hover:text-red-400 transition-colors shrink-0">
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            ))}
          </div>
        )}
      </Card>

      {showHistory && (
        <Card className="overflow-hidden">
          <div className="px-5 py-3 border-b border-gray-100">
            <h3 className="text-xs font-semibold text-gray-600">Alert History (last 100)</h3>
          </div>
          {history.length === 0 ? (
            <div className="py-8 text-center text-xs text-gray-400">No alerts fired yet</div>
          ) : (
            <div className="divide-y divide-gray-50 max-h-72 overflow-y-auto">
              {history.map(h => (
                <div key={h.id} className="flex items-center gap-4 px-5 py-3">
                  <AlertTriangle className="w-4 h-4 text-orange-400 shrink-0" />
                  <div className="flex-1 min-w-0">
                    <span className="text-xs font-medium text-gray-700">{h.rule_name}</span>
                    <span className="text-xs text-gray-400 ml-2">{h.metric}: {h.value} (threshold: {h.threshold})</span>
                  </div>
                  <span className="text-xs text-gray-400 shrink-0">{fmtDate(h.fired_at)}</span>
                </div>
              ))}
            </div>
          )}
        </Card>
      )}
    </div>
  )
}

// ─── Config Tab ──────────────────────────────────────────────────────────────

function ConfigTab() {
  const qc = useQueryClient()
  const { toast, msg } = useSimpleToast()
  const [showAdd, setShowAdd] = useState(false)
  const [revealed, setRevealed] = useState<Set<string>>(new Set())
  const [editing, setEditing] = useState<string | null>(null)
  const [editVal, setEditVal] = useState('')
  const [newForm, setNewForm] = useState({ key: '', value: '', is_secret: false })

  const { data: items = [] } = useQuery<OpsConfig[]>({
    queryKey: ['ops-config'],
    queryFn: () => api.get('/ops/config').then(r => r.data?.data ?? r.data ?? []),
  })

  const setMut = useMutation({
    mutationFn: (d: { key: string; value: string; is_secret: boolean }) => api.post('/ops/config', d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['ops-config'] }); setShowAdd(false); setNewForm({ key: '', value: '', is_secret: false }); setEditing(null); toast('Saved') },
    onError: () => toast('Save failed', 'err'),
  })

  const deleteMut = useMutation({
    mutationFn: (key: string) => api.delete(`/ops/config/${key}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['ops-config'] }); toast('Deleted') },
  })

  const PAYMENT_KEYS = ['STRIPE_SECRET_KEY', 'STRIPE_WEBHOOK_SECRET', 'PAYPAL_CLIENT_ID', 'PAYPAL_CLIENT_SECRET', 'PAYPAL_WEBHOOK_ID', 'PAYPAL_BASE_URL']

  return (
    <div className="space-y-4">
      {msg && <Toast msg={msg} />}

      {/* Quick-fill payment keys */}
      <Card className="p-5">
        <div className="flex items-center gap-2 mb-3">
          <Shield className="w-4 h-4 text-[#0071CE]" />
          <h3 className="text-sm font-semibold text-gray-700">Payment Keys</h3>
          <span className="text-xs text-gray-400">Quick-fill Stripe & PayPal credentials</span>
        </div>
        <div className="flex flex-wrap gap-2">
          {PAYMENT_KEYS.map(k => {
            const exists = items.some(i => i.key === k)
            return (
              <button key={k} onClick={() => { setShowAdd(true); setNewForm({ key: k, value: '', is_secret: true }) }}
                className={`px-2.5 py-1 rounded-lg text-xs font-mono font-medium transition-colors ${exists ? 'bg-green-50 text-green-700 border border-green-200' : 'bg-gray-100 text-gray-500 hover:bg-[#0071CE]/10 hover:text-[#0071CE]'}`}>
                {exists ? '✓ ' : '+ '}{k}
              </button>
            )
          })}
        </div>
      </Card>

      <div className="flex justify-between items-center">
        <p className="text-xs text-gray-500">{items.length} config entr{items.length !== 1 ? 'ies' : 'y'}</p>
        <button onClick={() => { setShowAdd(!showAdd); setNewForm({ key: '', value: '', is_secret: false }) }}
          className="flex items-center gap-1.5 bg-[#0071CE] text-white px-3 py-2 rounded-xl text-xs font-semibold hover:bg-[#005ba3] transition-colors">
          <Plus className="w-3.5 h-3.5" /> Add Key
        </button>
      </div>

      {showAdd && (
        <Card className="p-5">
          <SectionTitle>Set Config Key</SectionTitle>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <input placeholder="KEY_NAME" value={newForm.key} onChange={e => setNewForm(p => ({ ...p, key: e.target.value.toUpperCase() }))} className={`${input} font-mono`} />
            <input placeholder="Value" type={newForm.is_secret ? 'password' : 'text'} value={newForm.value} onChange={e => setNewForm(p => ({ ...p, value: e.target.value }))} className={input} />
          </div>
          <label className="flex items-center gap-2 mt-3 text-xs text-gray-500 cursor-pointer select-none">
            <input type="checkbox" checked={newForm.is_secret} onChange={e => setNewForm(p => ({ ...p, is_secret: e.target.checked }))} className="rounded" />
            Mark as secret (value will be masked in list)
          </label>
          <div className="flex gap-2 mt-3">
            <button onClick={() => setMut.mutate(newForm)} disabled={!newForm.key || !newForm.value || setMut.isPending}
              className="bg-[#0071CE] text-white px-4 py-2 rounded-xl text-xs font-semibold disabled:opacity-50 hover:bg-[#005ba3] flex items-center gap-1">
              <Save className="w-3.5 h-3.5" />{setMut.isPending ? 'Saving…' : 'Save'}
            </button>
            <button onClick={() => setShowAdd(false)} className="text-xs text-gray-400 px-3 py-2 hover:text-gray-600">Cancel</button>
          </div>
        </Card>
      )}

      <Card className="overflow-hidden">
        {items.length === 0 ? (
          <div className="py-12 text-center text-gray-400">
            <Settings className="w-8 h-8 mx-auto mb-2 text-gray-200" />
            <p className="text-sm">No config keys yet</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-50">
            {items.map(item => (
              <div key={item.id} className="flex items-center gap-4 px-5 py-3.5 hover:bg-gray-50 transition-colors">
                <code className="text-xs font-mono font-semibold text-gray-700 w-56 shrink-0 truncate">{item.key}</code>
                <div className="flex-1 min-w-0">
                  {editing === item.key ? (
                    <div className="flex gap-2">
                      <input type="text" value={editVal} onChange={e => setEditVal(e.target.value)}
                        className="flex-1 text-xs border border-gray-200 rounded-lg px-2 py-1 font-mono focus:outline-none focus:border-[#0071CE]" autoFocus />
                      <button onClick={() => setMut.mutate({ key: item.key, value: editVal, is_secret: item.is_secret })}
                        className="text-xs bg-green-500 text-white px-2 py-1 rounded-lg hover:bg-green-600">Save</button>
                      <button onClick={() => setEditing(null)} className="text-xs text-gray-400 px-2 py-1">✕</button>
                    </div>
                  ) : (
                    <div className="flex items-center gap-2">
                      <code className="text-xs font-mono text-gray-500 truncate max-w-xs">
                        {item.is_secret && !revealed.has(item.key) ? item.value : item.value}
                      </code>
                      {item.is_secret && (
                        <button onClick={() => setRevealed(p => { const n = new Set(p); p.has(item.key) ? n.delete(item.key) : n.add(item.key); return n })}
                          className="text-gray-300 hover:text-gray-500">
                          {revealed.has(item.key) ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                        </button>
                      )}
                      {item.is_secret && <span className="text-xs text-orange-400 font-medium">secret</span>}
                    </div>
                  )}
                </div>
                <span className="text-xs text-gray-300 shrink-0 hidden md:block">{fmtDate(item.updated_at)}</span>
                <div className="flex gap-1 shrink-0">
                  <button onClick={() => { setEditing(item.key); setEditVal(item.value) }}
                    className="p-1.5 rounded-lg hover:bg-blue-50 text-gray-300 hover:text-[#0071CE] transition-colors">
                    <Edit3 className="w-3.5 h-3.5" />
                  </button>
                  <button onClick={() => { if (confirm(`Delete key "${item.key}"?`)) deleteMut.mutate(item.key) }}
                    className="p-1.5 rounded-lg hover:bg-red-50 text-gray-300 hover:text-red-400 transition-colors">
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  )
}

// ─── Jobs Tab ─────────────────────────────────────────────────────────────────

function JobsTab() {
  const qc = useQueryClient()
  const { toast, msg } = useSimpleToast()
  const [showFailed, setShowFailed] = useState(false)

  const { data: stats, refetch } = useQuery<{ stats: Record<string, number> }>({
    queryKey: ['ops-job-stats'],
    queryFn: () => api.get('/ops/jobs/stats').then(r => r.data?.data ?? r.data),
    refetchInterval: 15_000,
  })

  const { data: failedData } = useQuery<{ failed_jobs: unknown[]; count: number }>({
    queryKey: ['ops-failed-jobs'],
    queryFn: () => api.get('/ops/jobs/failed').then(r => r.data?.data ?? r.data),
    enabled: showFailed,
  })

  const retryMut = useMutation({
    mutationFn: () => api.post('/ops/jobs/retry'),
    onSuccess: (r) => { qc.invalidateQueries({ queryKey: ['ops-job-stats'] }); qc.invalidateQueries({ queryKey: ['ops-failed-jobs'] }); toast(`Retried ${r.data?.data?.retried ?? 0} jobs`) },
    onError: () => toast('Retry failed', 'err'),
  })

  const queueStats = stats?.stats ?? {}

  return (
    <div className="space-y-4">
      {msg && <Toast msg={msg} />}
      <div className="flex justify-between items-center">
        <button onClick={() => refetch()} className="flex items-center gap-1 text-xs text-[#0071CE] hover:underline">
          <RefreshCw className="w-3.5 h-3.5" /> Refresh
        </button>
        <div className="flex gap-2">
          <button onClick={() => setShowFailed(!showFailed)}
            className={`flex items-center gap-1 text-xs px-3 py-1.5 rounded-xl border transition-colors ${showFailed ? 'bg-gray-900 text-white border-gray-900' : 'text-gray-500 border-gray-200 hover:bg-gray-50'}`}>
            <XCircle className="w-3.5 h-3.5" /> Failed Jobs
          </button>
          <button onClick={() => retryMut.mutate()} disabled={retryMut.isPending}
            className="flex items-center gap-1.5 bg-orange-500 text-white px-3 py-2 rounded-xl text-xs font-semibold hover:bg-orange-600 disabled:opacity-50 transition-colors">
            <Play className="w-3.5 h-3.5" />{retryMut.isPending ? 'Retrying…' : 'Retry All Failed'}
          </button>
        </div>
      </div>

      <div className="grid grid-cols-3 md:grid-cols-6 gap-3">
        {Object.entries(queueStats).map(([k, v]) => (
          <Card key={k} className="p-4 text-center">
            <p className={`text-2xl font-bold ${k === 'failed' && v > 0 ? 'text-red-500' : 'text-gray-900'}`}>{String(v)}</p>
            <p className="text-xs text-gray-400 capitalize mt-1">{k.replace(/_/g, ' ')}</p>
          </Card>
        ))}
      </div>

      {showFailed && (
        <Card className="overflow-hidden">
          <div className="px-5 py-3 border-b border-gray-100 flex items-center justify-between">
            <h3 className="text-xs font-semibold text-gray-600">Failed Jobs ({failedData?.count ?? 0})</h3>
          </div>
          {!failedData?.failed_jobs?.length ? (
            <div className="py-8 text-center text-xs text-gray-400 flex items-center justify-center gap-2">
              <CheckCircle className="w-4 h-4 text-green-400" /> No failed jobs
            </div>
          ) : (
            <div className="divide-y divide-gray-50 max-h-80 overflow-y-auto">
              {failedData.failed_jobs.map((job: unknown, i) => {
                const j = job as Record<string, unknown>
                return (
                  <div key={i} className="px-5 py-3 font-mono text-xs text-gray-500 hover:bg-gray-50">
                    <span className="text-[#0071CE] font-semibold">{String(j.type)}</span>
                    <span className="mx-2 text-gray-300">|</span>
                    <span className="text-red-400">{String(j.error ?? '')}</span>
                    <span className="ml-2 text-gray-300">{fmtDate(String(j.created_at ?? ''))}</span>
                  </div>
                )
              })}
            </div>
          )}
        </Card>
      )}
    </div>
  )
}

// ─── Toast ────────────────────────────────────────────────────────────────────

function Toast({ msg }: { msg: { text: string; type: 'ok' | 'err' } }) {
  return (
    <div className={`fixed bottom-6 left-1/2 -translate-x-1/2 z-50 px-4 py-2.5 rounded-xl text-sm font-medium shadow-lg flex items-center gap-2 ${msg.type === 'ok' ? 'bg-gray-900 text-white' : 'bg-red-500 text-white'}`}>
      {msg.type === 'ok' ? <CheckCircle className="w-4 h-4" /> : <XCircle className="w-4 h-4" />}
      {msg.text}
    </div>
  )
}

// ─── Shared styles ────────────────────────────────────────────────────────────

const input = 'w-full text-sm border border-gray-200 rounded-xl px-3 py-2 focus:outline-none focus:border-[#0071CE] transition-colors placeholder:text-gray-300'
const select = 'w-full text-sm border border-gray-200 rounded-xl px-3 py-2 focus:outline-none focus:border-[#0071CE] transition-colors bg-white'

// ─── Main Page ────────────────────────────────────────────────────────────────

const TABS: { key: Tab; label: string; icon: React.ElementType }[] = [
  { key: 'status',  label: 'System Status', icon: Activity },
  { key: 'cron',    label: 'Cron',          icon: Clock },
  { key: 'alerts',  label: 'Alerts',        icon: Bell },
  { key: 'config',  label: 'Config',        icon: Settings },
  { key: 'jobs',    label: 'Job Queue',     icon: Layers },
]

export default function OpsPage() {
  const { user, isAuthenticated } = useAuthStore()
  const router = useRouter()
  const [tab, setTab] = useState<Tab>('status')
  const role = user?.role
  const canAccessOps = hasAnyPermission(role, [PERMISSIONS.OPS_READ, PERMISSIONS.OPS_MANAGE])
  const canManageOps = hasPermission(role, PERMISSIONS.OPS_MANAGE)

  useEffect(() => {
    if (!isAuthenticated) { router.push('/login?next=/ops'); return }
    if (user && (!isInternalRole(role) || !canAccessOps)) {
      router.push('/')
    }
  }, [isAuthenticated, user, role, canAccessOps, router])

  if (!isAuthenticated || !user || !isInternalRole(role) || !canAccessOps) return null

  const isReadOnlyTab = tab !== 'status' && !canManageOps

  return (
    <div className="max-w-5xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="flex items-center gap-3 mb-6">
        <div className="p-2.5 bg-gray-900 rounded-xl">
          <Server className="w-5 h-5 text-white" />
        </div>
        <div>
          <h1 className="text-xl font-bold text-gray-900">Control Center</h1>
          <p className="text-xs text-gray-400">Cron · Alerts · Runtime Config · Job Queue</p>
        </div>
        <span className="ml-auto flex items-center gap-1.5 text-xs px-2.5 py-1 bg-gray-100 text-gray-500 rounded-full font-mono">
          <Shield className="w-3 h-3" /> {canManageOps ? 'ops manage' : 'ops read only'}
        </span>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 bg-gray-100 p-1 rounded-2xl mb-6 overflow-x-auto">
        {TABS.map(({ key, label, icon: Icon }) => (
          <button key={key} onClick={() => setTab(key)}
            className={`flex items-center gap-1.5 px-4 py-2 rounded-xl text-xs font-semibold transition-all whitespace-nowrap flex-1 justify-center
              ${tab === key ? 'bg-white shadow text-gray-900' : 'text-gray-500 hover:text-gray-700'}`}>
            <Icon className="w-3.5 h-3.5" />{label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      {tab === 'status' && <StatusTab />}
      {isReadOnlyTab ? (
        <Card className="p-6">
          <div className="flex items-start gap-3">
            <Shield className="w-5 h-5 text-gray-500 mt-0.5" />
            <div>
              <h3 className="text-sm font-semibold text-gray-800">Read-only access</h3>
              <p className="text-xs text-gray-500 mt-1">
                Your role can view system status only. Ops management permission is required for this section.
              </p>
            </div>
          </div>
        </Card>
      ) : (
        <>
          {tab === 'cron'   && <CronTab />}
          {tab === 'alerts' && <AlertsTab />}
          {tab === 'config' && <ConfigTab />}
          {tab === 'jobs'   && <JobsTab />}
        </>
      )}
    </div>
  )
}
