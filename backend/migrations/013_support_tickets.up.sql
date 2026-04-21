-- GeoCore Next — Support Tickets

CREATE TYPE ticket_status AS ENUM ('open', 'in_progress', 'waiting', 'resolved', 'closed');
CREATE TYPE ticket_priority AS ENUM ('low', 'medium', 'high', 'urgent');

CREATE TABLE support_tickets (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_to   UUID REFERENCES users(id),
    subject       VARCHAR(255) NOT NULL,
    status        ticket_status NOT NULL DEFAULT 'open',
    priority      ticket_priority NOT NULL DEFAULT 'medium',
    category      VARCHAR(100),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at     TIMESTAMPTZ
);

CREATE TABLE ticket_messages (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_id  UUID NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    sender_id  UUID NOT NULL REFERENCES users(id),
    body       TEXT NOT NULL,
    is_admin   BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tickets_status     ON support_tickets(status);
CREATE INDEX idx_tickets_user       ON support_tickets(user_id);
CREATE INDEX idx_tickets_assigned   ON support_tickets(assigned_to);
CREATE INDEX idx_ticket_msgs_ticket ON ticket_messages(ticket_id);
