DROP INDEX IF EXISTS idx_mr_items_material;
DROP INDEX IF EXISTS idx_mr_items_request;
DROP TABLE IF EXISTS material_request_items;

DROP TRIGGER IF EXISTS trg_mr_updated_at ON material_requests;
DROP INDEX IF EXISTS idx_mr_date;
DROP INDEX IF EXISTS idx_mr_project;
DROP INDEX IF EXISTS idx_mr_requester;
DROP INDEX IF EXISTS idx_mr_status;
DROP TABLE IF EXISTS material_requests;
