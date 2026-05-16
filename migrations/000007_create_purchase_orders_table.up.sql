-- Purchase Orders header + line items.

CREATE TABLE IF NOT EXISTS purchase_orders (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    po_number     VARCHAR(50)   NOT NULL UNIQUE,
    vendor_id     UUID          NOT NULL REFERENCES vendors(id) ON DELETE RESTRICT,
    order_date    DATE          NOT NULL DEFAULT CURRENT_DATE,
    status        VARCHAR(30)   NOT NULL DEFAULT 'Draft'
                  CHECK (status IN ('Draft', 'Pending', 'Partially Received', 'Completed', 'Cancelled')),
    notes         TEXT,
    created_by    UUID          REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_po_vendor ON purchase_orders(vendor_id);
CREATE INDEX IF NOT EXISTS idx_po_status ON purchase_orders(status);
CREATE INDEX IF NOT EXISTS idx_po_date   ON purchase_orders(order_date DESC);

DROP TRIGGER IF EXISTS trg_po_updated_at ON purchase_orders;
CREATE TRIGGER trg_po_updated_at
BEFORE UPDATE ON purchase_orders
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS purchase_order_items (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_order_id  UUID           NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    material_id        UUID           NOT NULL REFERENCES materials(id) ON DELETE RESTRICT,
    ordered_qty        INTEGER        NOT NULL CHECK (ordered_qty > 0),
    received_qty       INTEGER        NOT NULL DEFAULT 0 CHECK (received_qty >= 0),
    unit_price         NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (unit_price >= 0),
    notes              TEXT,
    created_at         TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    CONSTRAINT received_le_ordered CHECK (received_qty <= ordered_qty)
);

CREATE INDEX IF NOT EXISTS idx_po_items_po       ON purchase_order_items(purchase_order_id);
CREATE INDEX IF NOT EXISTS idx_po_items_material ON purchase_order_items(material_id);

DROP TRIGGER IF EXISTS trg_po_items_updated_at ON purchase_order_items;
CREATE TRIGGER trg_po_items_updated_at
BEFORE UPDATE ON purchase_order_items
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
