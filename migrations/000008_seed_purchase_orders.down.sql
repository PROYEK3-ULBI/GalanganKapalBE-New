-- Items get cascade-deleted when their parent PO is removed.
DELETE FROM purchase_orders WHERE po_number IN ('PO-2026-0112', 'PO-2026-0108', 'PO-2026-0105');
