-- 026: Admin Dashboard Upgrade — missing tables for Sections 2,3,6,7,8
-- All CREATE TABLE IF NOT EXISTS / ALTER TABLE ADD COLUMN IF NOT EXISTS

-- ═══════════════════════════════════════════════════════════════════════
-- SECTION 2: User Groups
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS user_groups (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) UNIQUE,
    description TEXT,
    price_plan_id INTEGER,
    permissions JSONB DEFAULT '{}',
    max_active_listings INTEGER DEFAULT 10,
    can_place_auctions BOOLEAN DEFAULT TRUE,
    requires_approval BOOLEAN DEFAULT FALSE,
    is_default BOOLEAN DEFAULT FALSE,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO user_groups (name, slug, description, is_default, sort_order) VALUES
  ('Free', 'free', 'Default free tier', TRUE, 0),
  ('Premium', 'premium', 'Premium sellers with extended limits', FALSE, 1),
  ('Business', 'business', 'Business accounts', FALSE, 2)
ON CONFLICT (slug) DO NOTHING;

-- Add group_id to users if not exists
ALTER TABLE users ADD COLUMN IF NOT EXISTS group_id INTEGER;

-- ═══════════════════════════════════════════════════════════════════════
-- SECTION 2: Custom User Fields
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS user_custom_fields (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    label VARCHAR(100) NOT NULL,
    label_en VARCHAR(100),
    field_type VARCHAR(20) NOT NULL CHECK (
        field_type IN ('text','number','select','boolean','date','url')
    ),
    options JSONB DEFAULT '[]',
    is_required BOOLEAN DEFAULT FALSE,
    placeholder VARCHAR(200),
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ═══════════════════════════════════════════════════════════════════════
-- SECTION 3: Listing Extras
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS listing_extras (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    description TEXT,
    type VARCHAR(20) CHECK (type IN ('featured','bold','highlight','gallery','video')),
    price DECIMAL(10,2) DEFAULT 0,
    duration_days INTEGER,
    is_active BOOLEAN DEFAULT TRUE
);

INSERT INTO listing_extras (name, type, price, duration_days, is_active) VALUES
  ('Featured Listing', 'featured', 5.00, 7, TRUE),
  ('Bold Title', 'bold', 2.00, 7, TRUE),
  ('Highlight Background', 'highlight', 3.00, 7, TRUE),
  ('Extra Gallery Slots', 'gallery', 4.00, 30, TRUE),
  ('Video Upload', 'video', 6.00, 30, TRUE)
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS listing_extra_purchases (
    id SERIAL PRIMARY KEY,
    listing_id UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    extra_id INTEGER NOT NULL REFERENCES listing_extras(id),
    purchased_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

-- ═══════════════════════════════════════════════════════════════════════
-- SECTION 6: Payment Gateways
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS payment_gateways (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    slug VARCHAR(50) UNIQUE,
    display_name VARCHAR(100),
    is_active BOOLEAN DEFAULT FALSE,
    is_sandbox BOOLEAN DEFAULT TRUE,
    config JSONB DEFAULT '{}',
    supported_currencies JSONB DEFAULT '["EGP","USD"]',
    fee_percent DECIMAL(5,2) DEFAULT 0,
    fee_fixed DECIMAL(10,2) DEFAULT 0,
    sort_order INTEGER DEFAULT 0
);

INSERT INTO payment_gateways (name, slug, display_name, is_active) VALUES
  ('Stripe', 'stripe', 'Stripe', FALSE),
  ('PayPal', 'paypal', 'PayPal', FALSE),
  ('Paymob', 'paymob', 'Paymob (مصر)', FALSE),
  ('Fawry', 'fawry', 'فوري', FALSE),
  ('Vodafone Cash', 'vodafone_cash', 'فودافون كاش', FALSE)
ON CONFLICT (slug) DO NOTHING;

-- SECTION 6: Invoices
CREATE TABLE IF NOT EXISTS invoices (
    id SERIAL PRIMARY KEY,
    invoice_number VARCHAR(20) UNIQUE,
    user_id UUID REFERENCES users(id),
    items JSONB NOT NULL DEFAULT '[]',
    subtotal DECIMAL(10,2) DEFAULT 0,
    discount DECIMAL(10,2) DEFAULT 0,
    tax DECIMAL(10,2) DEFAULT 0,
    total DECIMAL(10,2) DEFAULT 0,
    status VARCHAR(20) DEFAULT 'pending'
        CHECK (status IN ('pending','paid','refunded','cancelled')),
    gateway_id INTEGER,
    gateway_reference VARCHAR(200),
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    paid_at TIMESTAMPTZ
);

-- SECTION 6: Discount Codes
CREATE TABLE IF NOT EXISTS discount_codes (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    discount_type VARCHAR(20) CHECK (discount_type IN ('percent','fixed')),
    discount_value DECIMAL(10,2),
    applies_to VARCHAR(20) DEFAULT 'all'
        CHECK (applies_to IN ('all','classifieds','auctions','subscriptions')),
    min_order_amount DECIMAL(10,2) DEFAULT 0,
    max_uses INTEGER,
    uses_per_user INTEGER DEFAULT 1,
    current_uses INTEGER DEFAULT 0,
    user_group_id INTEGER,
    valid_from TIMESTAMPTZ,
    valid_until TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Add extra columns to plans if needed
ALTER TABLE plans ADD COLUMN IF NOT EXISTS billing_period VARCHAR(20) DEFAULT 'one_time';
ALTER TABLE plans ADD COLUMN IF NOT EXISTS max_images_per_listing INTEGER DEFAULT 5;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS listing_duration_days INTEGER DEFAULT 30;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS allow_classifieds BOOLEAN DEFAULT TRUE;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS classified_cost DECIMAL(10,2) DEFAULT 0;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS allow_auctions BOOLEAN DEFAULT TRUE;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS allow_dutch_auctions BOOLEAN DEFAULT FALSE;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS allow_reverse_auctions BOOLEAN DEFAULT FALSE;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS allow_buy_now BOOLEAN DEFAULT TRUE;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS allow_buy_now_only BOOLEAN DEFAULT FALSE;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS auction_cost DECIMAL(10,2) DEFAULT 0;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS commission_percent DECIMAL(5,2) DEFAULT 0;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS allow_featured BOOLEAN DEFAULT FALSE;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS featured_cost DECIMAL(10,2) DEFAULT 0;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS allow_storefront BOOLEAN DEFAULT FALSE;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS require_approval BOOLEAN DEFAULT FALSE;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS is_default BOOLEAN DEFAULT FALSE;

-- ═══════════════════════════════════════════════════════════════════════
-- SECTION 7: Email Templates
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS email_templates (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(100) UNIQUE,
    name VARCHAR(200),
    subject VARCHAR(300),
    body_html TEXT,
    body_text TEXT,
    variables JSONB DEFAULT '[]',
    is_active BOOLEAN DEFAULT TRUE,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO email_templates (slug, name, subject, body_html, variables) VALUES
  ('user_registered', 'تسجيل مستخدم جديد', 'مرحباً بك في {{site_name}}', '<p>مرحباً {{user_name}}</p>', '["site_name","user_name","activation_link"]'),
  ('listing_approved', 'موافقة على الإعلان', 'تم الموافقة على إعلانك', '<p>تم الموافقة على إعلانك {{listing_title}}</p>', '["listing_title","listing_url"]'),
  ('listing_rejected', 'رفض الإعلان', 'تم رفض إعلانك', '<p>تم رفض إعلانك بسبب: {{reason}}</p>', '["listing_title","reason"]'),
  ('auction_won', 'الفوز بمزاد', 'مبروك! فزت بالمزاد', '<p>فزت بالمزاد {{auction_title}} بسعر {{winning_bid}}</p>', '["auction_title","winning_bid","seller_contact"]'),
  ('bid_outbid', 'تخطي العرض', 'تم تخطي عرضك', '<p>تم تخطي عرضك في مزاد {{auction_title}}</p>', '["auction_title","current_bid","auction_url"]'),
  ('password_reset', 'إعادة تعيين كلمة المرور', 'إعادة تعيين كلمة المرور', '<p>اضغط هنا لإعادة تعيين كلمة المرور: {{reset_link}}</p>', '["reset_link","expiry_time"]'),
  ('kyc_approved', 'تم التحقق من الهوية', 'تم التحقق من هويتك', '<p>تم التحقق من هويتك بنجاح</p>', '["user_name"]'),
  ('kyc_rejected', 'رفض التحقق من الهوية', 'تم رفض التحقق من هويتك', '<p>تم رفض التحقق بسبب: {{reason}}</p>', '["reason"]'),
  ('payment_received', 'تأكيد الدفع', 'تم استلام دفعتك', '<p>تم استلام دفعتك بقيمة {{amount}}</p>', '["amount","invoice_number"]')
ON CONFLICT (slug) DO NOTHING;

-- SECTION 7: Static Pages
CREATE TABLE IF NOT EXISTS static_pages (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200),
    slug VARCHAR(200) UNIQUE,
    content TEXT,
    meta_title VARCHAR(200),
    meta_description TEXT,
    is_published BOOLEAN DEFAULT FALSE,
    show_in_footer BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO static_pages (title, slug, is_published) VALUES
  ('عن الموقع', 'about', FALSE),
  ('اتصل بنا', 'contact', FALSE),
  ('سياسة الخصوصية', 'privacy-policy', FALSE),
  ('الشروط والأحكام', 'terms-of-service', FALSE),
  ('الأسئلة الشائعة', 'faq', FALSE)
ON CONFLICT (slug) DO NOTHING;

-- SECTION 7: Announcements
CREATE TABLE IF NOT EXISTS announcements (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200),
    content TEXT,
    type VARCHAR(20) DEFAULT 'info'
        CHECK (type IN ('info','warning','success','error')),
    display_location VARCHAR(20) DEFAULT 'homepage'
        CHECK (display_location IN ('homepage','all','listing_form','auction_form')),
    target_group_id INTEGER,
    starts_at TIMESTAMPTZ DEFAULT NOW(),
    ends_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ═══════════════════════════════════════════════════════════════════════
-- SECTION 8: Geography
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS geo_regions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    name_ar VARCHAR(100),
    code VARCHAR(10),
    type VARCHAR(20) CHECK (type IN ('country','state','city')),
    parent_id INTEGER REFERENCES geo_regions(id),
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    is_active BOOLEAN DEFAULT TRUE,
    sort_order INTEGER DEFAULT 0
);

INSERT INTO geo_regions (name, name_ar, code, type) VALUES
  ('Egypt', 'مصر', 'EG', 'country'),
  ('Saudi Arabia', 'السعودية', 'SA', 'country'),
  ('UAE', 'الإمارات', 'AE', 'country'),
  ('Kuwait', 'الكويت', 'KW', 'country'),
  ('Jordan', 'الأردن', 'JO', 'country')
ON CONFLICT DO NOTHING;

-- Add settings column to categories
ALTER TABLE categories ADD COLUMN IF NOT EXISTS settings JSONB DEFAULT '{}';
