import { Router, type IRouter } from "express";

const router: IRouter = Router();

interface KYCProfile {
  id: string;
  user_id: string;
  status: "pending" | "under_review" | "approved" | "rejected" | "not_submitted";
  full_name?: string;
  id_number?: string;
  country?: string;
  nationality?: string;
  date_of_birth?: string;
  rejection_reason?: string;
  approved_at?: string;
  expires_at?: string;
  risk_level: "low" | "medium" | "high";
  documents: KYCDocument[];
  created_at: string;
  updated_at: string;
}

interface KYCDocument {
  id: string;
  kyc_profile_id: string;
  document_type: string;
  file_url: string;
  side: "front" | "back";
  verified: boolean;
  mime_type?: string;
  created_at: string;
}

// ── In-memory mock store (replaced by PostgreSQL in production Go backend) ───
const kycProfiles: KYCProfile[] = [
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
      {
        id: "doc-001",
        kyc_profile_id: "kyc-001",
        document_type: "emirates_id",
        file_url: "https://placehold.co/400x250?text=Emirates+ID+Front",
        side: "front",
        verified: false,
        mime_type: "image/jpeg",
        created_at: new Date(Date.now() - 2 * 86400000).toISOString(),
      },
      {
        id: "doc-002",
        kyc_profile_id: "kyc-001",
        document_type: "selfie",
        file_url: "https://placehold.co/300x300?text=Selfie",
        side: "front",
        verified: false,
        mime_type: "image/jpeg",
        created_at: new Date(Date.now() - 2 * 86400000).toISOString(),
      },
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
      {
        id: "doc-003",
        kyc_profile_id: "kyc-002",
        document_type: "passport",
        file_url: "https://placehold.co/400x280?text=Passport",
        side: "front",
        verified: false,
        mime_type: "image/jpeg",
        created_at: new Date(Date.now() - 86400000).toISOString(),
      },
      {
        id: "doc-004",
        kyc_profile_id: "kyc-002",
        document_type: "selfie",
        file_url: "https://placehold.co/300x300?text=Selfie",
        side: "front",
        verified: false,
        mime_type: "image/jpeg",
        created_at: new Date(Date.now() - 86400000).toISOString(),
      },
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
      {
        id: "doc-005",
        kyc_profile_id: "kyc-003",
        document_type: "national_id",
        file_url: "https://placehold.co/400x250?text=National+ID",
        side: "front",
        verified: true,
        mime_type: "image/jpeg",
        created_at: new Date(Date.now() - 5 * 86400000).toISOString(),
      },
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
    rejection_reason: "Document image is blurry and unreadable. Please resubmit with a clear photo.",
    risk_level: "medium",
    documents: [
      {
        id: "doc-006",
        kyc_profile_id: "kyc-004",
        document_type: "national_id",
        file_url: "https://placehold.co/400x250?text=Blurry+ID",
        side: "front",
        verified: false,
        mime_type: "image/jpeg",
        created_at: new Date(Date.now() - 7 * 86400000).toISOString(),
      },
    ],
    created_at: new Date(Date.now() - 7 * 86400000).toISOString(),
    updated_at: new Date(Date.now() - 4 * 86400000).toISOString(),
  },
];

// ── GET /api/v1/kyc/admin/stats ──────────────────────────────────────────────
router.get("/v1/kyc/admin/stats", (_req, res) => {
  const stats = {
    total: kycProfiles.length,
    pending: kycProfiles.filter((p) => p.status === "pending").length,
    under_review: kycProfiles.filter((p) => p.status === "under_review").length,
    approved: kycProfiles.filter((p) => p.status === "approved").length,
    rejected: kycProfiles.filter((p) => p.status === "rejected").length,
  };
  res.json({ success: true, data: stats });
});

// ── GET /api/v1/kyc/admin/list ───────────────────────────────────────────────
router.get("/v1/kyc/admin/list", (req, res) => {
  let filtered = [...kycProfiles];
  if (req.query.status) {
    filtered = filtered.filter((p) => p.status === req.query.status);
  }
  const page = parseInt((req.query.page as string) || "1");
  const perPage = parseInt((req.query.per_page as string) || "20");
  const total = filtered.length;
  const items = filtered.slice((page - 1) * perPage, page * perPage);
  res.json({
    success: true,
    data: items,
    meta: { total, page, per_page: perPage, pages: Math.ceil(total / perPage) },
  });
});

// ── GET /api/v1/kyc/admin/:id ────────────────────────────────────────────────
router.get("/v1/kyc/admin/:id", (req, res) => {
  const profile = kycProfiles.find((p) => p.id === req.params.id);
  if (!profile) {
    res.status(404).json({ error: "KYC profile not found" });
    return;
  }
  res.json({ success: true, data: profile });
});

// ── PUT /api/v1/kyc/admin/:id/approve ───────────────────────────────────────
router.put("/v1/kyc/admin/:id/approve", (req, res) => {
  const idx = kycProfiles.findIndex((p) => p.id === req.params.id);
  if (idx === -1) {
    res.status(404).json({ error: "KYC profile not found" });
    return;
  }
  kycProfiles[idx] = {
    ...kycProfiles[idx],
    status: "approved",
    approved_at: new Date().toISOString(),
    expires_at: new Date(Date.now() + 365 * 86400000).toISOString(),
    rejection_reason: "",
    updated_at: new Date().toISOString(),
    documents: kycProfiles[idx].documents.map((d) => ({ ...d, verified: true })),
  };
  res.json({ success: true, data: { message: "KYC approved." } });
});

// ── PUT /api/v1/kyc/admin/:id/reject ────────────────────────────────────────
router.put("/v1/kyc/admin/:id/reject", (req, res) => {
  const idx = kycProfiles.findIndex((p) => p.id === req.params.id);
  if (idx === -1) {
    res.status(404).json({ error: "KYC profile not found" });
    return;
  }
  const { reason } = req.body;
  if (!reason) {
    res.status(400).json({ error: "rejection reason required" });
    return;
  }
  kycProfiles[idx] = {
    ...kycProfiles[idx],
    status: "rejected",
    rejection_reason: reason,
    updated_at: new Date().toISOString(),
  };
  res.json({ success: true, data: { message: "KYC rejected." } });
});

// ── PUT /api/v1/kyc/admin/:id/under-review ──────────────────────────────────
router.put("/v1/kyc/admin/:id/under-review", (req, res) => {
  const idx = kycProfiles.findIndex((p) => p.id === req.params.id);
  if (idx === -1) {
    res.status(404).json({ error: "KYC profile not found" });
    return;
  }
  kycProfiles[idx] = {
    ...kycProfiles[idx],
    status: "under_review",
    updated_at: new Date().toISOString(),
  };
  res.json({ success: true, data: { message: "KYC marked as under review." } });
});

export default router;
