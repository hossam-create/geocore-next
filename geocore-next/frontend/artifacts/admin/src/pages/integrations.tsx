import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import { useToast } from "@/hooks/use-toast";
import {
  CreditCard, Mail, Bell, HardDrive, Globe,
  MessageSquare, Database, ChevronDown, ChevronUp,
  CheckCircle2, XCircle, Save, Eye, EyeOff, Info
} from "lucide-react";

// ─── Types ────────────────────────────────────────────────────────────────────
interface IntegrationStatus {
  key: string;
  configured: boolean;
  masked?: string;
  source: "env" | "db" | "unset";
  updated_at?: string;
}

// ─── Sections definition ──────────────────────────────────────────────────────
const SECTIONS = [
  {
    id: "stripe",
    label: "Stripe",
    icon: CreditCard,
    color: "#635BFF",
    description: "Credit card payments, subscriptions, and payouts",
    docsUrl: "https://dashboard.stripe.com/apikeys",
    fields: [
      { key: "STRIPE_SECRET_KEY", label: "Secret Key", hint: "sk_live_… or sk_test_…", sensitive: true },
      { key: "STRIPE_PUBLISHABLE_KEY", label: "Publishable Key", hint: "pk_live_… or pk_test_…", sensitive: false },
      { key: "STRIPE_WEBHOOK_SECRET", label: "Webhook Secret", hint: "whsec_…", sensitive: true },
    ],
  },
  {
    id: "paypal",
    label: "PayPal",
    icon: CreditCard,
    color: "#003087",
    description: "PayPal checkout, subscriptions, and refunds",
    docsUrl: "https://developer.paypal.com/dashboard/applications",
    fields: [
      { key: "PAYPAL_CLIENT_ID", label: "Client ID", hint: "App Client ID", sensitive: false },
      { key: "PAYPAL_CLIENT_SECRET", label: "Client Secret", hint: "App Client Secret", sensitive: true },
      { key: "PAYPAL_MODE", label: "Mode", hint: "sandbox or live", sensitive: false },
    ],
  },
  {
    id: "resend",
    label: "Resend (Email)",
    icon: Mail,
    color: "#000000",
    description: "Transactional emails: welcome, auction won, outbid, escrow, etc.",
    docsUrl: "https://resend.com/api-keys",
    fields: [
      { key: "RESEND_API_KEY", label: "API Key", hint: "re_…", sensitive: true },
      { key: "RESEND_FROM_EMAIL", label: "From Email", hint: "noreply@yourdomain.com", sensitive: false },
      { key: "RESEND_FROM_NAME", label: "From Name", hint: "GeoCore Marketplace", sensitive: false },
    ],
  },
  {
    id: "firebase",
    label: "Firebase (Push Notifications)",
    icon: Bell,
    color: "#FFCA28",
    description: "Mobile push notifications for bids, messages, and alerts",
    docsUrl: "https://console.firebase.google.com/project/_/settings/serviceaccounts/adminsdk",
    fields: [
      {
        key: "FIREBASE_SERVICE_ACCOUNT_JSON",
        label: "Service Account JSON",
        hint: "Paste the full JSON from Firebase Console → Project Settings → Service Accounts",
        sensitive: true,
        multiline: true,
      },
    ],
  },
  {
    id: "r2",
    label: "Cloudflare R2 (Image Storage)",
    icon: HardDrive,
    color: "#F38020",
    description: "User-uploaded listing images, store banners, and avatars",
    docsUrl: "https://dash.cloudflare.com/?to=/:account/r2",
    fields: [
      { key: "R2_ACCOUNT_ID", label: "Account ID", hint: "From Cloudflare dashboard", sensitive: false },
      { key: "R2_ACCESS_KEY_ID", label: "Access Key ID", hint: "R2 API token access key", sensitive: false },
      { key: "R2_SECRET_ACCESS_KEY", label: "Secret Access Key", hint: "R2 API token secret", sensitive: true },
      { key: "R2_BUCKET_NAME", label: "Bucket Name", hint: "e.g. geocore-images", sensitive: false },
      { key: "R2_PUBLIC_URL", label: "Public URL", hint: "https://pub-xxx.r2.dev or custom domain", sensitive: false },
    ],
  },
  {
    id: "google",
    label: "Google OAuth",
    icon: Globe,
    color: "#4285F4",
    description: "Sign in with Google for web and mobile",
    docsUrl: "https://console.cloud.google.com/apis/credentials",
    fields: [
      { key: "GOOGLE_CLIENT_ID", label: "Client ID", hint: "xxxx.apps.googleusercontent.com", sensitive: false },
      { key: "GOOGLE_CLIENT_SECRET", label: "Client Secret", hint: "GOCSPX-…", sensitive: true },
      { key: "GA_MEASUREMENT_ID", label: "Google Analytics ID", hint: "G-XXXXXXXXXX", sensitive: false },
    ],
  },
  {
    id: "apple",
    label: "Apple Sign In",
    icon: Globe,
    color: "#000000",
    description: "Sign in with Apple (required for iOS App Store)",
    docsUrl: "https://developer.apple.com/account/resources/authkeys/list",
    fields: [
      { key: "APPLE_TEAM_ID", label: "Team ID", hint: "10-character Team ID", sensitive: false },
      { key: "APPLE_KEY_ID", label: "Key ID", hint: "10-character Key ID", sensitive: false },
      { key: "APPLE_PRIVATE_KEY", label: "Private Key (.p8)", hint: "Paste contents of AuthKey_XXXXXXXX.p8", sensitive: true, multiline: true },
    ],
  },
  {
    id: "twilio",
    label: "Twilio (SMS)",
    icon: MessageSquare,
    color: "#F22F46",
    description: "SMS alerts and OTP verification",
    docsUrl: "https://console.twilio.com/",
    fields: [
      { key: "TWILIO_ACCOUNT_SID", label: "Account SID", hint: "ACxxxxxxxxxx", sensitive: false },
      { key: "TWILIO_AUTH_TOKEN", label: "Auth Token", hint: "From Twilio Console", sensitive: true },
      { key: "TWILIO_FROM_NUMBER", label: "From Number", hint: "+1234567890", sensitive: false },
    ],
  },
  {
    id: "whatsapp",
    label: "WhatsApp Business",
    icon: MessageSquare,
    color: "#25D366",
    description: "WhatsApp notifications for orders, bids, and alerts",
    docsUrl: "https://developers.facebook.com/apps/",
    fields: [
      { key: "WHATSAPP_API_KEY", label: "API Key / Access Token", hint: "Meta business API token", sensitive: true },
      { key: "WHATSAPP_PHONE_NUMBER_ID", label: "Phone Number ID", hint: "From Meta for Developers console", sensitive: false },
    ],
  },
  {
    id: "redis",
    label: "Redis (Production Cache)",
    icon: Database,
    color: "#DC382D",
    description: "Managed Redis for rate limiting, sessions, and real-time features",
    docsUrl: "https://redis.io/cloud/",
    fields: [
      { key: "REDIS_URL", label: "Redis URL", hint: "redis://:password@host:6379 or rediss://…", sensitive: true },
    ],
  },
];

// ─── Hooks ────────────────────────────────────────────────────────────────────
function useIntegrations() {
  return useQuery<IntegrationStatus[]>({
    queryKey: ["admin_integrations"],
    queryFn: async () => {
      const res = await api.get("/admin/integrations");
      return res.data.data ?? [];
    },
    staleTime: 30_000,
  });
}

function useSaveIntegrations() {
  const qc = useQueryClient();
  const { toast } = useToast();
  return useMutation({
    mutationFn: (body: Record<string, string>) => api.post("/admin/integrations", body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin_integrations"] });
      toast({ title: "Integration keys saved successfully" });
    },
    onError: (err: any) =>
      toast({
        title: "Failed to save",
        description: err?.response?.data?.error ?? "Please try again.",
        variant: "destructive",
      }),
  });
}

// ─── Field component ──────────────────────────────────────────────────────────
function IntegrationField({
  field,
  status,
  value,
  onChange,
}: {
  field: { key: string; label: string; hint: string; sensitive: boolean; multiline?: boolean };
  status?: IntegrationStatus;
  value: string;
  onChange: (v: string) => void;
}) {
  const [show, setShow] = useState(false);
  const isConfigured = status?.configured ?? false;
  const isEnv = status?.source === "env";

  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <Label className="text-xs font-medium text-foreground">{field.label}</Label>
        <div className="flex items-center gap-1.5">
          {isConfigured ? (
            <Badge variant="secondary" className="text-[10px] py-0 h-4 bg-emerald-500/10 text-emerald-600 border-emerald-500/20">
              <CheckCircle2 className="w-2.5 h-2.5 mr-1" />
              {isEnv ? "via env" : "configured"}
            </Badge>
          ) : (
            <Badge variant="secondary" className="text-[10px] py-0 h-4 bg-orange-500/10 text-orange-600 border-orange-500/20">
              <XCircle className="w-2.5 h-2.5 mr-1" />
              not set
            </Badge>
          )}
        </div>
      </div>

      <div className="relative">
        {field.multiline ? (
          <Textarea
            placeholder={isConfigured ? (status?.masked ?? "••••••••") : field.hint}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            disabled={isEnv}
            rows={3}
            className="font-mono text-xs resize-none bg-muted/30 disabled:opacity-60"
          />
        ) : (
          <Input
            type={field.sensitive && !show ? "password" : "text"}
            placeholder={isConfigured ? (status?.masked ?? "••••••••") : field.hint}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            disabled={isEnv}
            className="pr-8 font-mono text-xs bg-muted/30 disabled:opacity-60"
          />
        )}
        {field.sensitive && !field.multiline && (
          <button
            type="button"
            onClick={() => setShow((s) => !s)}
            className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
          >
            {show ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
          </button>
        )}
      </div>

      {isEnv && (
        <p className="text-[10px] text-muted-foreground flex items-center gap-1">
          <Info className="w-3 h-3" />
          Set via environment variable — edit in server environment to change
        </p>
      )}
    </div>
  );
}

// ─── Section card ─────────────────────────────────────────────────────────────
function SectionCard({
  section,
  statuses,
  onSave,
  saving,
}: {
  section: typeof SECTIONS[0];
  statuses: Record<string, IntegrationStatus>;
  onSave: (data: Record<string, string>) => void;
  saving: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [values, setValues] = useState<Record<string, string>>({});

  const Icon = section.icon;
  const totalFields = section.fields.length;
  const configuredCount = section.fields.filter((f) => statuses[f.key]?.configured).length;
  const allConfigured = configuredCount === totalFields;
  const someConfigured = configuredCount > 0;

  const handleSave = () => {
    const body: Record<string, string> = {};
    for (const f of section.fields) {
      if (values[f.key] !== undefined && values[f.key] !== "") {
        body[f.key] = values[f.key];
      }
    }
    onSave(body);
    setValues({});
  };

  return (
    <Card className="border border-border/60 shadow-sm overflow-hidden">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="w-full flex items-center gap-3 p-4 hover:bg-muted/30 transition-colors text-left"
      >
        <div
          className="w-9 h-9 rounded-lg flex items-center justify-center flex-shrink-0"
          style={{ backgroundColor: section.color + "18" }}
        >
          <Icon className="w-4.5 h-4.5" style={{ color: section.color }} />
        </div>

        <div className="flex-1 min-w-0">
          <p className="text-sm font-semibold text-foreground">{section.label}</p>
          <p className="text-xs text-muted-foreground truncate">{section.description}</p>
        </div>

        <div className="flex items-center gap-2 flex-shrink-0">
          {allConfigured ? (
            <Badge className="text-[10px] py-0 h-5 bg-emerald-500/10 text-emerald-600 border-emerald-500/20 border">
              <CheckCircle2 className="w-2.5 h-2.5 mr-1" />
              {configuredCount}/{totalFields} ready
            </Badge>
          ) : someConfigured ? (
            <Badge className="text-[10px] py-0 h-5 bg-amber-500/10 text-amber-600 border-amber-500/20 border">
              {configuredCount}/{totalFields} set
            </Badge>
          ) : (
            <Badge className="text-[10px] py-0 h-5 bg-red-500/10 text-red-500 border-red-500/20 border">
              <XCircle className="w-2.5 h-2.5 mr-1" />
              Not configured
            </Badge>
          )}
          {open ? (
            <ChevronUp className="w-4 h-4 text-muted-foreground" />
          ) : (
            <ChevronDown className="w-4 h-4 text-muted-foreground" />
          )}
        </div>
      </button>

      {open && (
        <div className="border-t border-border/50 p-4 space-y-4 bg-muted/10">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {section.fields.map((field) => (
              <div key={field.key} className={field.multiline ? "md:col-span-2" : ""}>
                <IntegrationField
                  field={field}
                  status={statuses[field.key]}
                  value={values[field.key] ?? ""}
                  onChange={(v) => setValues((prev) => ({ ...prev, [field.key]: v }))}
                />
              </div>
            ))}
          </div>

          <div className="flex items-center justify-between pt-2 border-t border-border/40">
            <a
              href={section.docsUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-primary hover:underline flex items-center gap-1"
            >
              <Globe className="w-3 h-3" />
              Get API keys →
            </a>
            <Button
              size="sm"
              onClick={handleSave}
              disabled={saving || Object.values(values).every((v) => !v)}
              className="h-8 text-xs"
            >
              <Save className="w-3.5 h-3.5 mr-1.5" />
              {saving ? "Saving…" : "Save"}
            </Button>
          </div>
        </div>
      )}
    </Card>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────
export default function IntegrationsPage() {
  const { data: integrations = [], isLoading } = useIntegrations();
  const save = useSaveIntegrations();

  const statusMap = Object.fromEntries(integrations.map((s) => [s.key, s]));

  const totalConfigured = integrations.filter((s) => s.configured).length;
  const totalKeys = integrations.length;

  return (
    <div className="space-y-6 max-w-4xl">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-3xl font-bold font-display tracking-tight text-foreground">
            Integrations
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            Connect external services — payment gateways, email, push notifications, storage, and more.
            Keys saved here are used as fallback when environment variables are not set.
          </p>
        </div>
        <div className="flex-shrink-0 text-right">
          <div className="text-2xl font-bold text-foreground">
            {totalConfigured}
            <span className="text-muted-foreground font-normal text-base">/{totalKeys}</span>
          </div>
          <p className="text-xs text-muted-foreground">keys configured</p>
        </div>
      </div>

      {/* Progress bar */}
      <div className="h-2 bg-muted rounded-full overflow-hidden">
        <div
          className="h-full bg-gradient-to-r from-primary to-emerald-500 rounded-full transition-all duration-500"
          style={{ width: totalKeys ? `${(totalConfigured / totalKeys) * 100}%` : "0%" }}
        />
      </div>

      {/* Info banner */}
      <Card className="p-4 border-blue-500/20 bg-blue-500/5 flex gap-3">
        <Info className="w-4 h-4 text-blue-500 flex-shrink-0 mt-0.5" />
        <div className="text-sm text-blue-700 dark:text-blue-300 space-y-1">
          <p className="font-medium">How this works</p>
          <p className="text-xs opacity-80">
            Keys saved here are stored securely in the database and used at runtime.
            If an environment variable with the same name exists on the server, it takes priority over the database value.
            Keys marked <strong>"via env"</strong> are already set as server environment variables and cannot be edited here.
          </p>
        </div>
      </Card>

      {/* Sections */}
      {isLoading ? (
        <div className="text-center py-12 text-muted-foreground text-sm">
          Loading integration status…
        </div>
      ) : (
        <div className="space-y-3">
          {SECTIONS.map((section) => (
            <SectionCard
              key={section.id}
              section={section}
              statuses={statusMap}
              onSave={(body) => save.mutate(body)}
              saving={save.isPending}
            />
          ))}
        </div>
      )}
    </div>
  );
}
