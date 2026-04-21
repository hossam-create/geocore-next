-- Plugin Marketplace

CREATE TYPE plugin_status AS ENUM ('draft', 'published', 'disabled', 'archived');

CREATE TABLE IF NOT EXISTS plugins (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    author_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            VARCHAR(200) NOT NULL,
    slug            VARCHAR(200) NOT NULL UNIQUE,
    description     TEXT,
    version         VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    category        VARCHAR(100) NOT NULL DEFAULT 'general',
    icon_url        TEXT,
    repo_url        TEXT,
    config_schema   JSONB DEFAULT '{}',
    price           NUMERIC(10,2) NOT NULL DEFAULT 0,
    currency        VARCHAR(10) NOT NULL DEFAULT 'USD',
    is_free         BOOLEAN NOT NULL DEFAULT true,
    install_count   INTEGER NOT NULL DEFAULT 0,
    avg_rating      NUMERIC(3,2) NOT NULL DEFAULT 0,
    status          plugin_status NOT NULL DEFAULT 'draft',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plugins_author   ON plugins(author_id);
CREATE INDEX idx_plugins_status   ON plugins(status);
CREATE INDEX idx_plugins_category ON plugins(category);

CREATE TABLE IF NOT EXISTS plugin_installs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    plugin_id       UUID NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    config          JSONB DEFAULT '{}',
    is_active       BOOLEAN NOT NULL DEFAULT true,
    installed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(plugin_id, user_id)
);

CREATE INDEX idx_plugin_installs_user ON plugin_installs(user_id);
