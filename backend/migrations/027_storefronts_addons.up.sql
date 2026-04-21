-- 027: Storefronts admin column + Addons table

-- Add is_featured to storefronts if not exists
ALTER TABLE storefronts ADD COLUMN IF NOT EXISTS is_featured BOOLEAN DEFAULT FALSE;

-- Addons table (skeleton for future plugin marketplace)
CREATE TABLE IF NOT EXISTS addons (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) UNIQUE,
    version VARCHAR(20) DEFAULT '1.0.0',
    description TEXT,
    is_active BOOLEAN DEFAULT FALSE,
    is_installed BOOLEAN DEFAULT FALSE,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO addons (name, slug, description, is_installed, is_active) VALUES
  ('Loyalty Points', 'loyalty-points', 'Reward users with points for purchases and referrals', TRUE, TRUE),
  ('Multi-Currency', 'multi-currency', 'Support for multiple currencies with real-time exchange rates', TRUE, FALSE)
ON CONFLICT (slug) DO NOTHING;
