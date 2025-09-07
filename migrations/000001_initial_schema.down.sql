-- Drop indexes first
DROP INDEX IF EXISTS idx_versions_config_created;
DROP INDEX IF EXISTS idx_versions_config_version;
DROP INDEX IF EXISTS idx_configurations_name;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS versions;
DROP TABLE IF EXISTS configurations;
