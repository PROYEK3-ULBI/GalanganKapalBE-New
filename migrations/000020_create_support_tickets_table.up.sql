-- Support tickets submitted by any authenticated user.

CREATE TABLE IF NOT EXISTS support_tickets (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_no    VARCHAR(50)  NOT NULL UNIQUE,
    user_id      UUID         NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    subject      VARCHAR(255) NOT NULL,
    message      TEXT         NOT NULL,
    status       VARCHAR(20)  NOT NULL DEFAULT 'open'
                 CHECK (status IN ('open', 'in_progress', 'resolved', 'closed')),
    priority     VARCHAR(10)  NOT NULL DEFAULT 'medium'
                 CHECK (priority IN ('low', 'medium', 'high')),
    -- Admin response (optional, populated when ticket is resolved).
    response     TEXT,
    handled_by   UUID         REFERENCES users(id) ON DELETE SET NULL,
    resolved_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tickets_user     ON support_tickets(user_id);
CREATE INDEX IF NOT EXISTS idx_tickets_status   ON support_tickets(status);
CREATE INDEX IF NOT EXISTS idx_tickets_priority ON support_tickets(priority);
CREATE INDEX IF NOT EXISTS idx_tickets_date     ON support_tickets(created_at DESC);

DROP TRIGGER IF EXISTS trg_tickets_updated_at ON support_tickets;
CREATE TRIGGER trg_tickets_updated_at
BEFORE UPDATE ON support_tickets
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
