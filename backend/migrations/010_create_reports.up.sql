-- GeoCore Next — User / Listing Reports Queue

CREATE TYPE report_target_type AS ENUM ('listing', 'user');
CREATE TYPE report_status AS ENUM ('pending', 'reviewed', 'dismissed', 'actioned');

CREATE TABLE reports (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reporter_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type     report_target_type NOT NULL,
    target_id       UUID NOT NULL,
    reason          VARCHAR(100) NOT NULL,
    description     TEXT,
    status          report_status NOT NULL DEFAULT 'pending',
    reviewed_by     UUID REFERENCES users(id),
    reviewed_at     TIMESTAMPTZ,
    admin_note      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT reports_unique_report UNIQUE (reporter_id, target_type, target_id)
);

CREATE INDEX idx_reports_status      ON reports(status);
CREATE INDEX idx_reports_target      ON reports(target_type, target_id);
CREATE INDEX idx_reports_reporter    ON reports(reporter_id);
CREATE INDEX idx_reports_created_at  ON reports(created_at DESC);
