-- Drop trigger
DROP TRIGGER IF EXISTS update_drivers_updated_at ON drivers;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_drivers_deleted_at;
DROP INDEX IF EXISTS idx_drivers_current_route_id;
DROP INDEX IF EXISTS idx_drivers_status;
DROP INDEX IF EXISTS idx_drivers_license_number;
DROP INDEX IF EXISTS idx_drivers_email;

-- Drop table
DROP TABLE IF EXISTS drivers;
