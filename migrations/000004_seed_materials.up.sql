-- Seed materials matching frontend mockData.js for parity during integration.
-- Prices are in IDR (Rupiah).

INSERT INTO materials (sku, name, category, unit, stock, min_stock, reorder_point, price, hazmat, heat_number, location)
VALUES
    ('PLT-AH36-1020',  'Steel Plate AH36 10mm',          'Steel Plates',         'Sheet',     145, 50, 75,  2850000,  FALSE, 'HN-2026-0451', 'Yard-A1'),
    ('PLT-AH36-1220',  'Steel Plate AH36 12mm',          'Steel Plates',         'Sheet',      89, 40, 60,  3420000,  FALSE, 'HN-2026-0452', 'Yard-A2'),
    ('PLT-DH36-1620',  'Steel Plate DH36 16mm',          'Steel Plates',         'Sheet',      12, 20, 30,  4580000,  FALSE, 'HN-2026-0389', 'Yard-A3'),
    ('WLD-E7018-350',  'Welding Electrode E7018 3.2mm',  'Welding Consumables',  'Kg',        520, 200, 300, 45000,   FALSE, NULL,           'WH-B1'),
    ('WLD-E7018-400',  'Welding Electrode E7018 4.0mm',  'Welding Consumables',  'Kg',        180, 150, 200, 48000,   FALSE, NULL,           'WH-B1'),
    ('PNT-EPX-MAR20',  'Marine Epoxy Primer 20L',        'Paint & Coating',      'Pail',       35, 15, 25,  1250000,  TRUE,  NULL,           'WH-C1'),
    ('PNT-AFO-RED20',  'Antifouling Paint Red 20L',      'Paint & Coating',      'Pail',        0, 10, 20,  1850000,  TRUE,  NULL,           'WH-C1'),
    ('PPE-PIP-SCH40',  'Pipe Schedule 40 6inch',         'Piping',               'Length',     67, 20, 35,  890000,   FALSE, 'HN-2026-0510', 'Yard-D1'),
    ('BLT-HEX-M20',    'Hex Bolt M20x60 Grade 8.8',      'Fasteners',            'Pcs',      2400, 500, 800, 8500,    FALSE, NULL,           'WH-E1'),
    ('GAS-ACT-50L',    'Acetylene Gas 50L Cylinder',     'Gas & Chemicals',      'Cylinder',    8, 10, 15,  450000,   TRUE,  NULL,           'GAS-YARD'),
    ('GAS-OXY-50L',    'Oxygen Gas 50L Cylinder',        'Gas & Chemicals',      'Cylinder',   15, 10, 15,  280000,   TRUE,  NULL,           'GAS-YARD'),
    ('PLT-SS316-0810', 'Stainless Steel 316L 8mm',       'Steel Plates',         'Sheet',      22, 10, 15,  8750000,  FALSE, 'HN-2026-0601', 'Yard-A4'),
    ('CBL-PWR-35MM',   'Power Cable XLPE 35mm2',         'Electrical',           'Meter',     450, 200, 300, 125000,  FALSE, NULL,           'WH-F1'),
    ('INS-RCK-50MM',   'Rockwool Insulation 50mm',       'Insulation',           'Roll',       28, 15, 20,  385000,   FALSE, NULL,           'WH-G1'),
    ('VLV-GTR-DN50',   'Gate Valve DN50 PN16',           'Valves & Fittings',    'Pcs',        18, 8, 12,   1650000,  FALSE, NULL,           'WH-D2')
ON CONFLICT (sku) DO NOTHING;
