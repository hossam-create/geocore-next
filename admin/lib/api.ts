import axios from "axios";

const api = axios.create({
  baseURL: "/api/v1",
  headers: { "Content-Type": "application/json" },
});

function getErrorStatus(error: unknown): number | undefined {
  return (error as { response?: { status?: number } } | null)?.response?.status;
}

function isRouteMissing(status: number | undefined): boolean {
  return status === 404 || status === 405;
}

async function tryEndpointFallbacks<T>(requests: Array<() => Promise<T>>): Promise<T> {
  let lastError: unknown;
  for (const makeRequest of requests) {
    try {
      return await makeRequest();
    } catch (error) {
      lastError = error;
      if (isRouteMissing(getErrorStatus(error))) {
        continue;
      }
      throw error;
    }
  }
  throw lastError;
}

api.interceptors.request.use((config) => {
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("admin_token");
    if (token) config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401 && typeof window !== "undefined") {
      localStorage.removeItem("admin_token");
      localStorage.removeItem("admin_user");
      window.location.href = "/login";
    }
    return Promise.reject(err);
  }
);

export default api;

// ── Settings ────────────────────────────────────────────────────────────────

export const settingsApi = {
  getAll: () => api.get("/admin/settings").then((r) => r.data?.data ?? r.data),
  getByCategory: (cat: string) =>
    api.get(`/admin/settings/${cat}`).then((r) => r.data?.data ?? r.data),
  update: (key: string, value: unknown) =>
    api.put(`/admin/settings/${key}`, { value }),
  bulkUpdate: (settings: Record<string, unknown>) =>
    api.put("/admin/settings/bulk", { settings }),
};

// ── Feature Flags ───────────────────────────────────────────────────────────

export const featuresApi = {
  getAll: () => api.get("/admin/features").then((r) => r.data?.data ?? r.data),
  update: (key: string, data: { enabled?: boolean; rollout_pct?: number; allowed_groups?: string[] }) =>
    api.put(`/admin/features/${key}`, data),
};

// ── Users ───────────────────────────────────────────────────────────────────

export const usersApi = {
  list: (params?: Record<string, string>) =>
    api.get("/admin/users", { params }).then((r) => r.data),
  get: (id: string) => api.get(`/admin/users/${id}`).then((r) => r.data?.data ?? r.data),
  update: (id: string, data: Record<string, unknown>) => api.put(`/admin/users/${id}`, data),
  ban: (id: string, reason?: string) => api.post(`/admin/users/${id}/ban`, { reason: reason ?? "Banned by admin" }),
  unban: (id: string) => api.post(`/admin/users/${id}/unban`),
  delete: (id: string) => api.delete(`/admin/users/${id}`),
  verify: (id: string) => api.put(`/admin/users/${id}/verify`),
  changeRole: (id: string, role: string) => api.put(`/admin/users/${id}/role`, { role }),
  changeGroup: (id: string, groupId: number) => api.put(`/admin/users/${id}/group`, { group_id: groupId }),
  suspend: (id: string, until: string) => api.put(`/admin/users/${id}/suspend`, { suspended_until: until }),
  impersonate: (id: string) => api.post(`/admin/users/${id}/impersonate`).then((r) => r.data?.data ?? r.data),
  listings: (id: string) => api.get(`/admin/users/${id}/listings`).then((r) => r.data),
  orders: (id: string) => api.get(`/admin/users/${id}/orders`).then((r) => r.data),
};

// ── Listings ────────────────────────────────────────────────────────────────

export const listingsApi = {
  list: (params?: Record<string, string>) =>
    api.get("/admin/listings", { params }).then((r) => r.data),
  pending: () => api.get("/admin/listings/pending").then((r) => r.data),
  get: (id: string) => api.get(`/admin/listings/${id}`).then((r) => r.data?.data ?? r.data),
  update: (id: string, data: Record<string, unknown>) => api.put(`/admin/listings/${id}`, data),
  approve: (id: string) => api.put(`/admin/listings/${id}/approve`),
  reject: (id: string, reason = "Rejected by admin") =>
    api.put(`/admin/listings/${id}/reject`, { reason }),
  feature: (id: string) => api.put(`/admin/listings/${id}/feature`),
  extend: (id: string, days: number) => api.put(`/admin/listings/${id}/extend`, { days }),
  delete: (id: string) => api.delete(`/admin/listings/${id}`),
  bulkApprove: (ids: string[]) => api.post("/admin/listings/bulk-approve", { ids }),
  bulkReject: (ids: string[]) => api.post("/admin/listings/bulk-reject", { ids }),
  bulkDelete: (ids: string[]) => api.post("/admin/listings/bulk-delete", { ids }),
  moderation: (params?: Record<string, string>) =>
    api.get("/admin/listings/moderation", { params }).then((r) => r.data?.data ?? r.data),
  bulk: (ids: string[], action: "approve" | "reject") =>
    api.post("/admin/listings/bulk", { ids, action }),
};

// ── Orders ──────────────────────────────────────────────────────────────────

export const ordersApi = {
  list: async (params?: Record<string, string>) => {
    try {
      return await api.get("/admin/orders", { params }).then((r) => r.data);
    } catch (error) {
      const status = getErrorStatus(error);
      if (status === 404 || status === 405) {
        return [];
      }
      throw error;
    }
  },
};

// ── Auctions ────────────────────────────────────────────────────────────────

export const auctionsApi = {
  canModerate: async () => {
    const probeId = "capability-probe";
    const probes: Array<() => Promise<unknown>> = [
      () => api.put(`/admin/auctions/${probeId}/pause`),
      () => api.post(`/admin/auctions/${probeId}/pause`),
      () => api.put(`/auctions/${probeId}/pause`),
      () => api.post(`/auctions/${probeId}/pause`),
      () => api.put(`/admin/auctions/${probeId}/cancel`),
      () => api.post(`/admin/auctions/${probeId}/cancel`),
      () => api.put(`/auctions/${probeId}/cancel`),
      () => api.post(`/auctions/${probeId}/cancel`),
    ];

    for (const probe of probes) {
      try {
        await probe();
        return true;
      } catch (error) {
        if (isRouteMissing(getErrorStatus(error))) {
          continue;
        }
        // Any non-missing status (401/400/422...) means the endpoint exists.
        return true;
      }
    }

    return false;
  },
  list: async (params?: Record<string, string>) => {
    try {
      return await api.get("/admin/auctions", { params }).then((r) => r.data);
    } catch (error) {
      const status = (error as { response?: { status?: number } } | null)?.response?.status;
      if (status === 404 || status === 405) {
        return api.get("/auctions", { params }).then((r) => r.data);
      }
      throw error;
    }
  },
  pause: async (id: string) => {
    try {
      return await tryEndpointFallbacks([
        () => api.put(`/admin/auctions/${id}/pause`),
        () => api.post(`/admin/auctions/${id}/pause`),
        () => api.put(`/auctions/${id}/pause`),
        () => api.post(`/auctions/${id}/pause`),
      ]);
    } catch (error) {
      if (isRouteMissing(getErrorStatus(error))) {
        throw new Error("Auction pause action is not available on this backend yet.");
      }
      throw error;
    }
  },
  cancel: async (id: string) => {
    try {
      return await tryEndpointFallbacks([
        () => api.put(`/admin/auctions/${id}/cancel`),
        () => api.post(`/admin/auctions/${id}/cancel`),
        () => api.put(`/auctions/${id}/cancel`),
        () => api.post(`/auctions/${id}/cancel`),
      ]);
    } catch (error) {
      if (isRouteMissing(getErrorStatus(error))) {
        throw new Error("Auction cancel action is not available on this backend yet.");
      }
      throw error;
    }
  },
  get: (id: string) => api.get(`/admin/auctions/${id}`).then((r) => r.data?.data ?? r.data),
  pending: () => api.get("/admin/auctions/pending").then((r) => r.data),
  approve: (id: string) => api.put(`/admin/auctions/${id}/approve`),
  reject: (id: string, reason?: string) => api.put(`/admin/auctions/${id}/reject`, { reason }),
  extend: (id: string, hours: number) => api.put(`/admin/auctions/${id}/extend`, { hours }),
  bids: (id: string) => api.get(`/admin/auctions/${id}/bids`).then((r) => r.data?.data ?? r.data),
  deleteBid: (auctionId: string, bidId: string) => api.delete(`/admin/auctions/${auctionId}/bids/${bidId}`),
};

// ── Categories ──────────────────────────────────────────────────────────────

export const categoriesApi = {
  list: () => api.get("/admin/categories").then((r) => r.data?.data ?? r.data),
  create: (data: Record<string, unknown>) => api.post("/admin/categories", data),
  update: (id: string, data: Record<string, unknown>) => api.put(`/admin/categories/${id}`, data),
  delete: (id: string) => api.delete(`/admin/categories/${id}`),
  reorder: (id: string, data: Record<string, unknown>) => api.put(`/admin/categories/${id}/reorder`, data),
  fields: (id: string) => api.get(`/admin/categories/${id}/fields`).then((r) => r.data?.data ?? r.data),
  createField: (id: string, data: Record<string, unknown>) => api.post(`/admin/categories/${id}/fields`, data),
  updateField: (id: string, fieldId: string, data: Record<string, unknown>) => api.put(`/admin/categories/${id}/fields/${fieldId}`, data),
  deleteField: (id: string, fieldId: string) => api.delete(`/admin/categories/${id}/fields/${fieldId}`),
};

// ── Storefronts ─────────────────────────────────────────────────────────────

export const storefrontsApi = {
  list: (params?: Record<string, string>) => api.get("/admin/storefronts", { params }).then((r) => r.data?.data ?? r.data),
  get: (id: string) => api.get(`/admin/storefronts/${id}`).then((r) => r.data?.data ?? r.data),
  approve: (id: string) => api.put(`/admin/storefronts/${id}/approve`),
  suspend: (id: string) => api.put(`/admin/storefronts/${id}/suspend`),
  feature: (id: string) => api.put(`/admin/storefronts/${id}/feature`),
  delete: (id: string) => api.delete(`/admin/storefronts/${id}`),
};

// ── Invoices ────────────────────────────────────────────────────────────────

export const invoicesApi = {
  list: (params?: Record<string, string>) => api.get("/admin/invoices", { params }).then((r) => r.data?.data ?? r.data),
  get: (id: number) => api.get(`/admin/invoices/${id}`).then((r) => r.data?.data ?? r.data),
};

// ── Tickets ─────────────────────────────────────────────────────────────────

export const ticketsApi = {
  list: (params?: Record<string, string>) =>
    api.get("/admin/tickets", { params }).then((r) => r.data),
  get: (id: string) => api.get(`/admin/tickets/${id}`).then((r) => r.data?.data ?? r.data),
  reply: (id: string, body: string) => api.post(`/admin/tickets/${id}/reply`, { body }),
  updateStatus: (id: string, status: string) => api.patch(`/admin/tickets/${id}`, { status }),
};

// ── Dashboard ───────────────────────────────────────────────────────────────

export const dashboardApi = {
  stats: () => api.get("/admin/stats").then((r) => r.data?.data ?? r.data),
  revenue: () => api.get("/admin/revenue").then((r) => r.data?.data ?? r.data),
};

// ── Analytics ───────────────────────────────────────────────────────────────

export const analyticsApi = {
  overview: (params?: Record<string, string>) =>
    api.get("/admin/analytics/overview", { params }).then((r) => r.data?.data ?? r.data),
  traffic: (params?: Record<string, string>) =>
    api.get("/admin/analytics/traffic", { params }).then((r) => r.data?.data ?? r.data),
  topCategories: (params?: Record<string, string>) =>
    api.get("/admin/analytics/top-categories", { params }).then((r) => r.data?.data ?? r.data),
  funnel: (params?: Record<string, string>) =>
    api.get("/admin/analytics/funnel", { params }).then((r) => r.data?.data ?? r.data),
  timeseries: (params?: Record<string, string>) =>
    api.get("/admin/analytics/timeseries", { params }).then((r) => r.data?.data ?? r.data),
};

// ── Reports ─────────────────────────────────────────────────────────────────

export const reportsApi = {
  list: (params?: Record<string, string>) =>
    api.get("/admin/reports", { params }).then((r) => r.data),
  review: (id: string, data: { status: string; admin_note?: string }) =>
    api.patch(`/admin/reports/${id}`, data),
};

// ── Disputes ────────────────────────────────────────────────────────────────

export const disputesApi = {
  list: (params?: Record<string, string>) =>
    api.get("/disputes", { params }).then((r) => r.data),
  get: (id: string) => api.get(`/disputes/${id}`).then((r) => r.data?.data ?? r.data),
  resolve: (id: string, data: Record<string, unknown>) => api.put(`/disputes/${id}/resolve`, data),
  reply: (id: string, body: string) => api.post(`/disputes/${id}/messages`, { body }),
};

// ── Escrow ──────────────────────────────────────────────────────────────────

export const escrowApi = {
  list: (params?: Record<string, string>) =>
    api.get("/admin/payments/payouts", { params }).then((r) => r.data?.data ?? r.data),
  release: async (id: string, notes?: string) => {
    try {
      return await api.post(`/escrow/${id}/release`);
    } catch (error) {
      const status = (error as { response?: { status?: number } } | null)?.response?.status;
      if (status === 404 || status === 405) {
        return api.post("/payments/release-escrow", { escrow_id: id, notes });
      }
      throw error;
    }
  },
};

// ── Logs ────────────────────────────────────────────────────────────────────

export const logsApi = {
  list: (params?: Record<string, string>) =>
    api.get("/admin/logs", { params }).then((r) => r.data),
};

// ── Transactions ────────────────────────────────────────────────────────────

export const transactionsApi = {
  list: (params?: Record<string, string>) =>
    api.get("/admin/transactions", { params }).then((r) => r.data),
};

// ── KYC Admin (used as Custodii decision queue for now) ────────────────────

export const kycApi = {
  list: () => api.get("/kyc/admin/list").then((r) => r.data?.data ?? r.data),
  approve: (id: string, notes = "Approved by Custodii") =>
    api.put(`/kyc/admin/${id}/approve`, { notes }),
  reject: (id: string, reason = "Rejected by Custodii") =>
    api.put(`/kyc/admin/${id}/reject`, { reason }),
};

// ── User Groups ────────────────────────────────────────────────────────────

export const userGroupsApi = {
  list: () => api.get("/admin/user-groups").then((r) => r.data?.data ?? r.data),
  create: (data: Record<string, unknown>) => api.post("/admin/user-groups", data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/admin/user-groups/${id}`, data),
  delete: (id: number) => api.delete(`/admin/user-groups/${id}`),
};

// ── User Custom Fields ─────────────────────────────────────────────────────

export const userFieldsApi = {
  list: () => api.get("/admin/user-fields").then((r) => r.data?.data ?? r.data),
  create: (data: Record<string, unknown>) => api.post("/admin/user-fields", data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/admin/user-fields/${id}`, data),
  delete: (id: number) => api.delete(`/admin/user-fields/${id}`),
};

// ── Email Templates ────────────────────────────────────────────────────────

export const emailTemplatesApi = {
  list: () => api.get("/admin/email-templates").then((r) => r.data?.data ?? r.data),
  get: (slug: string) => api.get(`/admin/email-templates/${slug}`).then((r) => r.data?.data ?? r.data),
  update: (slug: string, data: Record<string, unknown>) => api.put(`/admin/email-templates/${slug}`, data),
  preview: (slug: string) => api.post(`/admin/email-templates/${slug}/preview`).then((r) => r.data?.data ?? r.data),
  test: (slug: string, email: string) => api.post(`/admin/email-templates/${slug}/test`, { email }),
};

// ── Static Pages ───────────────────────────────────────────────────────────

export const staticPagesApi = {
  list: () => api.get("/admin/pages").then((r) => r.data?.data ?? r.data),
  get: (id: number) => api.get(`/admin/pages/${id}`).then((r) => r.data?.data ?? r.data),
  create: (data: Record<string, unknown>) => api.post("/admin/pages", data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/admin/pages/${id}`, data),
  delete: (id: number) => api.delete(`/admin/pages/${id}`),
};

// ── Announcements ──────────────────────────────────────────────────────────

export const announcementsApi = {
  list: () => api.get("/admin/announcements").then((r) => r.data?.data ?? r.data),
  create: (data: Record<string, unknown>) => api.post("/admin/announcements", data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/admin/announcements/${id}`, data),
  delete: (id: number) => api.delete(`/admin/announcements/${id}`),
};

// ── Geography ──────────────────────────────────────────────────────────────

export const geographyApi = {
  list: () => api.get("/admin/geography").then((r) => r.data?.data ?? r.data),
  children: (id: number) => api.get(`/admin/geography/${id}/children`).then((r) => r.data?.data ?? r.data),
  create: (data: Record<string, unknown>) => api.post("/admin/geography", data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/admin/geography/${id}`, data),
  delete: (id: number) => api.delete(`/admin/geography/${id}`),
};

// ── Payment Gateways ───────────────────────────────────────────────────────

export const gatewaysApi = {
  list: () => api.get("/admin/payment-gateways").then((r) => r.data?.data ?? r.data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/admin/payment-gateways/${id}`, data),
};

// ── Discount Codes ─────────────────────────────────────────────────────────

export const discountCodesApi = {
  list: () => api.get("/admin/discount-codes").then((r) => r.data?.data ?? r.data),
  create: (data: Record<string, unknown>) => api.post("/admin/discount-codes", data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/admin/discount-codes/${id}`, data),
  delete: (id: number) => api.delete(`/admin/discount-codes/${id}`),
};

// ── Listing Extras ─────────────────────────────────────────────────────────

export const listingExtrasApi = {
  list: () => api.get("/admin/listing-extras").then((r) => r.data?.data ?? r.data),
  create: (data: Record<string, unknown>) => api.post("/admin/listing-extras", data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/admin/listing-extras/${id}`, data),
  delete: (id: number) => api.delete(`/admin/listing-extras/${id}`),
};

// ── Price Plans (extended) ─────────────────────────────────────────────────

export const plansApi = {
  list: () => api.get("/admin/plans").then((r) => r.data?.data ?? r.data),
  create: (data: Record<string, unknown>) => api.post("/admin/plans", data),
  update: (id: string, data: Record<string, unknown>) => api.put(`/admin/plans/${id}`, data),
  delete: (id: string) => api.delete(`/admin/plans/${id}`),
};

// ── Enhanced Dashboard ─────────────────────────────────────────────────────

export const dashboardFullApi = {
  get: () => api.get("/admin/dashboard").then((r) => r.data?.data ?? r.data),
};

// ── Trust & Safety ────────────────────────────────────────────────────────

export const trustApi = {
  listFlags: (params?: Record<string, string>) =>
    api.get("/admin/trust/flags", { params }).then((r) => r.data?.data ?? r.data),
  getFlag: (id: string) =>
    api.get(`/admin/trust/flags/${id}`).then((r) => r.data?.data ?? r.data),
  resolveFlag: (id: string, data: { status: string; notes?: string }) =>
    api.patch(`/admin/trust/flags/${id}/resolve`, data),
  getStats: () =>
    api.get("/admin/trust/stats").then((r) => r.data?.data ?? r.data),
  bulkResolve: (ids: string[], status: string) =>
    api.post("/admin/trust/flags/bulk-resolve", { ids, status }),
};

// ── Sellers ────────────────────────────────────────────────────────────────

export const sellersApi = {
  top: (params?: Record<string, string>) =>
    api.get("/admin/sellers/top", { params }).then((r) => r.data?.data ?? r.data),
  report: (id: string) =>
    api.get(`/admin/sellers/${id}/report`).then((r) => r.data?.data ?? r.data),
};

// ── Ops Health ─────────────────────────────────────────────────────────────

export const opsApi = {
  health: () =>
    api.get("/admin/ops/health").then((r) => r.data?.data ?? r.data),
  jobs: () =>
    api.get("/admin/ops/jobs").then((r) => r.data?.data ?? r.data),
  triggerJob: (name: string) =>
    api.post(`/admin/ops/jobs/${name}/trigger`),
};

// ── Compliance ─────────────────────────────────────────────────────────────

export const complianceApi = {
  gdprExport: (userId: string) =>
    api.get(`/admin/compliance/gdpr/${userId}`).then((r) => r.data?.data ?? r.data),
  gdprDelete: (userId: string) =>
    api.delete(`/admin/compliance/gdpr/${userId}`),
  auditLogs: (params?: Record<string, string>) =>
    api.get("/admin/compliance/audit", { params }).then((r) => r.data?.data ?? r.data),
  exportAuditCsv: (params?: Record<string, string>) =>
    api.get("/admin/compliance/audit/export", { params, responseType: "blob" }),
};

// ── Banner Ads ─────────────────────────────────────────────────────────────

export const adsApi = {
  list: (params?: Record<string, string>) =>
    api.get("/admin/ads", { params }).then((r) => r.data?.data ?? r.data),
  create: (data: Record<string, unknown>) =>
    api.post("/admin/ads", data).then((r) => r.data?.data ?? r.data),
  update: (id: string, data: Record<string, unknown>) =>
    api.put(`/admin/ads/${id}`, data).then((r) => r.data?.data ?? r.data),
  delete: (id: string) =>
    api.delete(`/admin/ads/${id}`),
  toggle: (id: string) =>
    api.patch(`/admin/ads/${id}/toggle`).then((r) => r.data?.data ?? r.data),
};

// ── Chargebacks ────────────────────────────────────────────────────────────

export const chargebacksApi = {
  list: (params?: Record<string, string>) =>
    api.get("/admin/chargebacks", { params }).then((r) => r.data),
  get: (id: string) =>
    api.get(`/admin/chargebacks/${id}`).then((r) => r.data?.data ?? r.data),
  submitEvidence: (id: string, data: { evidence_type: string; file_url?: string; description?: string }) =>
    api.post(`/admin/chargebacks/${id}/evidence`, data).then((r) => r.data?.data ?? r.data),
};

// ── Addon Marketplace ──────────────────────────────────────────────────────

export const addonsApi = {
  list: (params?: Record<string, string>) =>
    api.get("/admin/addons", { params }).then((r) => r.data),
  get: (id: string) =>
    api.get(`/admin/addons/${id}`).then((r) => r.data?.data ?? r.data),
  stats: () =>
    api.get("/admin/addons/stats").then((r) => r.data?.data ?? r.data),
  install: (id: string) =>
    api.post(`/admin/addons/${id}/install`).then((r) => r.data?.data ?? r.data),
  uninstall: (id: string) =>
    api.post(`/admin/addons/${id}/uninstall`).then((r) => r.data?.data ?? r.data),
  enable: (id: string) =>
    api.post(`/admin/addons/${id}/enable`).then((r) => r.data?.data ?? r.data),
  disable: (id: string) =>
    api.post(`/admin/addons/${id}/disable`).then((r) => r.data?.data ?? r.data),
  updateConfig: (id: string, config: string) =>
    api.put(`/admin/addons/${id}/config`, { config }).then((r) => r.data?.data ?? r.data),
  reviews: (id: string) =>
    api.get(`/admin/addons/${id}/reviews`).then((r) => r.data?.data ?? r.data),
  addReview: (id: string, data: { rating: number; review?: string; version?: string }) =>
    api.post(`/admin/addons/${id}/reviews`, data).then((r) => r.data?.data ?? r.data),
};

// ── CMS (Content Management) ──────────────────────────────────────────────

export const cmsApi = {
  // Hero Slides
  slides: {
    list: () => api.get("/admin/cms/slides").then((r) => r.data?.data ?? r.data),
    create: (data: Record<string, unknown>) => api.post("/admin/cms/slides", data).then((r) => r.data?.data ?? r.data),
    update: (id: string, data: Record<string, unknown>) => api.put(`/admin/cms/slides/${id}`, data).then((r) => r.data?.data ?? r.data),
    delete: (id: string) => api.delete(`/admin/cms/slides/${id}`).then((r) => r.data),
    reorder: (order: string[]) => api.put("/admin/cms/slides/reorder", { order }).then((r) => r.data),
  },
  // Content Blocks
  blocks: {
    list: (params?: Record<string, string>) => api.get("/admin/cms/blocks", { params }).then((r) => r.data?.data ?? r.data),
    get: (slug: string) => api.get(`/admin/cms/blocks/${slug}`).then((r) => r.data?.data ?? r.data),
    create: (data: Record<string, unknown>) => api.post("/admin/cms/blocks", data).then((r) => r.data?.data ?? r.data),
    update: (slug: string, data: Record<string, unknown>) => api.put(`/admin/cms/blocks/${slug}`, data).then((r) => r.data?.data ?? r.data),
    delete: (slug: string) => api.delete(`/admin/cms/blocks/${slug}`).then((r) => r.data),
  },
  // Media Library
  media: {
    list: (params?: Record<string, string>) => api.get("/admin/cms/media", { params }).then((r) => r.data?.data ?? r.data),
    upload: (formData: FormData) => api.post("/admin/cms/media", formData, { headers: { "Content-Type": "multipart/form-data" } }).then((r) => r.data?.data ?? r.data),
    delete: (id: string) => api.delete(`/admin/cms/media/${id}`).then((r) => r.data),
  },
  // Site Settings
  settings: {
    list: (params?: Record<string, string>) => api.get("/admin/cms/settings", { params }).then((r) => r.data?.data ?? r.data),
    get: (key: string) => api.get(`/admin/cms/settings/${key}`).then((r) => r.data?.data ?? r.data),
    update: (key: string, value: string) => api.put(`/admin/cms/settings/${key}`, { value }).then((r) => r.data?.data ?? r.data),
    bulkUpdate: (data: Record<string, string>) => api.put("/admin/cms/settings/bulk", data).then((r) => r.data),
  },
  // Navigation
  nav: {
    list: (location?: string) => api.get("/admin/cms/nav", { params: location ? { location } : undefined }).then((r) => r.data?.data ?? r.data),
    create: (data: Record<string, unknown>) => api.post("/admin/cms/nav", data).then((r) => r.data?.data ?? r.data),
    update: (id: string, data: Record<string, unknown>) => api.put(`/admin/cms/nav/${id}`, data).then((r) => r.data?.data ?? r.data),
    delete: (id: string) => api.delete(`/admin/cms/nav/${id}`).then((r) => r.data),
    reorder: (order: string[]) => api.put("/admin/cms/nav/reorder", { order }).then((r) => r.data),
  },
};
