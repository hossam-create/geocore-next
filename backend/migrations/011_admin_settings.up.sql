-- GeoCore Next — Admin Settings Engine

CREATE TABLE admin_settings (
    key         VARCHAR(255) PRIMARY KEY,
    value       TEXT NOT NULL,
    type        VARCHAR(50) NOT NULL,
    category    VARCHAR(100) NOT NULL,
    label       VARCHAR(255) NOT NULL,
    description TEXT,
    options     JSONB,
    is_public   BOOLEAN DEFAULT FALSE,
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_by  UUID REFERENCES users(id)
);

CREATE INDEX idx_admin_settings_category  ON admin_settings(category);
CREATE INDEX idx_admin_settings_public    ON admin_settings(is_public) WHERE is_public = TRUE;
