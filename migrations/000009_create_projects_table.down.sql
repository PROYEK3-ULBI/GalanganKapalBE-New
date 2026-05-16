DROP TRIGGER IF EXISTS trg_projects_updated_at ON projects;
DROP INDEX IF EXISTS idx_projects_status;
DROP INDEX IF EXISTS idx_projects_code;
DROP TABLE IF EXISTS projects;
