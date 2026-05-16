DROP TRIGGER IF EXISTS trg_transactions_updated_at ON transactions;
DROP INDEX IF EXISTS idx_transactions_date;
DROP INDEX IF EXISTS idx_transactions_user;
DROP INDEX IF EXISTS idx_transactions_po;
DROP INDEX IF EXISTS idx_transactions_project;
DROP INDEX IF EXISTS idx_transactions_material;
DROP INDEX IF EXISTS idx_transactions_type;
DROP INDEX IF EXISTS idx_transactions_no;
DROP TABLE IF EXISTS transactions;
