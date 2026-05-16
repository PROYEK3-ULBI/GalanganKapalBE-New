-- Activity log: append-only audit trail of significant user actions.
-- Inserted via fire-and-forget from each module that performs mutations.

CREATE TABLE IF NOT EXISTS activity_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action        VARCHAR(100) NOT NULL,
    detail        TEXT,
    type          VARCHAR(20)  NOT NULL DEFAULT 'info'
                  CHECK (type IN ('info', 'success', 'warning', 'danger')),
    user_id       UUID         REFERENCES users(id) ON DELETE SET NULL,
    -- Optional resource link for click-through navigation.
    resource_type VARCHAR(50),
    resource_id   VARCHAR(100),
    -- Optional metadata for grouping similar entries.
    category      VARCHAR(50),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_activity_user      ON activity_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_activity_date      ON activity_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_activity_type      ON activity_logs(type);
CREATE INDEX IF NOT EXISTS idx_activity_category  ON activity_logs(category);
CREATE INDEX IF NOT EXISTS idx_activity_resource  ON activity_logs(resource_type, resource_id);
