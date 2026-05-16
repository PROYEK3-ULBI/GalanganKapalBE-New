DROP TRIGGER IF EXISTS trg_materials_updated_at ON materials;
DROP INDEX IF EXISTS idx_materials_low_stock;
DROP INDEX IF EXISTS idx_materials_hazmat;
DROP INDEX IF EXISTS idx_materials_category;
DROP INDEX IF EXISTS idx_materials_sku;
DROP TABLE IF EXISTS materials;
