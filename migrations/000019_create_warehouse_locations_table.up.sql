-- Warehouse storage locations referenced by materials.location and tools.location.

CREATE TABLE IF NOT EXISTS warehouse_locations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(50)  NOT NULL UNIQUE,
    name        VARCHAR(255),
    type        VARCHAR(50),
    capacity    INTEGER,
    notes       TEXT,
    status      VARCHAR(20)  NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_warehouse_status ON warehouse_locations(status);

DROP TRIGGER IF EXISTS trg_warehouse_updated_at ON warehouse_locations;
CREATE TRIGGER trg_warehouse_updated_at
BEFORE UPDATE ON warehouse_locations
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Seed default locations matching frontend mockData.
INSERT INTO warehouse_locations (code, name, type)
VALUES
    ('Yard-A1',  'Yard A1', 'Outdoor Yard'),
    ('Yard-A2',  'Yard A2', 'Outdoor Yard'),
    ('Yard-A3',  'Yard A3', 'Outdoor Yard'),
    ('Yard-A4',  'Yard A4', 'Outdoor Yard'),
    ('Yard-D1',  'Yard D1', 'Outdoor Yard'),
    ('WH-B1',    'Warehouse B1', 'Indoor Warehouse'),
    ('WH-C1',    'Warehouse C1', 'Indoor Warehouse'),
    ('WH-D2',    'Warehouse D2', 'Indoor Warehouse'),
    ('WH-E1',    'Warehouse E1', 'Indoor Warehouse'),
    ('WH-F1',    'Warehouse F1', 'Indoor Warehouse'),
    ('WH-G1',    'Warehouse G1', 'Indoor Warehouse'),
    ('GAS-YARD', 'Gas Storage Yard', 'HAZMAT')
ON CONFLICT (code) DO NOTHING;
