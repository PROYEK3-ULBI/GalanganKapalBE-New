-- Transactions ledger: single table tracking all stock movements.
-- Types:
--   receipt: vendor → warehouse (stock +qty)
--   issue:   warehouse → project (stock -qty)
--   scrap:   project → disposal  (stock -qty)
--   return:  project → warehouse (stock +qty, reusable material)
--
-- Each row is immutable once created; corrections must be done via reversing transactions.
-- Stock is updated atomically with the row insert via service-layer transactions.

CREATE TABLE IF NOT EXISTS transactions (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_no     VARCHAR(50)  NOT NULL UNIQUE,
    type               VARCHAR(20)  NOT NULL CHECK (type IN ('receipt', 'issue', 'scrap', 'return')),

    material_id        UUID         NOT NULL REFERENCES materials(id) ON DELETE RESTRICT,
    qty                INTEGER      NOT NULL CHECK (qty > 0),

    project_id         UUID         REFERENCES projects(id) ON DELETE RESTRICT,
    vendor_id          UUID         REFERENCES vendors(id)  ON DELETE RESTRICT,
    purchase_order_id  UUID         REFERENCES purchase_orders(id)      ON DELETE SET NULL,
    po_item_id         UUID         REFERENCES purchase_order_items(id) ON DELETE SET NULL,
    user_id            UUID         REFERENCES users(id) ON DELETE SET NULL,

    heat_number        VARCHAR(50),
    notes              TEXT,
    transaction_date   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- Domain-level integrity rules:
    -- receipt requires vendor_id (or po linkage).
    CONSTRAINT receipt_has_source CHECK (
        type <> 'receipt' OR vendor_id IS NOT NULL OR purchase_order_id IS NOT NULL
    ),
    -- issue requires project_id (must consume into a hull/drydock).
    CONSTRAINT issue_has_project CHECK (
        type <> 'issue' OR project_id IS NOT NULL
    )
);

CREATE INDEX IF NOT EXISTS idx_transactions_no       ON transactions(transaction_no);
CREATE INDEX IF NOT EXISTS idx_transactions_type     ON transactions(type);
CREATE INDEX IF NOT EXISTS idx_transactions_material ON transactions(material_id);
CREATE INDEX IF NOT EXISTS idx_transactions_project  ON transactions(project_id);
CREATE INDEX IF NOT EXISTS idx_transactions_po       ON transactions(purchase_order_id);
CREATE INDEX IF NOT EXISTS idx_transactions_user     ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_date     ON transactions(transaction_date DESC);

DROP TRIGGER IF EXISTS trg_transactions_updated_at ON transactions;
CREATE TRIGGER trg_transactions_updated_at
BEFORE UPDATE ON transactions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
