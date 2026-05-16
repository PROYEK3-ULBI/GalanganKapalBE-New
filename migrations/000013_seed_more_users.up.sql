-- Additional users matching frontend mockData (for richer admin dashboard UI).
-- Password for all: admin123 (same hash as initial demo users).

INSERT INTO users (email, password_hash, name, role, avatar, department, status)
VALUES
    ('dedi.k@shipyard.co.id',
     '$2a$10$zHDrfcR0ZHttWfdM/KHoPO80txOJHtGUxq9vnvV52wNy.gWw1wC.i',
     'Dedi Kurniawan',  'supervisor', 'DK', 'QC',           'active'),
    ('eka.p@shipyard.co.id',
     '$2a$10$zHDrfcR0ZHttWfdM/KHoPO80txOJHtGUxq9vnvV52wNy.gWw1wC.i',
     'Eka Prasetya',    'staff',      'EP', 'Warehouse',    'inactive'),
    ('fitri.h@shipyard.co.id',
     '$2a$10$zHDrfcR0ZHttWfdM/KHoPO80txOJHtGUxq9vnvV52wNy.gWw1wC.i',
     'Fitri Handayani', 'staff',      'FH', 'Procurement',  'active'),
    ('gunawan.w@shipyard.co.id',
     '$2a$10$zHDrfcR0ZHttWfdM/KHoPO80txOJHtGUxq9vnvV52wNy.gWw1wC.i',
     'Gunawan Wibowo',  'admin',      'GW', 'IT',           'active'),
    ('hendra.s@shipyard.co.id',
     '$2a$10$zHDrfcR0ZHttWfdM/KHoPO80txOJHtGUxq9vnvV52wNy.gWw1wC.i',
     'Hendra Susanto',  'supervisor', 'HS', 'Engineering',  'active')
ON CONFLICT (email) DO NOTHING;
