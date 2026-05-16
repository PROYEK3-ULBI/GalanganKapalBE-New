-- Seed default vendors matching frontend mockData.

INSERT INTO vendors (name, contact, phone, email, status)
VALUES
    ('PT Krakatau Steel',    'Agus Hermawan',   '021-5551234', 'sales@krakatausteel.co.id', 'active'),
    ('PT Baja Utama',        'Rudi Setiawan',   '031-7778899', 'order@bajautama.co.id',     'active'),
    ('PT Gas Industri',      'Sari Mulyani',    '021-3334455', 'cs@gasindustri.co.id',      'active'),
    ('PT Lincoln Electric',  'Tommy Halim',     '021-6667788', 'id@lincolnelectric.com',    'active'),
    ('PT Jotun Indonesia',   'Dewi Anggraeni',  '021-8889900', 'order@jotun.co.id',         'active')
ON CONFLICT (name) DO NOTHING;
