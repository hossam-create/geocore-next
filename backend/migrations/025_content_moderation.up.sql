-- Content moderation: restricted keywords + moderation log

CREATE TABLE IF NOT EXISTS restricted_keywords (
    id          SERIAL PRIMARY KEY,
    keyword     VARCHAR(100) NOT NULL,
    severity    VARCHAR(20) NOT NULL CHECK (severity IN ('block','flag')),
    message_en  VARCHAR(300),
    message_ar  VARCHAR(300),
    category    VARCHAR(50),
    is_active   BOOLEAN DEFAULT TRUE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_rk_active ON restricted_keywords(is_active) WHERE is_active = TRUE;

CREATE TABLE IF NOT EXISTS moderation_logs (
    id            BIGSERIAL PRIMARY KEY,
    target_type   VARCHAR(20) NOT NULL,
    target_id     UUID NOT NULL,
    action        VARCHAR(20) NOT NULL,
    reason        TEXT,
    moderator_id  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ml_target ON moderation_logs(target_type, target_id);

-- Seed common restricted keywords
INSERT INTO restricted_keywords (keyword, severity, message_en, message_ar) VALUES
    ('counterfeit', 'block', 'Counterfeit items are not allowed', 'المنتجات المقلدة غير مسموحة'),
    ('replica', 'block', 'Replica items are not allowed', 'المنتجات المقلدة غير مسموحة'),
    ('fake', 'flag', 'Content flagged for review', 'المحتوى معلق للمراجعة'),
    ('stolen', 'block', 'Suspected stolen goods', 'بضائع مسروقة مشتبه بها'),
    ('drugs', 'block', 'Illegal items are not allowed', 'المواد الممنوعة غير مسموحة'),
    ('weapon', 'block', 'Weapons are not allowed', 'الأسلحة غير مسموحة'),
    ('gun', 'block', 'Firearms are not allowed', 'الأسلحة النارية غير مسموحة'),
    ('whatsapp me', 'flag', 'External contact flagged', 'محاولة تواصل خارجي'),
    ('call me now', 'flag', 'External contact flagged', 'محاولة تواصل خارجي');
