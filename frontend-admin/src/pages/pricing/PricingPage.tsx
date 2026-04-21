import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { ReadOnlyNotice } from "@/components/authz/ReadOnlyNotice"
import { api } from "@/api/client"
import { Pencil, Loader2, CheckCircle, XCircle, Star, Zap, Crown, Building2 } from "lucide-react"
import { useAuthStore } from "@/store/auth"
import { PERMISSIONS, hasPermission } from "@/lib/permissions"

interface Plan {
  id: string
  name: string
  display_name: string
  price_monthly: number
  currency: string
  stripe_price_id?: string
  listing_limit: number
  features: string[]
  is_active: boolean
  sort_order: number
}

const PLAN_ICONS: Record<string, React.ElementType> = {
  free: Star, basic: Zap, pro: Crown, enterprise: Building2,
}

export function PricingPage() {
  const qc = useQueryClient()
  const [editId, setEditId] = useState<string | null>(null)
  const [form, setForm] = useState<Partial<Plan>>({})
  const [featuresText, setFeaturesText] = useState("")
  const [error, setError] = useState("")
  const role = useAuthStore((state) => state.user?.role)
  const canManagePlans = hasPermission(role, PERMISSIONS.PLANS_MANAGE)

  const { data: plans = [], isLoading } = useQuery<Plan[]>({
    queryKey: ["admin-plans"],
    queryFn: () => api.get("/api/v1/admin/plans").then((r: { data: Plan[] | { data?: Plan[] } }) =>
      (r.data as { data?: Plan[] }).data ?? (r.data as Plan[]) ?? []
    ),
  })

  const save = useMutation({
    mutationFn: (data: Partial<Plan>) => api.put(`/api/v1/admin/plans/${editId}`, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin-plans"] })
      setEditId(null)
      setForm({})
      setFeaturesText("")
      setError("")
    },
    onError: (err: unknown) => {
      const e = err as { response?: { data?: { error?: string } } }
      setError(e?.response?.data?.error ?? "Save failed")
    },
  })

  const handleEdit = (plan: Plan) => {
    setForm({ ...plan })
    setFeaturesText(plan.features.join("\n"))
    setEditId(plan.id)
    setError("")
  }

  const handleSave = () => {
    if (!canManagePlans) return
    const features = featuresText.split("\n").map(s => s.trim()).filter(Boolean)
    save.mutate({ ...form, features })
  }

  if (isLoading) {
    return (
      <div className="flex justify-center py-16">
        <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Price Plans</h1>
        <p className="text-gray-500 text-sm mt-0.5">Edit subscription tiers, limits, and features</p>
        {!canManagePlans && <ReadOnlyNotice className="mt-2" />}
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-xl text-sm text-red-700">{error}</div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {plans.map(plan => {
          const Icon = PLAN_ICONS[plan.name] ?? Star
          const isEditing = editId === plan.id

          return (
            <Card key={plan.id} className={`p-5 ${isEditing ? "border-blue-400 ring-1 ring-blue-400" : ""}`}>
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-2">
                  <Icon className="w-5 h-5 text-blue-600" />
                  <span className="font-bold text-gray-900 capitalize">{plan.display_name}</span>
                  {!plan.is_active && (
                    <span className="text-xs bg-red-100 text-red-600 px-1.5 py-0.5 rounded-full">inactive</span>
                  )}
                </div>
                {!isEditing && canManagePlans && (
                  <Button variant="ghost" size="sm" className="h-8 w-8 p-0" onClick={() => handleEdit(plan)}>
                    <Pencil className="w-4 h-4" />
                  </Button>
                )}
              </div>

              {isEditing ? (
                <div className="space-y-3">
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <p className="text-xs font-medium text-gray-600 mb-1">Display Name</p>
                      <Input value={form.display_name ?? ""} onChange={e => setForm(f => ({ ...f, display_name: e.target.value }))} className="h-8 text-sm" />
                    </div>
                    <div>
                      <p className="text-xs font-medium text-gray-600 mb-1">Price/Month ({form.currency ?? "AED"})</p>
                      <Input type="number" value={form.price_monthly ?? 0} onChange={e => setForm(f => ({ ...f, price_monthly: parseFloat(e.target.value) || 0 }))} className="h-8 text-sm" />
                    </div>
                    <div>
                      <p className="text-xs font-medium text-gray-600 mb-1">Listing Limit (0 = unlimited)</p>
                      <Input type="number" value={form.listing_limit ?? 0} onChange={e => setForm(f => ({ ...f, listing_limit: parseInt(e.target.value) || 0 }))} className="h-8 text-sm" />
                    </div>
                    <div>
                      <p className="text-xs font-medium text-gray-600 mb-1">Stripe Price ID</p>
                      <Input value={form.stripe_price_id ?? ""} onChange={e => setForm(f => ({ ...f, stripe_price_id: e.target.value }))} className="h-8 text-sm font-mono" placeholder="price_..." />
                    </div>
                  </div>
                  <div>
                    <p className="text-xs font-medium text-gray-600 mb-1">Features (one per line)</p>
                    <textarea
                      value={featuresText}
                      onChange={e => setFeaturesText(e.target.value)}
                      rows={4}
                      className="w-full px-3 py-2 border border-input rounded-md text-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring resize-none"
                    />
                  </div>
                  <div className="flex items-center gap-2">
                    <label className="flex items-center gap-1.5 text-xs cursor-pointer">
                      <input type="checkbox" checked={form.is_active ?? true} onChange={e => setForm(f => ({ ...f, is_active: e.target.checked }))} />
                      Active
                    </label>
                  </div>
                  <div className="flex gap-2">
                    <Button size="sm" onClick={handleSave} disabled={save.isPending} className="flex items-center gap-1.5">
                      {save.isPending ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <CheckCircle className="w-3.5 h-3.5" />}
                      Save
                    </Button>
                    <Button size="sm" variant="outline" onClick={() => { setEditId(null); setForm({}) }} className="flex items-center gap-1.5">
                      <XCircle className="w-3.5 h-3.5" /> Cancel
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="space-y-2">
                  <div className="flex items-end gap-1">
                    <span className="text-2xl font-extrabold text-gray-900">
                      {plan.price_monthly === 0 ? "Free" : `${plan.currency} ${plan.price_monthly.toLocaleString()}`}
                    </span>
                    {plan.price_monthly > 0 && <span className="text-xs text-gray-400 mb-1">/mo</span>}
                  </div>
                  <p className="text-xs text-gray-500">
                    {plan.listing_limit === 0 ? "Unlimited listings" : `${plan.listing_limit} active listings`}
                  </p>
                  {plan.stripe_price_id && (
                    <p className="text-xs text-gray-400 font-mono truncate">{plan.stripe_price_id}</p>
                  )}
                  <ul className="mt-2 space-y-1">
                    {plan.features.map((f, i) => (
                      <li key={i} className="text-xs text-gray-600 flex items-center gap-1.5">
                        <CheckCircle className="w-3 h-3 text-green-500 flex-shrink-0" />{f}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </Card>
          )
        })}
      </div>
    </div>
  )
}
