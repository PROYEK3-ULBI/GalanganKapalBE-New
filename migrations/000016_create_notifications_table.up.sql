-- Per-user notifications. Inserted by other modules via the notification service.

CREATE TABLE IF NOT EXISTS notifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       VARCHAR(255) NOT NULL,
    message     TEXT         NOT NULL,
    type        VARCHAR(20)  NOT NULL DEFAULT 'info'
                CHECK (type IN ('info', 'success', 'warning', 'danger')),
    -- Optional link to a related resource for click-through.
    link        VARCHAR(255),
    -- Optional metadata (e.g. for grouping similar notifications).
    category    VARCHAR(50),
    read        BOOLEAN      NOT NULL DEFAULT FALSE,
    read_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notif_user_unread ON notifications(user_id, read, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notif_user_date   ON notifications(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notif_category    ON notifications(category);
