import { useSettings, useSaveSettings } from "@/hooks/use-settings";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Switch } from "@/components/ui/switch";
import { Save, Mail, Settings2, ShieldCheck } from "lucide-react";
import { useToast } from "@/hooks/use-toast";

export default function SettingsPage() {
  const { data: settings, isLoading } = useSettings();
  const save = useSaveSettings();
  const { toast } = useToast();

  if (isLoading) return <div>Loading settings...</div>;

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    save.mutate(settings, {
      onSuccess: () => toast({ title: "Settings saved successfully" })
    });
  };

  return (
    <div className="space-y-6 max-w-4xl">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold font-display tracking-tight text-foreground">Site Settings</h1>
        <Button onClick={handleSave} disabled={save.isPending} className="shadow-sm">
          <Save className="w-4 h-4 mr-2" /> {save.isPending ? "Saving..." : "Save Changes"}
        </Button>
      </div>

      <Tabs defaultValue="general" className="w-full">
        <TabsList className="bg-muted/50 border border-border/50 p-1 rounded-xl h-auto mb-6">
          <TabsTrigger value="general" className="rounded-lg py-2.5 px-6 data-[state=active]:bg-background data-[state=active]:shadow-sm"><Settings2 className="w-4 h-4 mr-2"/> General</TabsTrigger>
          <TabsTrigger value="listings" className="rounded-lg py-2.5 px-6 data-[state=active]:bg-background data-[state=active]:shadow-sm"><ShieldCheck className="w-4 h-4 mr-2"/> Listing Rules</TabsTrigger>
          <TabsTrigger value="emails" className="rounded-lg py-2.5 px-6 data-[state=active]:bg-background data-[state=active]:shadow-sm"><Mail className="w-4 h-4 mr-2"/> Email Templates</TabsTrigger>
        </TabsList>

        <TabsContent value="general" className="animate-in fade-in slide-in-from-bottom-2">
          <Card className="p-6 border-none shadow-sm space-y-6">
            <div className="grid grid-cols-2 gap-6">
              <div className="space-y-2">
                <Label>Site Name</Label>
                <Input defaultValue={settings?.general?.site_name} className="bg-muted/30" />
              </div>
              <div className="space-y-2">
                <Label>Contact Email</Label>
                <Input defaultValue={settings?.general?.contact_email} type="email" className="bg-muted/30" />
              </div>
              <div className="space-y-2">
                <Label>Default Currency</Label>
                <Input defaultValue={settings?.general?.currency} className="bg-muted/30" />
              </div>
              <div className="space-y-2">
                <Label>Max Price Limit</Label>
                <Input defaultValue={settings?.general?.max_price} type="number" className="bg-muted/30" />
              </div>
            </div>

            <div className="flex items-center justify-between p-4 bg-muted/20 rounded-xl border border-border/50">
              <div>
                <p className="font-semibold text-foreground">Maintenance Mode</p>
                <p className="text-sm text-muted-foreground">Disable public access to the platform</p>
              </div>
              <Switch defaultChecked={settings?.general?.maintenance_mode} />
            </div>
          </Card>
        </TabsContent>

        <TabsContent value="listings" className="animate-in fade-in slide-in-from-bottom-2">
          <Card className="p-6 border-none shadow-sm space-y-6">
            <div className="flex items-center justify-between p-4 bg-primary/5 rounded-xl border border-primary/20">
              <div>
                <p className="font-semibold text-primary">Require Manual Approval</p>
                <p className="text-sm text-primary/70">All new listings go to the pending queue first</p>
              </div>
              <Switch defaultChecked={settings?.listing_rules?.require_approval} className="data-[state=checked]:bg-primary" />
            </div>

            <div className="grid grid-cols-2 gap-6">
              <div className="space-y-2">
                <Label>Max Images Per Listing</Label>
                <Input defaultValue={settings?.listing_rules?.max_images} type="number" className="bg-muted/30" />
              </div>
              <div className="space-y-2">
                <Label>Default Duration (Days)</Label>
                <Input defaultValue={settings?.listing_rules?.duration_days} type="number" className="bg-muted/30" />
              </div>
            </div>
          </Card>
        </TabsContent>

        <TabsContent value="emails" className="animate-in fade-in slide-in-from-bottom-2">
          <div className="grid gap-4">
            {['Welcome Email', 'Verification Link', 'Listing Approved', 'Bid Won'].map((tpl) => (
              <Card key={tpl} className="p-5 border-none shadow-sm flex items-center justify-between group hover:shadow-md transition-shadow">
                <div className="flex items-center gap-4">
                  <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center">
                    <Mail className="w-5 h-5 text-muted-foreground" />
                  </div>
                  <div>
                    <p className="font-semibold text-foreground">{tpl}</p>
                    <p className="text-sm text-muted-foreground font-mono mt-0.5">Subject: {{
                      'Welcome Email': 'Welcome to GeoCore!',
                      'Verification Link': 'Verify your email address',
                      'Listing Approved': 'Your listing is now live!',
                      'Bid Won': 'Congratulations! You won the auction.'
                    }[tpl]}</p>
                  </div>
                </div>
                <Button variant="outline">Edit Template</Button>
              </Card>
            ))}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
