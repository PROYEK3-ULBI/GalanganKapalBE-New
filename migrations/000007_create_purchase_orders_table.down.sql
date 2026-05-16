DROP TRIGGER IF EXISTS trg_po_items_updated_at ON purchase_order_items;
DROP INDEX IF EXISTS idx_po_items_material;
DROP INDEX IF EXISTS idx_po_items_po;
DROP TABLE IF EXISTS purchase_order_items;

DROP TRIGGER IF EXISTS trg_po_updated_at ON purchase_orders;
DROP INDEX IF EXISTS idx_po_date;
DROP INDEX IF EXISTS idx_po_status;
DROP INDEX IF EXISTS idx_po_vendor;
DROP TABLE IF EXISTS purchase_orders;
