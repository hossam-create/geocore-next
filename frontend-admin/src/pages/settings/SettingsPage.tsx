import { useState } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ReadOnlyNotice } from "@/components/authz/ReadOnlyNotice"
import { Save } from "lucide-react"
import { useAuthStore } from "@/store/auth"
import { PERMISSIONS, hasPermission } from "@/lib/permissions"

export function SettingsPage() {
  const [saved, setSaved] = useState(false)
  const role = useAuthStore((state) => state.user?.role)
  const canWriteSettings = hasPermission(role, PERMISSIONS.SETTINGS_WRITE)

  const handleSave = () => {
    if (!canWriteSettings) return
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  return (
    <div className="space-y-4 max-w-4xl">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Site Settings</h1>
          <p className="text-gray-500 text-sm mt-0.5">Configure your GeoCore platform</p>
        </div>
        {canWriteSettings ? (
          <Button onClick={handleSave}>
            <Save className="w-4 h-4" />
            {saved ? "Saved!" : "Save Changes"}
          </Button>
        ) : (
          <ReadOnlyNotice />
        )}
      </div>

      <div className={canWriteSettings ? "" : "pointer-events-none opacity-80"}>
        <Tabs defaultValue="general">
          <TabsList>
            <TabsTrigger value="general">General</TabsTrigger>
            <TabsTrigger value="listings">Listings</TabsTrigger>
            <TabsTrigger value="emails">Emails</TabsTrigger>
            <TabsTrigger value="payments">Payments</TabsTrigger>
          </TabsList>

        <TabsContent value="general">
          <Card className="p-6 space-y-5 mt-4">
            <h3 className="font-semibold text-gray-900">General Settings</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Site Name</label>
                <Input defaultValue="GeoCore Marketplace" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Site URL</label>
                <Input defaultValue="https://geocore.io" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Support Email</label>
                <Input defaultValue="support@geocore.io" type="email" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Default Currency</label>
                <Input defaultValue="USD" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Default Language</label>
                <Input defaultValue="en" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Timezone</label>
                <Input defaultValue="UTC" />
              </div>
            </div>
          </Card>
        </TabsContent>

        <TabsContent value="listings">
          <Card className="p-6 space-y-5 mt-4">
            <h3 className="font-semibold text-gray-900">Listing Settings</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Max Images per Listing</label>
                <Input defaultValue="10" type="number" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Max Title Length</label>
                <Input defaultValue="150" type="number" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Listing Duration (days)</label>
                <Input defaultValue="60" type="number" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Max Listings per User</label>
                <Input defaultValue="50" type="number" />
              </div>
            </div>
            <div className="flex items-center gap-6">
              {[
                { label: "Require Moderation", desc: "All listings require admin approval" },
                { label: "Allow Auctions", desc: "Users can create auction listings" },
                { label: "Allow Storefronts", desc: "Users can create storefront pages" },
              ].map((item) => (
                <label key={item.label} className="flex items-start gap-3 cursor-pointer">
                  <input type="checkbox" defaultChecked className="mt-0.5 rounded" />
                  <div>
                    <p className="text-sm font-medium text-gray-700">{item.label}</p>
                    <p className="text-xs text-gray-400">{item.desc}</p>
                  </div>
                </label>
              ))}
            </div>
          </Card>
        </TabsContent>

        <TabsContent value="emails">
          <Card className="p-6 space-y-5 mt-4">
            <h3 className="font-semibold text-gray-900">Email Configuration</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">SMTP Host</label>
                <Input defaultValue="smtp.sendgrid.net" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">SMTP Port</label>
                <Input defaultValue="587" type="number" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">From Name</label>
                <Input defaultValue="GeoCore" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">From Email</label>
                <Input defaultValue="noreply@geocore.io" type="email" />
              </div>
            </div>
          </Card>
        </TabsContent>

        <TabsContent value="payments">
          <Card className="p-6 space-y-5 mt-4">
            <h3 className="font-semibold text-gray-900">Payment Settings</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Final Value Fee (%)</label>
                <Input defaultValue="5" type="number" min="0" max="100" step="0.5" />
                <p className="text-xs text-gray-400 mt-1">Percentage deducted from completed auction sales</p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Listing Fee (USD)</label>
                <Input defaultValue="0" type="number" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Featured Listing Fee (USD)</label>
                <Input defaultValue="9.99" type="number" />
              </div>
            </div>
          </Card>
        </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}
