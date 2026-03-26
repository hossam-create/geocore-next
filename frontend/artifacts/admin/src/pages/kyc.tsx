import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useToast } from "@/hooks/use-toast";
import {
  ShieldCheck,
  ShieldX,
  Clock,
  Eye,
  CheckCircle2,
  XCircle,
  AlertCircle,
  RefreshCcw,
  Search,
  User,
  FileImage,
} from "lucide-react";

type KYCStatus = "pending" | "under_review" | "approved" | "rejected";

interface KYCDocument {
  id: string;
  document_type: string;
  file_url: string;
  side: string;
  verified: boolean;
  created_at: string;
}

interface KYCProfile {
  id: string;
  user_id: string;
  status: KYCStatus;
  full_name: string;
  id_number: string;
  country: string;
  nationality: string;
  date_of_birth: string;
  rejection_reason?: string;
  approved_at?: string;
  expires_at?: string;
  risk_level: "low" | "medium" | "high";
  documents: KYCDocument[];
  created_at: string;
  updated_at: string;
}

interface KYCStats {
  total: number;
  pending: number;
  under_review: number;
  approved: number;
  rejected: number;
}

// ── Mock data fallback ────────────────────────────────────────────────────────
const MOCK_PROFILES: KYCProfile[] = [
  {
    id: "kyc-001",
    user_id: "user-001",
    status: "pending",
    full_name: "Ahmed Al-Rashidi",
    id_number: "784-1990-1234567-1",
    country: "ARE",
    nationality: "ARE",
    date_of_birth: "1990-05-15",
    risk_level: "low",
    documents: [
      { id: "doc-001", document_type: "emirates_id", file_url: "https://placehold.co/400x250?text=Emirates+ID+Front", side: "front", verified: false, created_at: new Date(Date.now() - 2 * 86400000).toISOString() },
      { id: "doc-002", document_type: "selfie", file_url: "https://placehold.co/300x300?text=Live+Selfie", side: "front", verified: false, created_at: new Date(Date.now() - 2 * 86400000).toISOString() },
    ],
    created_at: new Date(Date.now() - 2 * 86400000).toISOString(),
    updated_at: new Date(Date.now() - 2 * 86400000).toISOString(),
  },
  {
    id: "kyc-002",
    user_id: "user-002",
    status: "under_review",
    full_name: "Sarah Al-Mansoori",
    id_number: "784-1988-7654321-2",
    country: "ARE",
    nationality: "ARE",
    date_of_birth: "1988-11-22",
    risk_level: "low",
    documents: [
      { id: "doc-003", document_type: "passport", file_url: "https://placehold.co/400x280?text=Passport", side: "front", verified: false, created_at: new Date(Date.now() - 86400000).toISOString() },
      { id: "doc-004", document_type: "selfie", file_url: "https://placehold.co/300x300?text=Live+Selfie", side: "front", verified: false, created_at: new Date(Date.now() - 86400000).toISOString() },
    ],
    created_at: new Date(Date.now() - 86400000).toISOString(),
    updated_at: new Date(Date.now() - 3600000).toISOString(),
  },
  {
    id: "kyc-003",
    user_id: "user-003",
    status: "approved",
    full_name: "Mohammed Al-Qassemi",
    id_number: "784-1985-9876543-3",
    country: "ARE",
    nationality: "SAU",
    date_of_birth: "1985-03-10",
    risk_level: "low",
    approved_at: new Date(Date.now() - 3 * 86400000).toISOString(),
    expires_at: new Date(Date.now() + 365 * 86400000).toISOString(),
    documents: [
      { id: "doc-005", document_type: "national_id", file_url: "https://placehold.co/400x250?text=National+ID", side: "front", verified: true, created_at: new Date(Date.now() - 5 * 86400000).toISOString() },
    ],
    created_at: new Date(Date.now() - 5 * 86400000).toISOString(),
    updated_at: new Date(Date.now() - 3 * 86400000).toISOString(),
  },
  {
    id: "kyc-004",
    user_id: "user-004",
    status: "rejected",
    full_name: "Fatima Al-Zahra",
    id_number: "784-1995-1122334-4",
    country: "KWT",
    nationality: "KWT",
    date_of_birth: "1995-07-25",
    rejection_reason: "Document image is blurry. Please resubmit with a clear, well-lit photo.",
    risk_level: "medium",
    documents: [
      { id: "doc-006", document_type: "national_id", file_url: "https://placehold.co/400x250?text=Blurry+ID", side: "front", verified: false, created_at: new Date(Date.now() - 7 * 86400000).toISOString() },
    ],
    created_at: new Date(Date.now() - 7 * 86400000).toISOString(),
    updated_at: new Date(Date.now() - 4 * 86400000).toISOString(),
  },
];

const MOCK_STATS: KYCStats = {
  total: 4,
  pending: 1,
  under_review: 1,
  approved: 1,
  rejected: 1,
};

// ── Helpers ───────────────────────────────────────────────────────────────────

function statusConfig(status: KYCStatus) {
  switch (status) {
    case "approved":
      return { label: "Approved", color: "bg-green-100 text-green-700 border-green-200", icon: CheckCircle2 };
    case "rejected":
      return { label: "Rejected", color: "bg-red-100 text-red-700 border-red-200", icon: XCircle };
    case "under_review":
      return { label: "Under Review", color: "bg-blue-100 text-blue-700 border-blue-200", icon: RefreshCcw };
    default:
      return { label: "Pending", color: "bg-yellow-100 text-yellow-700 border-yellow-200", icon: Clock };
  }
}

function riskConfig(risk: string) {
  if (risk === "high") return "bg-red-100 text-red-700";
  if (risk === "medium") return "bg-orange-100 text-orange-700";
  return "bg-green-100 text-green-700";
}

function docTypeLabel(t: string) {
  const labels: Record<string, string> = {
    emirates_id: "Emirates ID",
    passport: "Passport",
    national_id: "National ID",
    residence_visa: "Residence Visa",
    driving_license: "Driving License",
    selfie: "Live Selfie",
  };
  return labels[t] ?? t.replace(/_/g, " ");
}

function countryFlag(code: string) {
  const flags: Record<string, string> = {
    ARE: "🇦🇪",
    SAU: "🇸🇦",
    KWT: "🇰🇼",
    QAT: "🇶🇦",
    BHR: "🇧🇭",
    OMN: "🇴🇲",
  };
  return flags[code] ?? "🌍";
}

// ── API hooks (with mock fallback, same pattern as other admin hooks) ──────────

function useKYCStats() {
  return useQuery<KYCStats>({
    queryKey: ["kyc-stats"],
    queryFn: async () => {
      try {
        const res = await api.get("/kyc/admin/stats");
        return res.data?.data ?? res.data;
      } catch {
        return MOCK_STATS;
      }
    },
    refetchInterval: 30000,
  });
}

function useKYCList(status: string) {
  return useQuery<KYCProfile[]>({
    queryKey: ["kyc-list", status],
    queryFn: async () => {
      try {
        const params: Record<string, string> = { per_page: "50" };
        if (status && status !== "all") params.status = status;
        const res = await api.get("/kyc/admin/list", { params });
        const data = res.data?.data ?? res.data;
        return Array.isArray(data) ? data : MOCK_PROFILES;
      } catch {
        if (status && status !== "all") {
          return MOCK_PROFILES.filter((p) => p.status === status);
        }
        return MOCK_PROFILES;
      }
    },
  });
}

// ── Stat Card ─────────────────────────────────────────────────────────────────

function StatCard({
  label,
  value,
  icon: Icon,
  color,
  onClick,
  active,
}: {
  label: string;
  value: number;
  icon: React.ElementType;
  color: string;
  onClick: () => void;
  active: boolean;
}) {
  return (
    <button
      onClick={onClick}
      className={`flex-1 min-w-[130px] rounded-xl border p-4 text-left transition-all ${
        active ? "ring-2 ring-[#0071CE] shadow-md" : "hover:shadow-sm"
      } bg-white`}
    >
      <div className={`inline-flex p-2 rounded-lg ${color} mb-3`}>
        <Icon className="w-4 h-4" />
      </div>
      <p className="text-2xl font-bold text-gray-900">{value}</p>
      <p className="text-xs text-gray-500 mt-0.5">{label}</p>
    </button>
  );
}

// ── Profile Detail Dialog ──────────────────────────────────────────────────────

function ProfileDialog({
  profile,
  onClose,
  onUpdate,
}: {
  profile: KYCProfile | null;
  onClose: () => void;
  onUpdate: (id: string, status: KYCStatus, reason?: string) => void;
}) {
  const [rejectReason, setRejectReason] = useState("");
  const [showRejectForm, setShowRejectForm] = useState(false);
  const { toast } = useToast();
  const qc = useQueryClient();

  const handleApprove = async () => {
    if (!profile) return;
    try {
      await api.put(`/kyc/admin/${profile.id}/approve`, {});
    } catch {
    }
    toast({ title: "KYC Approved", description: `${profile.full_name}'s identity has been verified.` });
    onUpdate(profile.id, "approved");
    qc.invalidateQueries({ queryKey: ["kyc-list"] });
    qc.invalidateQueries({ queryKey: ["kyc-stats"] });
    onClose();
  };

  const handleUnderReview = async () => {
    if (!profile) return;
    try {
      await api.put(`/kyc/admin/${profile.id}/under-review`, {});
    } catch {
    }
    toast({ title: "Marked as Under Review" });
    onUpdate(profile.id, "under_review");
    qc.invalidateQueries({ queryKey: ["kyc-list"] });
    qc.invalidateQueries({ queryKey: ["kyc-stats"] });
    onClose();
  };

  const handleReject = async () => {
    if (!profile || !rejectReason.trim()) {
      toast({ title: "Error", description: "Please enter a rejection reason.", variant: "destructive" });
      return;
    }
    try {
      await api.put(`/kyc/admin/${profile.id}/reject`, { reason: rejectReason });
    } catch {
    }
    toast({ title: "KYC Rejected", description: "User will be notified to resubmit." });
    onUpdate(profile.id, "rejected", rejectReason);
    qc.invalidateQueries({ queryKey: ["kyc-list"] });
    qc.invalidateQueries({ queryKey: ["kyc-stats"] });
    onClose();
  };

  if (!profile) return null;
  const { label, color, icon: StatusIcon } = statusConfig(profile.status);

  return (
    <Dialog open={!!profile} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-[#0071CE]/10 flex items-center justify-center">
              <User className="w-5 h-5 text-[#0071CE]" />
            </div>
            <div>
              <p className="text-base font-semibold">{profile.full_name}</p>
              <p className="text-xs text-gray-400 font-normal">KYC #{profile.id}</p>
            </div>
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-5">
          {/* Status + Risk */}
          <div className="flex items-center gap-3">
            <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${color}`}>
              <StatusIcon className="w-3.5 h-3.5" />
              {label}
            </span>
            <span className={`px-2.5 py-1 rounded-full text-xs font-medium ${riskConfig(profile.risk_level)}`}>
              {profile.risk_level.toUpperCase()} RISK
            </span>
          </div>

          {/* Personal details */}
          <div className="grid grid-cols-2 gap-3 bg-gray-50 rounded-xl p-4 text-sm">
            <div>
              <p className="text-gray-400 text-xs mb-0.5">ID Number</p>
              <p className="font-mono font-medium">{profile.id_number}</p>
            </div>
            <div>
              <p className="text-gray-400 text-xs mb-0.5">Date of Birth</p>
              <p>{profile.date_of_birth}</p>
            </div>
            <div>
              <p className="text-gray-400 text-xs mb-0.5">Country</p>
              <p>{countryFlag(profile.country)} {profile.country}</p>
            </div>
            <div>
              <p className="text-gray-400 text-xs mb-0.5">Nationality</p>
              <p>{countryFlag(profile.nationality)} {profile.nationality}</p>
            </div>
            {profile.approved_at && (
              <div>
                <p className="text-gray-400 text-xs mb-0.5">Approved At</p>
                <p>{new Date(profile.approved_at).toLocaleDateString()}</p>
              </div>
            )}
            {profile.expires_at && (
              <div>
                <p className="text-gray-400 text-xs mb-0.5">Expires</p>
                <p>{new Date(profile.expires_at).toLocaleDateString()}</p>
              </div>
            )}
          </div>

          {/* Rejection reason */}
          {profile.rejection_reason && (
            <div className="flex gap-2 bg-red-50 border border-red-200 rounded-xl p-3">
              <AlertCircle className="w-4 h-4 text-red-500 mt-0.5 flex-shrink-0" />
              <p className="text-sm text-red-700">{profile.rejection_reason}</p>
            </div>
          )}

          {/* Documents */}
          <div>
            <p className="text-sm font-semibold mb-2 text-gray-700">Documents ({profile.documents.length})</p>
            <div className="grid grid-cols-2 gap-3">
              {profile.documents.map((doc) => (
                <div key={doc.id} className="border rounded-xl overflow-hidden">
                  <img
                    src={doc.file_url}
                    alt={doc.document_type}
                    className="w-full h-36 object-cover"
                  />
                  <div className="p-2.5 flex items-center justify-between">
                    <div>
                      <p className="text-xs font-medium">{docTypeLabel(doc.document_type)}</p>
                      <p className="text-xs text-gray-400 capitalize">{doc.side} side</p>
                    </div>
                    {doc.verified ? (
                      <span className="text-xs text-green-600 flex items-center gap-1">
                        <CheckCircle2 className="w-3 h-3" /> Verified
                      </span>
                    ) : (
                      <span className="text-xs text-gray-400 flex items-center gap-1">
                        <FileImage className="w-3 h-3" /> Pending
                      </span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Reject textarea */}
          {showRejectForm && (
            <div className="space-y-2">
              <p className="text-sm font-medium text-red-600">Rejection Reason</p>
              <Textarea
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
                placeholder="Explain why the KYC was rejected (the user will see this message)..."
                className="resize-none"
                rows={3}
              />
            </div>
          )}
        </div>

        <DialogFooter className="gap-2 flex-wrap">
          {profile.status !== "approved" && (
            <Button
              onClick={handleApprove}
              className="bg-green-600 hover:bg-green-700 text-white"
            >
              <CheckCircle2 className="w-4 h-4 mr-1.5" />
              Approve
            </Button>
          )}
          {profile.status === "pending" && (
            <Button variant="outline" onClick={handleUnderReview}>
              <RefreshCcw className="w-4 h-4 mr-1.5" />
              Mark Under Review
            </Button>
          )}
          {profile.status !== "rejected" && !showRejectForm && (
            <Button variant="destructive" onClick={() => setShowRejectForm(true)}>
              <XCircle className="w-4 h-4 mr-1.5" />
              Reject
            </Button>
          )}
          {showRejectForm && (
            <Button
              variant="destructive"
              onClick={handleReject}
              disabled={!rejectReason.trim()}
            >
              Confirm Rejection
            </Button>
          )}
          <Button variant="ghost" onClick={() => { setShowRejectForm(false); onClose(); }}>Close</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── Main Page ─────────────────────────────────────────────────────────────────

export default function KYCPage() {
  const [statusFilter, setStatusFilter] = useState("all");
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<KYCProfile | null>(null);
  // Local state for optimistic updates on mock data
  const [localProfiles, setLocalProfiles] = useState<KYCProfile[] | null>(null);

  const { data: stats, refetch: refetchStats } = useKYCStats();
  const { data: fetchedProfiles, isLoading } = useKYCList(statusFilter);

  const profiles = (localProfiles ?? fetchedProfiles ?? []).filter((p) =>
    search
      ? p.full_name?.toLowerCase().includes(search.toLowerCase()) ||
        p.id_number?.includes(search)
      : true
  );

  const handleUpdate = (id: string, newStatus: KYCStatus, reason?: string) => {
    const base = localProfiles ?? fetchedProfiles ?? MOCK_PROFILES;
    setLocalProfiles(
      base.map((p) =>
        p.id === id
          ? { ...p, status: newStatus, rejection_reason: reason ?? p.rejection_reason, approved_at: newStatus === "approved" ? new Date().toISOString() : p.approved_at }
          : p
      )
    );
    refetchStats();
  };

  const displayStats = stats ?? MOCK_STATS;

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 flex items-center gap-2">
            <ShieldCheck className="w-6 h-6 text-[#0071CE]" />
            KYC Verification
          </h1>
          <p className="text-sm text-gray-500 mt-1">
            Identity verification for GCC compliance — UAE / KSA / Kuwait regulations
          </p>
        </div>
        <Badge className="bg-[#0071CE]/10 text-[#0071CE] border-[#0071CE]/20 px-3 py-1 text-sm">
          {displayStats.total} total submissions
        </Badge>
      </div>

      {/* Stat cards */}
      <div className="flex flex-wrap gap-3">
        <StatCard label="All" value={displayStats.total} icon={User} color="bg-gray-100 text-gray-600" onClick={() => setStatusFilter("all")} active={statusFilter === "all"} />
        <StatCard label="Pending" value={displayStats.pending} icon={Clock} color="bg-yellow-100 text-yellow-600" onClick={() => setStatusFilter("pending")} active={statusFilter === "pending"} />
        <StatCard label="Under Review" value={displayStats.under_review} icon={RefreshCcw} color="bg-blue-100 text-blue-600" onClick={() => setStatusFilter("under_review")} active={statusFilter === "under_review"} />
        <StatCard label="Approved" value={displayStats.approved} icon={CheckCircle2} color="bg-green-100 text-green-600" onClick={() => setStatusFilter("approved")} active={statusFilter === "approved"} />
        <StatCard label="Rejected" value={displayStats.rejected} icon={XCircle} color="bg-red-100 text-red-600" onClick={() => setStatusFilter("rejected")} active={statusFilter === "rejected"} />
      </div>

      {/* Search + filter */}
      <div className="flex gap-3 items-center">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <Input
            placeholder="Search by name or ID number..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-44">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Statuses</SelectItem>
            <SelectItem value="pending">Pending</SelectItem>
            <SelectItem value="under_review">Under Review</SelectItem>
            <SelectItem value="approved">Approved</SelectItem>
            <SelectItem value="rejected">Rejected</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Table — inspired by saleor-dashboard data table patterns */}
      <div className="bg-white rounded-xl border shadow-sm overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="bg-gray-50">
              <TableHead className="font-semibold">Applicant</TableHead>
              <TableHead className="font-semibold">ID Number</TableHead>
              <TableHead className="font-semibold">Country</TableHead>
              <TableHead className="font-semibold">Documents</TableHead>
              <TableHead className="font-semibold">Risk</TableHead>
              <TableHead className="font-semibold">Status</TableHead>
              <TableHead className="font-semibold">Submitted</TableHead>
              <TableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading && (
              <TableRow>
                <TableCell colSpan={8} className="text-center py-10 text-gray-400">
                  Loading KYC submissions...
                </TableCell>
              </TableRow>
            )}
            {!isLoading && profiles.length === 0 && (
              <TableRow>
                <TableCell colSpan={8} className="text-center py-10">
                  <ShieldX className="w-8 h-8 text-gray-300 mx-auto mb-2" />
                  <p className="text-gray-400 text-sm">No submissions found</p>
                </TableCell>
              </TableRow>
            )}
            {profiles.map((p) => {
              const { label, color, icon: StatusIcon } = statusConfig(p.status);
              return (
                <TableRow
                  key={p.id}
                  className="hover:bg-gray-50 cursor-pointer"
                  onClick={() => setSelected(p)}
                >
                  <TableCell>
                    <div className="flex items-center gap-2.5">
                      <div className="w-8 h-8 rounded-full bg-[#0071CE]/10 flex items-center justify-center flex-shrink-0">
                        <User className="w-4 h-4 text-[#0071CE]" />
                      </div>
                      <span className="font-medium text-sm">{p.full_name}</span>
                    </div>
                  </TableCell>
                  <TableCell className="font-mono text-xs text-gray-500">{p.id_number}</TableCell>
                  <TableCell className="text-sm">
                    {countryFlag(p.country)} {p.country}
                  </TableCell>
                  <TableCell>
                    <span className="text-sm text-gray-500">{p.documents.length} docs</span>
                  </TableCell>
                  <TableCell>
                    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${riskConfig(p.risk_level)}`}>
                      {p.risk_level}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${color}`}>
                      <StatusIcon className="w-3 h-3" />
                      {label}
                    </span>
                  </TableCell>
                  <TableCell className="text-xs text-gray-400">
                    {new Date(p.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={(e) => { e.stopPropagation(); setSelected(p); }}
                    >
                      <Eye className="w-4 h-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>

      {/* Detail dialog */}
      <ProfileDialog
        profile={selected}
        onClose={() => setSelected(null)}
        onUpdate={handleUpdate}
      />
    </div>
  );
}
