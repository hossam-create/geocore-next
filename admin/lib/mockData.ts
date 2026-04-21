/* Mock data for Phase 1 dashboard skeletons */

export const mockListings = [
  { id: "L-001", title: "iPhone 15 Pro Max", user: "Ali Ahmed", status: "pending", price: 1200, created: "2026-04-01T10:00:00Z" },
  { id: "L-002", title: "MacBook Pro M4", user: "Sara Mohamed", status: "flagged", price: 2500, created: "2026-04-01T11:00:00Z" },
  { id: "L-003", title: "Vintage Rolex", user: "Omar Hassan", status: "approved", price: 8500, created: "2026-04-01T09:00:00Z" },
  { id: "L-004", title: "Antique Vase", user: "Mona Ali", status: "pending", price: 450, created: "2026-03-31T15:00:00Z" },
  { id: "L-005", title: "Gaming PC RTX 5090", user: "Khaled Zein", status: "rejected", price: 3200, created: "2026-03-31T12:00:00Z" },
];

export const mockAuctions = [
  { id: "A-001", title: "Rare Painting", bids: 15, currentPrice: 5200, status: "live", endsAt: "2026-04-03T18:00:00Z" },
  { id: "A-002", title: "Gold Necklace", bids: 8, currentPrice: 1800, status: "live", endsAt: "2026-04-03T20:00:00Z" },
  { id: "A-003", title: "Classic Car", bids: 32, currentPrice: 45000, status: "ending", endsAt: "2026-04-02T23:30:00Z" },
  { id: "A-004", title: "Signed Jersey", bids: 0, currentPrice: 500, status: "scheduled", endsAt: "2026-04-05T12:00:00Z" },
];

export const mockOrders = [
  { id: "ORD-5001", buyer: "Ali Ahmed", seller: "Sara Mohamed", amount: 1200, status: "completed", date: "2026-04-01T14:00:00Z" },
  { id: "ORD-5002", buyer: "Omar Hassan", seller: "Mona Ali", amount: 450, status: "pending", date: "2026-04-02T10:00:00Z" },
  { id: "ORD-5003", buyer: "Khaled Zein", seller: "Layla Farid", amount: 3200, status: "in_progress", date: "2026-04-02T08:00:00Z" },
  { id: "ORD-5004", buyer: "Nour Samy", seller: "Ahmed Gamal", amount: 890, status: "cancelled", date: "2026-03-30T16:00:00Z" },
];

export const mockEscrow = [
  { id: "ESC-001", orderId: "ORD-5002", amount: 450, buyer: "Omar Hassan", seller: "Mona Ali", status: "hold", decision: "pending" },
  { id: "ESC-002", orderId: "ORD-5003", amount: 3200, buyer: "Khaled Zein", seller: "Layla Farid", status: "hold", decision: "approved" },
  { id: "ESC-003", orderId: "ORD-5001", amount: 1200, buyer: "Ali Ahmed", seller: "Sara Mohamed", status: "released", decision: "approved" },
  { id: "ESC-004", orderId: "ORD-5005", amount: 6700, buyer: "Youssef Ali", seller: "Hana Mostafa", status: "hold", decision: "rejected" },
];

export const mockDecisions = [
  { id: "DEC-001", asset: "Listing L-002", action: "approve_listing", riskScore: 87, status: "pending", source: "auto_flag", createdAt: "2026-04-02T10:30:00Z" },
  { id: "DEC-002", asset: "Escrow ESC-001", action: "release_funds", riskScore: 12, status: "pending", source: "escrow_service", createdAt: "2026-04-02T11:00:00Z" },
  { id: "DEC-003", asset: "Auction A-003", action: "extend_auction", riskScore: 45, status: "approved", source: "ops_team", createdAt: "2026-04-02T09:00:00Z" },
  { id: "DEC-004", asset: "User U-123", action: "unban_user", riskScore: 62, status: "rejected", source: "support", createdAt: "2026-04-01T16:00:00Z" },
  { id: "DEC-005", asset: "Listing L-004", action: "approve_listing", riskScore: 5, status: "pending", source: "auto_review", createdAt: "2026-04-02T12:00:00Z" },
];

export const mockAuditLogs = [
  { id: "AUD-001", action: "decision.approved", actor: "system", target: "DEC-003", details: "Auto-approved: risk < 30", timestamp: "2026-04-02T09:01:00Z" },
  { id: "AUD-002", action: "decision.rejected", actor: "admin@geocore.app", target: "DEC-004", details: "User ban upheld after review", timestamp: "2026-04-01T16:05:00Z" },
  { id: "AUD-003", action: "escrow.released", actor: "system", target: "ESC-003", details: "Auto-release after 48h confirmation", timestamp: "2026-04-01T14:30:00Z" },
  { id: "AUD-004", action: "listing.flagged", actor: "system", target: "L-002", details: "Flagged by fraud detection", timestamp: "2026-04-01T11:05:00Z" },
];

export const mockUsers = [
  { id: "U-001", name: "Ali Ahmed", email: "ali@example.com", role: "user", status: "active", joined: "2026-01-15T00:00:00Z" },
  { id: "U-002", name: "Sara Mohamed", email: "sara@example.com", role: "user", status: "active", joined: "2026-02-20T00:00:00Z" },
  { id: "U-003", name: "Omar Hassan", email: "omar@example.com", role: "user", status: "banned", joined: "2025-11-10T00:00:00Z" },
  { id: "U-004", name: "Admin User", email: "admin@geocore.app", role: "super_admin", status: "active", joined: "2025-10-01T00:00:00Z" },
];

export const mockTickets = [
  { id: "T-001", user: "Ali Ahmed", subject: "Payment not received", status: "open", priority: "high", created: "2026-04-02T08:00:00Z" },
  { id: "T-002", user: "Sara Mohamed", subject: "Can't access account", status: "in_progress", priority: "medium", created: "2026-04-01T14:00:00Z" },
  { id: "T-003", user: "Omar Hassan", subject: "Scam report", status: "open", priority: "urgent", created: "2026-04-02T10:00:00Z" },
];

export const mockDisputes = [
  { id: "D-001", buyer: "Ali Ahmed", seller: "Sara Mohamed", reason: "Item not as described", amount: 1200, status: "open", created: "2026-04-01T12:00:00Z" },
  { id: "D-002", buyer: "Omar Hassan", seller: "Mona Ali", reason: "Item not delivered", amount: 450, status: "in_progress", created: "2026-03-30T09:00:00Z" },
];

export const mockRiskAlerts = [
  { id: "RA-001", type: "suspicious_activity", severity: "high", user: "U-003", message: "Multiple failed payment attempts", createdAt: "2026-04-02T11:30:00Z" },
  { id: "RA-002", type: "fraud_detection", severity: "urgent", user: "U-005", message: "Account created from known VPN", createdAt: "2026-04-02T10:00:00Z" },
  { id: "RA-003", type: "velocity_check", severity: "medium", user: "U-002", message: "Unusual listing volume (15 in 1hr)", createdAt: "2026-04-02T09:00:00Z" },
];

export const mockDashboardStats = {
  totalUsers: 12453,
  activeListings: 3821,
  liveAuctions: 156,
  pendingDecisions: 23,
  revenue: 245600,
  gmv: 1280000,
  fraudRate: 2.3,
  decisionsApproved: 85,
  escrowHeld: 45200,
  openTickets: 18,
  openDisputes: 7,
};

export const mockTrustFlags = [
  { id: "TF-001", target_type: "user", target_id: "U-003", flag_type: "velocity_spike", severity: "high", source: "rule_engine", status: "open", notes: "21 bids in 1 hour", risk_score: 70, created_at: "2026-04-02T11:30:00Z" },
  { id: "TF-002", target_type: "listing", target_id: "L-002", flag_type: "prohibited_item", severity: "critical", source: "keyword_filter", status: "open", notes: "Contains banned keyword", risk_score: 90, created_at: "2026-04-02T10:00:00Z" },
  { id: "TF-003", target_type: "user", target_id: "U-005", flag_type: "new_account_high_value", severity: "medium", source: "rule_engine", status: "investigating", notes: "Account 3 days old, listed item at $800", risk_score: 60, created_at: "2026-04-02T09:00:00Z" },
  { id: "TF-004", target_type: "user", target_id: "U-006", flag_type: "mass_listing", severity: "high", source: "rule_engine", status: "open", notes: "52 listings in 24h", risk_score: 80, created_at: "2026-04-01T16:00:00Z" },
  { id: "TF-005", target_type: "listing", target_id: "L-010", flag_type: "repeated_reporter", severity: "low", source: "user_reports", status: "resolved", notes: "3 reports this week — reviewed, no action", risk_score: 30, created_at: "2026-04-01T12:00:00Z", resolved_at: "2026-04-01T14:00:00Z", resolved_by: "admin@geocore.app" },
];

export const mockTrustStats = {
  open_flags: 12,
  critical_flags: 3,
  auto_resolved_today: 47,
  manual_review_needed: 5,
  banned_today: 2,
  fraud_prevented_usd: 15400,
};

export const mockSellers = [
  { id: "S-001", username: "TechStore EG", gmv: 284500, avg_rating: 4.8, dispute_rate: 0.02, refund_rate: 0.01, flag_count: 0, total_sales: 342, joined: "2025-06-15T00:00:00Z", status: "active" },
  { id: "S-002", username: "Vintage Finds", gmv: 156200, avg_rating: 4.5, dispute_rate: 0.05, refund_rate: 0.03, flag_count: 1, total_sales: 189, joined: "2025-09-01T00:00:00Z", status: "active" },
  { id: "S-003", username: "AutoParts Pro", gmv: 98700, avg_rating: 3.9, dispute_rate: 0.12, refund_rate: 0.08, flag_count: 4, total_sales: 67, joined: "2025-11-20T00:00:00Z", status: "under_review" },
  { id: "S-004", username: "JewelryBox", gmv: 412000, avg_rating: 4.9, dispute_rate: 0.01, refund_rate: 0.01, flag_count: 0, total_sales: 520, joined: "2025-03-10T00:00:00Z", status: "active" },
  { id: "S-005", username: "QuickDeals", gmv: 23400, avg_rating: 3.2, dispute_rate: 0.25, refund_rate: 0.15, flag_count: 8, total_sales: 23, joined: "2026-03-28T00:00:00Z", status: "suspended" },
];

export const mockKYC = [
  { id: "KYC-001", user_id: "U-010", user_name: "Fatma Ibrahim", user_email: "fatma@example.com", kyc_status: "pending", phone_verified: true, id_document_url: "/uploads/kyc/id-u010.jpg", submitted_at: "2026-04-02T08:00:00Z" },
  { id: "KYC-002", user_id: "U-011", user_name: "Ahmed Nabil", user_email: "ahmed.n@example.com", kyc_status: "pending", phone_verified: false, submitted_at: "2026-04-01T15:00:00Z" },
  { id: "KYC-003", user_id: "U-012", user_name: "Mona Selim", user_email: "mona.s@example.com", kyc_status: "rejected", phone_verified: true, id_document_url: "/uploads/kyc/id-u012.jpg", submitted_at: "2026-03-30T10:00:00Z", reviewed_by: "admin@geocore.app", rejection_reason: "Document expired" },
  { id: "KYC-004", user_id: "U-013", user_name: "Kareem Adel", user_email: "kareem@example.com", kyc_status: "pending", phone_verified: true, id_document_url: "/uploads/kyc/id-u013.jpg", submitted_at: "2026-04-02T12:00:00Z" },
];

export const mockOpsHealth = {
  api_latency_p95_ms: 145,
  db_connections: 23,
  db_max_connections: 100,
  redis_memory_mb: 512,
  redis_max_memory_mb: 1024,
  active_auctions: 342,
  active_websocket_connections: 1204,
  job_queue_depth: 15,
  error_rate_1h: 0.002,
  last_payment_at: "2026-04-02T14:23:00Z",
  uptime_seconds: 864000,
};

export const mockComplianceAudit = [
  { id: "AUD-101", admin_user_id: "U-004", admin_name: "Admin User", action: "user.ban", target_type: "user", target_id: "U-003", old_value: "active", new_value: "banned", ip_address: "192.168.1.100", created_at: "2026-04-02T10:05:00Z" },
  { id: "AUD-102", admin_user_id: "U-004", admin_name: "Admin User", action: "listing.approve", target_type: "listing", target_id: "L-001", old_value: "pending", new_value: "approved", ip_address: "192.168.1.100", created_at: "2026-04-02T09:30:00Z" },
  { id: "AUD-103", admin_user_id: "system", admin_name: "System", action: "trust_flag.auto_created", target_type: "user", target_id: "U-006", old_value: "", new_value: "mass_listing", ip_address: "127.0.0.1", created_at: "2026-04-01T16:00:00Z" },
  { id: "AUD-104", admin_user_id: "U-004", admin_name: "Admin User", action: "settings.update", target_type: "setting", target_id: "payments.platform_fee_pct", old_value: "5.0", new_value: "4.5", ip_address: "192.168.1.100", created_at: "2026-04-01T11:20:00Z" },
  { id: "AUD-105", admin_user_id: "U-004", admin_name: "Admin User", action: "feature.toggle", target_type: "feature_flag", target_id: "feature.live_streaming", old_value: "false", new_value: "true", ip_address: "192.168.1.100", created_at: "2026-04-01T10:00:00Z" },
];
