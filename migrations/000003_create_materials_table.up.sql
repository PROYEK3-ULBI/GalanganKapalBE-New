-- Materials table: catalog of all stock-keeping units (SKUs).
-- The "status" field is computed at query time from stock vs min_stock,
-- not stored, to avoid data drift.

CREATE TABLE IF NOT EXISTS materials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku             VARCHAR(50)    NOT NULL UNIQUE,
    name            VARCHAR(255)   NOT NULL,
    category        VARCHAR(100)   NOT NULL,
    unit            VARCHAR(20)    NOT NULL,
    stock           INTEGER        NOT NULL DEFAULT 0 CHECK (stock >= 0),
    min_stock       INTEGER        NOT NULL DEFAULT 0 CHECK (min_stock >= 0),
    reorder_point   INTEGER        NOT NULL DEFAULT 0 CHECK (reorder_point >= 0),
    price           NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (price >= 0),
    hazmat          BOOLEAN        NOT NULL DEFAULT FALSE,
    heat_number     VARCHAR(50),
    location        VARCHAR(50),
    specifications  TEXT,
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_materials_sku       ON materials(sku);
CREATE INDEX IF NOT EXISTS idx_materials_category  ON materials(category);
CREATE INDEX IF NOT EXISTS idx_materials_hazmat    ON materials(hazmat) WHERE hazmat = TRUE;
CREATE INDEX IF NOT EXISTS idx_materials_low_stock ON materials(stock, min_stock);

-- Reuse the set_updated_at trigger from migration 000001.
DROP TRIGGER IF EXISTS trg_materials_updated_at ON materials;
CREATE TRIGGER trg_materials_updated_at
BEFORE UPDATE ON materials
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
