DROP TRIGGER IF EXISTS trg_tickets_updated_at ON support_tickets;
DROP INDEX IF EXISTS idx_tickets_date;
DROP INDEX IF EXISTS idx_tickets_priority;
DROP INDEX IF EXISTS idx_tickets_status;
DROP INDEX IF EXISTS idx_tickets_user;
DROP TABLE IF EXISTS support_tickets;
