-- Seed sample purchase orders matching frontend mockData.
-- Uses CTEs to look up vendor and material UUIDs by their natural keys.

WITH
v_krakatau AS (SELECT id FROM vendors WHERE name = 'PT Krakatau Steel'),
v_baja     AS (SELECT id FROM vendors WHERE name = 'PT Baja Utama'),
v_gas      AS (SELECT id FROM vendors WHERE name = 'PT Gas Industri'),
po_inserts AS (
    INSERT INTO purchase_orders (po_number, vendor_id, order_date, status)
    VALUES
        ('PO-2026-0112', (SELECT id FROM v_krakatau), '2026-04-25', 'Partially Received'),
        ('PO-2026-0108', (SELECT id FROM v_baja),     '2026-04-20', 'Completed'),
        ('PO-2026-0105', (SELECT id FROM v_gas),      '2026-04-18', 'Partially Received')
    ON CONFLICT (po_number) DO NOTHING
    RETURNING id, po_number
)
INSERT INTO purchase_order_items (purchase_order_id, material_id, ordered_qty, received_qty, unit_price)
SELECT p.id, m.id, items.ordered, items.received, items.unit_price
FROM po_inserts p
JOIN (
    VALUES
        ('PO-2026-0112', 'PLT-AH36-1020', 100, 50, 2850000),
        ('PO-2026-0112', 'PLT-AH36-1220', 60,  0,  3420000),
        ('PO-2026-0108', 'BLT-HEX-M20',   500, 500, 8500),
        ('PO-2026-0105', 'GAS-OXY-50L',   10,  10, 280000),
        ('PO-2026-0105', 'GAS-ACT-50L',   15,  0,  450000)
) AS items(po_number, sku, ordered, received, unit_price) ON items.po_number = p.po_number
JOIN materials m ON m.sku = items.sku;
