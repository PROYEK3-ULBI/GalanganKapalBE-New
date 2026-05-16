-- Vendors / Suppliers table.

CREATE TABLE IF NOT EXISTS vendors (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL UNIQUE,
    contact     VARCHAR(255),
    phone       VARCHAR(50),
    email       VARCHAR(255),
    address     TEXT,
    status      VARCHAR(20)  NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_vendors_name   ON vendors(name);
CREATE INDEX IF NOT EXISTS idx_vendors_status ON vendors(status);

DROP TRIGGER IF EXISTS trg_vendors_updated_at ON vendors;
CREATE TRIGGER trg_vendors_updated_at
BEFORE UPDATE ON vendors
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
