-- Seed default projects matching frontend mockData.

INSERT INTO projects (code, name, type, status, completion_pct)
VALUES
    ('H-2026-001',  'Hull 001 - MV Pacific Explorer',     'Bulk Carrier',     'In Progress', 72),
    ('H-2026-002',  'Hull 002 - MT Ocean Star',           'Oil Tanker',       'In Progress', 45),
    ('H-2026-003',  'Hull 003 - KM Nusantara Pride',      'Passenger Ferry',  'In Progress', 28),
    ('H-2026-004',  'Hull 004 - TB Samudra 12',           'Tugboat',          'Planning',     5),
    ('DR-2026-001', 'Drydock - MV Garuda Express',        'Container Ship',   'In Drydock',  60),
    ('DR-2026-002', 'Drydock - KM Bahari Jaya',           'General Cargo',    'In Drydock',  85)
ON CONFLICT (code) DO NOTHING;
