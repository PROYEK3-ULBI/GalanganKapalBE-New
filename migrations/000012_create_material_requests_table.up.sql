-- Material requests: Staff submits, Supervisor approves/rejects.

CREATE TABLE IF NOT EXISTS material_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_no      VARCHAR(50)  NOT NULL UNIQUE,
    type            VARCHAR(50)  NOT NULL DEFAULT 'Material Request'
                    CHECK (type IN ('Material Request', 'Tool Request', 'Purchase Request')),
    project_id      UUID         REFERENCES projects(id) ON DELETE SET NULL,
    priority        VARCHAR(10)  NOT NULL DEFAULT 'medium'
                    CHECK (priority IN ('low', 'medium', 'high')),
    reason          TEXT         NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'approved', 'rejected')),

    requester_id    UUID         NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    approver_id     UUID         REFERENCES users(id) ON DELETE SET NULL,
    approval_notes  TEXT,
    approved_at     TIMESTAMPTZ,

    request_date    DATE         NOT NULL DEFAULT CURRENT_DATE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mr_status     ON material_requests(status);
CREATE INDEX IF NOT EXISTS idx_mr_requester  ON material_requests(requester_id);
CREATE INDEX IF NOT EXISTS idx_mr_project    ON material_requests(project_id);
CREATE INDEX IF NOT EXISTS idx_mr_date       ON material_requests(request_date DESC);

DROP TRIGGER IF EXISTS trg_mr_updated_at ON material_requests;
CREATE TRIGGER trg_mr_updated_at
BEFORE UPDATE ON material_requests
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS material_request_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id   UUID         NOT NULL REFERENCES material_requests(id) ON DELETE CASCADE,
    material_id  UUID         NOT NULL REFERENCES materials(id) ON DELETE RESTRICT,
    qty          INTEGER      NOT NULL CHECK (qty > 0),
    notes        TEXT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mr_items_request  ON material_request_items(request_id);
CREATE INDEX IF NOT EXISTS idx_mr_items_material ON material_request_items(material_id);
