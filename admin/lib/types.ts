export interface AdminSetting {
  key: string;
  value: string;
  type: "bool" | "string" | "number" | "select" | "json" | "secret";
  category: string;
  label: string;
  description?: string;
  options?: { value: string; label: string }[];
  is_public: boolean;
  is_secret: boolean;
  updated_at: string;
  updated_by?: string;
}

export interface CategoryGroup {
  category: string;
  settings: AdminSetting[];
}

export interface FeatureFlag {
  key: string;
  enabled: boolean;
  rollout_pct: number;
  allowed_groups?: string[];
  category?: string;
  description?: string;
  created_at: string;
}

export interface AdminAuditLog {
  id: string;
  admin_user_id: string;
  admin_name?: string;
  action: string;
  target_type: string;
  target_id: string;
  old_value?: string;
  new_value?: string;
  ip_address: string;
  user_agent?: string;
  created_at: string;
}

export interface User {
  id: string;
  name: string;
  email: string;
  role: string;
  is_active: boolean;
  is_banned: boolean;
  avatar_url?: string;
  created_at: string;
}

export interface Listing {
  id: string;
  title: string;
  price: number;
  currency: string;
  status: string;
  seller_id: string;
  seller_name?: string;
  category_name?: string;
  created_at: string;
  image_url?: string;
}

export interface Auction {
  id: string;
  listing_id: string;
  title?: string;
  start_price: number;
  current_bid: number;
  bid_count: number;
  status: string;
  ends_at: string;
  created_at: string;
}

export interface Order {
  id: string;
  buyer_id: string;
  buyer_name?: string;
  seller_id: string;
  seller_name?: string;
  total: number;
  currency: string;
  status: string;
  created_at: string;
}

export interface Dispute {
  id: string;
  order_id: string;
  opener_id: string;
  opener_name?: string;
  reason: string;
  status: string;
  created_at: string;
}

export interface SupportTicket {
  id: string;
  user_id: string;
  user_name?: string;
  assigned_to?: string;
  subject: string;
  status: "open" | "in_progress" | "waiting" | "resolved" | "closed";
  priority: "low" | "medium" | "high" | "urgent";
  category?: string;
  created_at: string;
  messages?: TicketMessage[];
}

export interface TicketMessage {
  id: string;
  ticket_id: string;
  sender_id: string;
  sender_name?: string;
  body: string;
  is_admin: boolean;
  created_at: string;
}

export interface Category {
  id: string;
  parent_id?: string;
  name_en: string;
  name_ar?: string;
  slug: string;
  icon?: string;
  sort_order: number;
  is_active: boolean;
  children?: Category[];
}

export interface PaginatedResponse<T> {
  data: T[];
  meta: {
    total: number;
    page: number;
    per_page: number;
    pages: number;
  };
}

export interface DashboardStats {
  total_users: number;
  total_listings: number;
  total_orders: number;
  total_revenue: number;
  pending_listings: number;
  active_auctions: number;
  open_disputes: number;
  open_tickets: number;
}

export interface TrustFlag {
  id: string;
  target_type: string;
  target_id: string;
  flag_type: string;
  severity: "low" | "medium" | "high" | "critical";
  source: string;
  status: "open" | "investigating" | "resolved" | "false_positive";
  notes?: string;
  risk_score?: number;
  created_at: string;
  resolved_at?: string;
  resolved_by?: string;
}

export interface TrustStats {
  open_flags: number;
  critical_flags: number;
  auto_resolved_today: number;
  manual_review_needed: number;
  banned_today: number;
  fraud_prevented_usd: number;
}

export interface Seller {
  id: string;
  username: string;
  gmv: number;
  avg_rating: number;
  dispute_rate: number;
  refund_rate: number;
  flag_count: number;
  total_sales: number;
  joined: string;
  status: "active" | "suspended" | "under_review";
}

export interface KYCSubmission {
  id: string;
  user_id: string;
  user_name: string;
  user_email: string;
  kyc_status: "none" | "pending" | "approved" | "rejected";
  phone_verified: boolean;
  id_document_url?: string;
  submitted_at: string;
  reviewed_by?: string;
  rejection_reason?: string;
}

export interface OpsHealth {
  api_latency_p95_ms: number;
  db_connections: number;
  db_max_connections: number;
  redis_memory_mb: number;
  redis_max_memory_mb: number;
  active_auctions: number;
  active_websocket_connections: number;
  job_queue_depth: number;
  error_rate_1h: number;
  last_payment_at: string;
  uptime_seconds: number;
}

export interface ComplianceAudit {
  id: string;
  admin_user_id: string;
  admin_name?: string;
  action: string;
  target_type: string;
  target_id: string;
  old_value?: string;
  new_value?: string;
  ip_address: string;
  created_at: string;
}
