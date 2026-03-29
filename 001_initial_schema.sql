-- GeoCore Next - Complete Database Schema

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- USERS
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email           VARCHAR(255) UNIQUE NOT NULL,
    phone           VARCHAR(20),
    password_hash   VARCHAR(255) NOT NULL,
    name            VARCHAR(100) NOT NULL,
    avatar_url      TEXT,
    role            VARCHAR(20) NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin', 'moderator')),
    is_verified     BOOLEAN NOT NULL DEFAULT FALSE,
    country         VARCHAR(2),
    bio             TEXT,
    rating          DECIMAL(3,2) DEFAULT 0.00,
    total_reviews   INT DEFAULT 0,
    total_listings  INT DEFAULT 0,
    is_blocked      BOOLEAN NOT NULL DEFAULT FALSE,
    last_seen_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_users_email ON users(email);

-- CATEGORIES
CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    parent_id   UUID REFERENCES categories(id) ON DELETE SET NULL,
    name_en     VARCHAR(100) NOT NULL,
    name_ar     VARCHAR(100),
    slug        VARCHAR(100) UNIQUE NOT NULL,
    icon        VARCHAR(50),
    sort_order  INT DEFAULT 0,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO categories (name_en, name_ar, slug, icon) VALUES
    ('Vehicles', 'السيارات', 'vehicles', 'car'),
    ('Real Estate', 'العقارات', 'real-estate', 'home'),
    ('Electronics', 'الإلكترونيات', 'electronics', 'smartphone'),
    ('Furniture', 'الأثاث', 'furniture', 'sofa'),
    ('Clothing', 'الملابس', 'clothing', 'shirt'),
    ('Jobs', 'الوظائف', 'jobs', 'briefcase'),
    ('Services', 'الخدمات', 'services', 'tool'),
    ('Animals & Pets', 'الحيوانات', 'animals-pets', 'paw'),
    ('Sports & Hobbies', 'الرياضة', 'sports-hobbies', 'ball'),
    ('Kids & Baby', 'الأطفال', 'kids-baby', 'toy');

-- LISTINGS
CREATE TABLE listings (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id     UUID NOT NULL REFERENCES categories(id),
    title           VARCHAR(200) NOT NULL,
    description     TEXT NOT NULL,
    price           DECIMAL(15,2),
    currency        VARCHAR(3) NOT NULL DEFAULT 'USD',
    type            VARCHAR(20) NOT NULL CHECK (type IN ('classifieds', 'auction', 'buy_now', 'wanted')),
    status          VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'sold', 'expired', 'deleted', 'pending', 'rejected')),
    country         VARCHAR(2) NOT NULL,
    city            VARCHAR(100),
    location        GEOGRAPHY(POINT, 4326),
    images          TEXT[] DEFAULT '{}',
    attributes      JSONB DEFAULT '{}',
    view_count      INT NOT NULL DEFAULT 0,
    is_featured     BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at      TIMESTAMPTZ,
    sold_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_listings_user ON listings(user_id);
CREATE INDEX idx_listings_category ON listings(category_id);
CREATE INDEX idx_listings_status ON listings(status);
CREATE INDEX idx_listings_country ON listings(country);
CREATE INDEX idx_listings_created ON listings(created_at DESC);
CREATE INDEX idx_listings_location ON listings USING GIST(location);
CREATE INDEX idx_listings_fts ON listings USING GIN(to_tsvector('english', title || ' ' || description));
CREATE INDEX idx_listings_title_trgm ON listings USING GIN(title gin_trgm_ops);

-- AUCTIONS
CREATE TABLE auctions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id      UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    start_price     DECIMAL(15,2) NOT NULL,
    current_bid     DECIMAL(15,2) NOT NULL,
    reserve_price   DECIMAL(15,2),
    bid_increment   DECIMAL(15,2) NOT NULL DEFAULT 1.00,
    currency        VARCHAR(3) NOT NULL DEFAULT 'USD',
    status          VARCHAR(20) NOT NULL DEFAULT 'upcoming' CHECK (status IN ('upcoming', 'active', 'ended', 'cancelled')),
    starts_at       TIMESTAMPTZ NOT NULL,
    ends_at         TIMESTAMPTZ NOT NULL,
    winner_id       UUID REFERENCES users(id),
    total_bids      INT NOT NULL DEFAULT 0,
    extension_mins  INT NOT NULL DEFAULT 5,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_auctions_status ON auctions(status);
CREATE INDEX idx_auctions_ends_at ON auctions(ends_at) WHERE status = 'active';

-- BIDS
CREATE TABLE bids (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    auction_id  UUID NOT NULL REFERENCES auctions(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount      DECIMAL(15,2) NOT NULL,
    is_auto_bid BOOLEAN NOT NULL DEFAULT FALSE,
    max_amount  DECIMAL(15,2),
    placed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_bids_auction ON bids(auction_id);
CREATE INDEX idx_bids_amount ON bids(auction_id, amount DESC);

-- CONVERSATIONS
CREATE TABLE conversations (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id      UUID REFERENCES listings(id) ON DELETE SET NULL,
    participant1_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    participant2_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_message    TEXT,
    last_message_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_conversations_p1 ON conversations(participant1_id);
CREATE INDEX idx_conversations_p2 ON conversations(participant2_id);

-- MESSAGES
CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content         TEXT NOT NULL,
    type            VARCHAR(20) NOT NULL DEFAULT 'text' CHECK (type IN ('text', 'image', 'offer', 'system')),
    is_read         BOOLEAN NOT NULL DEFAULT FALSE,
    read_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_messages_conv ON messages(conversation_id);
CREATE INDEX idx_messages_created ON messages(conversation_id, created_at DESC);

-- PAYMENTS
CREATE TABLE payments (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID NOT NULL REFERENCES users(id),
    listing_id          UUID REFERENCES listings(id),
    auction_id          UUID REFERENCES auctions(id),
    amount              DECIMAL(15,2) NOT NULL,
    currency            VARCHAR(3) NOT NULL,
    type                VARCHAR(30) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'failed', 'refunded')),
    provider            VARCHAR(20) NOT NULL DEFAULT 'stripe',
    provider_payment_id VARCHAR(255),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- FAVORITES
CREATE TABLE favorites (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    listing_id  UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, listing_id)
);

-- REVIEWS
CREATE TABLE reviews (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reviewer_id UUID NOT NULL REFERENCES users(id),
    reviewed_id UUID NOT NULL REFERENCES users(id),
    listing_id  UUID REFERENCES listings(id),
    rating      SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment     TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER listings_updated_at BEFORE UPDATE ON listings FOR EACH ROW EXECUTE FUNCTION update_updated_at();
