-- Seed default tools matching frontend mockData.

INSERT INTO tools (sku, name, category, status, condition, location, calibration_due_date)
VALUES
    ('TL-WLD-500A', 'Portable Welder 500A',          'Welding',     'Available',   'Good',         'Tool Room A',  '2026-06-15'),
    ('TL-LSR-PRX',  'Laser Alignment Pro X',         'Measurement', 'Available',   'Good',         'Tool Room A',  '2026-05-10'),
    ('TL-HYD-TW1',  'Hydraulic Torque Wrench',       'Fastening',   'Available',   'Good',         'Tool Room A',  '2026-07-20'),
    ('TL-UTG-001',  'Ultrasonic Thickness Gauge',    'Measurement', 'Available',   'Good',         'Drydock Area', '2026-05-30'),
    ('TL-PLS-120',  'Plasma Cutter CNC 120A',        'Cutting',     'Maintenance', 'Needs Repair', 'Workshop B',   '2026-05-05'),
    ('TL-MAG-DP1',  'Magnetic Drill Press',          'Drilling',    'Available',   'Good',         'Tool Room B',  '2026-08-01'),
    ('TL-OXF-SET',  'Oxy-Fuel Cutting Set',          'Cutting',     'Available',   'Good',         'Yard Storage', '2026-09-15'),
    ('TL-GEN-10K',  'Portable Generator 10KVA',      'Power',       'Available',   'Good',         'Yard Storage', NULL)
ON CONFLICT (sku) DO NOTHING;
