-- Tools (welders, cutters, gauges, etc.) with checkout/return tracking and calibration dates.

CREATE TABLE IF NOT EXISTS tools (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku                   VARCHAR(50)  NOT NULL UNIQUE,
    name                  VARCHAR(255) NOT NULL,
    category              VARCHAR(100) NOT NULL,
    status                VARCHAR(20)  NOT NULL DEFAULT 'Available'
                          CHECK (status IN ('Available', 'In Use', 'Maintenance')),
    condition             VARCHAR(50)  NOT NULL DEFAULT 'Good'
                          CHECK (condition IN ('Good', 'Fair', 'Needs Repair', 'Out of Order')),
    location              VARCHAR(100),
    borrower_id           UUID         REFERENCES users(id) ON DELETE SET NULL,
    borrow_date           DATE,
    calibration_due_date  DATE,
    notes                 TEXT,
    image_url             TEXT,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- Domain integrity: a tool 'In Use' must have a borrower; otherwise must not.
    CONSTRAINT borrower_consistent CHECK (
        (status = 'In Use' AND borrower_id IS NOT NULL AND borrow_date IS NOT NULL)
        OR
        (status <> 'In Use' AND borrower_id IS NULL AND borrow_date IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_tools_sku        ON tools(sku);
CREATE INDEX IF NOT EXISTS idx_tools_status     ON tools(status);
CREATE INDEX IF NOT EXISTS idx_tools_category   ON tools(category);
CREATE INDEX IF NOT EXISTS idx_tools_borrower   ON tools(borrower_id);
CREATE INDEX IF NOT EXISTS idx_tools_calib_due  ON tools(calibration_due_date);

DROP TRIGGER IF EXISTS trg_tools_updated_at ON tools;
CREATE TRIGGER trg_tools_updated_at
BEFORE UPDATE ON tools
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Tool checkout history (audit trail of borrow/return actions).
CREATE TABLE IF NOT EXISTS tool_history (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tool_id      UUID         NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    action       VARCHAR(20)  NOT NULL CHECK (action IN ('checkout', 'return', 'maintenance', 'available')),
    user_id      UUID         REFERENCES users(id) ON DELETE SET NULL,
    notes        TEXT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tool_history_tool ON tool_history(tool_id);
CREATE INDEX IF NOT EXISTS idx_tool_history_user ON tool_history(user_id);
CREATE INDEX IF NOT EXISTS idx_tool_history_date ON tool_history(created_at DESC);
