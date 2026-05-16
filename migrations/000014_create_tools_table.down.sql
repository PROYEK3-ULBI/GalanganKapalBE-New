DROP INDEX IF EXISTS idx_tool_history_date;
DROP INDEX IF EXISTS idx_tool_history_user;
DROP INDEX IF EXISTS idx_tool_history_tool;
DROP TABLE IF EXISTS tool_history;

DROP TRIGGER IF EXISTS trg_tools_updated_at ON tools;
DROP INDEX IF EXISTS idx_tools_calib_due;
DROP INDEX IF EXISTS idx_tools_borrower;
DROP INDEX IF EXISTS idx_tools_category;
DROP INDEX IF EXISTS idx_tools_status;
DROP INDEX IF EXISTS idx_tools_sku;
DROP TABLE IF EXISTS tools;
