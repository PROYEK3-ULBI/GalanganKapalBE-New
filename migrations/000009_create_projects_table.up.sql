-- Projects (vessel hulls under construction or drydocks).
-- code is the natural key used in transactions (e.g. H-2026-001, DR-2026-001).

CREATE TABLE IF NOT EXISTS projects (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code            VARCHAR(30)  NOT NULL UNIQUE,
    name            VARCHAR(255) NOT NULL,
    type            VARCHAR(100) NOT NULL,
    status          VARCHAR(30)  NOT NULL DEFAULT 'Planning'
                    CHECK (status IN ('Planning', 'In Progress', 'In Drydock', 'On Hold', 'Completed', 'Cancelled')),
    completion_pct  INTEGER      NOT NULL DEFAULT 0
                    CHECK (completion_pct >= 0 AND completion_pct <= 100),
    start_date      DATE,
    target_date     DATE,
    notes           TEXT,
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_projects_code   ON projects(code);
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);

DROP TRIGGER IF EXISTS trg_projects_updated_at ON projects;
CREATE TRIGGER trg_projects_updated_at
BEFORE UPDATE ON projects
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
