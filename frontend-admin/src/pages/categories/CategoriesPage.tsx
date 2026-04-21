import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { ReadOnlyNotice } from "@/components/authz/ReadOnlyNotice"
import { api } from "@/api/client"
import { Plus, Pencil, Trash2, ChevronRight, Loader2, CheckCircle, XCircle } from "lucide-react"
import { useAuthStore } from "@/store/auth"
import { PERMISSIONS, hasPermission } from "@/lib/permissions"

interface Category {
  id: string
  parent_id?: string
  name_en: string
  name_ar?: string
  slug: string
  icon?: string
  sort_order: number
  is_active: boolean
  children?: Category[]
}

const emptyForm = (): Omit<Category, "id" | "children"> => ({
  parent_id: undefined,
  name_en: "",
  name_ar: "",
  slug: "",
  icon: "",
  sort_order: 0,
  is_active: true,
})

export function CategoriesPage() {
  const qc = useQueryClient()
  const [form, setForm] = useState(emptyForm())
  const [editId, setEditId] = useState<string | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [error, setError] = useState("")
  const role = useAuthStore((state) => state.user?.role)
  const canManageCatalog = hasPermission(role, PERMISSIONS.CATALOG_MANAGE)

  const { data: categories = [], isLoading } = useQuery<Category[]>({
    queryKey: ["admin-categories"],
    queryFn: () => api.get("/api/v1/admin/categories").then((r: { data: { data?: Category[]; } | Category[] }) => (r.data as { data?: Category[] }).data ?? (r.data as Category[]) ?? []),
  })

  const upsert = useMutation({
    mutationFn: (data: typeof form) =>
      editId
        ? api.put(`/api/v1/admin/categories/${editId}`, data)
        : api.post("/api/v1/admin/categories", data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin-categories"] })
      setForm(emptyForm())
      setEditId(null)
      setShowForm(false)
      setError("")
    },
    onError: (err: unknown) => {
      const e = err as { response?: { data?: { error?: string } } }
      setError(e?.response?.data?.error ?? "Save failed")
    },
  })

  const remove = useMutation({
    mutationFn: (id: string) => api.delete(`/api/v1/admin/categories/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin-categories"] }),
  })

  const handleEdit = (cat: Category) => {
    setForm({ parent_id: cat.parent_id, name_en: cat.name_en, name_ar: cat.name_ar ?? "", slug: cat.slug, icon: cat.icon ?? "", sort_order: cat.sort_order, is_active: cat.is_active })
    setEditId(cat.id)
    setShowForm(true)
  }

  const autoSlug = (name: string) => name.toLowerCase().replace(/\s+/g, "-").replace(/[^a-z0-9-]/g, "")

  const renderTree = (cats: Category[], depth = 0) =>
    cats.map(cat => (
      <div key={cat.id}>
        <div className={`flex items-center justify-between py-2.5 px-3 rounded-lg hover:bg-gray-50 ${depth > 0 ? "ml-6 border-l border-gray-200 pl-4" : ""}`}>
          <div className="flex items-center gap-2 min-w-0">
            {depth > 0 && <ChevronRight className="w-3 h-3 text-gray-400 flex-shrink-0" />}
            <span className="text-sm font-medium text-gray-900 truncate">{cat.name_en}</span>
            <span className="text-xs text-gray-400 font-mono">{cat.slug}</span>
            {!cat.is_active && <span className="text-xs bg-red-100 text-red-600 px-1.5 py-0.5 rounded-full">inactive</span>}
          </div>
          {canManageCatalog && (
            <div className="flex gap-1 flex-shrink-0">
              <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={() => handleEdit(cat)}>
                <Pencil className="w-3.5 h-3.5" />
              </Button>
              <Button variant="ghost" size="sm" className="h-7 w-7 p-0 text-red-500 hover:text-red-700" onClick={() => { if (confirm(`Delete "${cat.name_en}"?`)) remove.mutate(cat.id) }}>
                <Trash2 className="w-3.5 h-3.5" />
              </Button>
            </div>
          )}
        </div>
        {cat.children && cat.children.length > 0 && renderTree(cat.children, depth + 1)}
      </div>
    ))

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Categories</h1>
          <p className="text-gray-500 text-sm mt-0.5">Manage listing categories and subcategories</p>
          {!canManageCatalog && <ReadOnlyNotice className="mt-2" />}
        </div>
        {canManageCatalog && (
          <Button onClick={() => { setForm(emptyForm()); setEditId(null); setShowForm(true) }} className="flex items-center gap-2">
            <Plus className="w-4 h-4" /> New Category
          </Button>
        )}
      </div>

      {/* Form */}
      {canManageCatalog && showForm && (
        <Card className="p-5 border-blue-200 bg-blue-50/30">
          <h2 className="font-semibold text-gray-800 mb-4">{editId ? "Edit Category" : "New Category"}</h2>
          {error && <p className="text-sm text-red-600 mb-3 bg-red-50 border border-red-200 rounded p-2">{error}</p>}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-xs font-medium text-gray-700 mb-1">Name (EN) *</p>
              <Input value={form.name_en} onChange={e => setForm(f => ({ ...f, name_en: e.target.value, slug: f.slug || autoSlug(e.target.value) }))} placeholder="Electronics" className="mt-1" />
            </div>
            <div>
              <p className="text-xs font-medium text-gray-700 mb-1">Name (AR)</p>
              <Input value={form.name_ar} onChange={e => setForm(f => ({ ...f, name_ar: e.target.value }))} placeholder="إلكترونيات" className="mt-1" />
            </div>
            <div>
              <p className="text-xs font-medium text-gray-700 mb-1">Slug *</p>
              <Input value={form.slug} onChange={e => setForm(f => ({ ...f, slug: e.target.value }))} placeholder="electronics" className="mt-1 font-mono text-sm" />
            </div>
            <div>
              <p className="text-xs font-medium text-gray-700 mb-1">Icon</p>
              <Input value={form.icon} onChange={e => setForm(f => ({ ...f, icon: e.target.value }))} placeholder="💻" className="mt-1" />
            </div>
            <div>
              <p className="text-xs font-medium text-gray-700 mb-1">Sort Order</p>
              <Input type="number" value={form.sort_order} onChange={e => setForm(f => ({ ...f, sort_order: parseInt(e.target.value) || 0 }))} className="mt-1" />
            </div>
            <div>
              <p className="text-xs font-medium text-gray-700 mb-1">Parent Category</p>
              <select
                value={form.parent_id ?? ""}
                onChange={e => setForm(f => ({ ...f, parent_id: e.target.value || undefined }))}
                className="mt-1 w-full h-9 rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              >
                <option value="">— Top level —</option>
                {categories.map(c => <option key={c.id} value={c.id}>{c.name_en}</option>)}
              </select>
            </div>
          </div>
          <div className="flex items-center gap-2 mt-4">
            <label className="flex items-center gap-2 text-sm cursor-pointer">
              <input type="checkbox" checked={form.is_active} onChange={e => setForm(f => ({ ...f, is_active: e.target.checked }))} />
              Active
            </label>
          </div>
          <div className="flex gap-2 mt-4">
            <Button onClick={() => upsert.mutate(form)} disabled={upsert.isPending || !form.name_en || !form.slug} className="flex items-center gap-2">
              {upsert.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <CheckCircle className="w-4 h-4" />}
              {editId ? "Save Changes" : "Create"}
            </Button>
            <Button variant="outline" onClick={() => { setShowForm(false); setEditId(null); setForm(emptyForm()) }}>
              <XCircle className="w-4 h-4 mr-1" /> Cancel
            </Button>
          </div>
        </Card>
      )}

      {/* Category tree */}
      <Card className="p-4">
        {isLoading ? (
          <div className="flex justify-center py-8"><Loader2 className="w-6 h-6 animate-spin text-blue-500" /></div>
        ) : categories.length === 0 ? (
          <p className="text-center text-gray-400 py-8 text-sm">No categories yet. Create your first one above.</p>
        ) : (
          <div className="space-y-1">{renderTree(categories)}</div>
        )}
      </Card>
    </div>
  )
}
