-- Seed demo users matching frontend AuthContext credentials.
-- Password for all demo accounts: admin123
-- Hash generated with bcrypt cost 10.

INSERT INTO users (email, password_hash, name, role, avatar, department, status)
VALUES
    ('admin@shipyard.co.id',
     '$2a$10$uUrqxWMpnQtCoXDTHVcKoOTsGdkn9qeMERiKEyLh53OSfi7lF5HTW',
     'Ahmad Fauzi', 'admin', 'AF', 'IT', 'active'),
    ('supervisor@shipyard.co.id',
     '$2a$10$uUrqxWMpnQtCoXDTHVcKoOTsGdkn9qeMERiKEyLh53OSfi7lF5HTW',
     'Budi Santoso', 'supervisor', 'BS', 'Operations', 'active'),
    ('staff@shipyard.co.id',
     '$2a$10$uUrqxWMpnQtCoXDTHVcKoOTsGdkn9qeMERiKEyLh53OSfi7lF5HTW',
     'Citra Dewi', 'staff', 'CD', 'Warehouse', 'active')
ON CONFLICT (email) DO NOTHING;
