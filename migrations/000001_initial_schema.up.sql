CREATE TABLE configurations (
    name TEXT PRIMARY KEY,
    current_version INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    configuration_name TEXT NOT NULL,
    version_number INTEGER NOT NULL,
    json_data TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (configuration_name) REFERENCES configurations(name),
    UNIQUE(configuration_name, version_number)
);

-- Index for fast configuration lookups (supports FR-006, FR-007)
CREATE INDEX idx_configurations_name ON configurations(name);

-- Index for version queries (supports FR-007, FR-010)
CREATE INDEX idx_versions_config_version ON versions(configuration_name, version_number);

-- Index for latest version lookups (supports FR-006)
CREATE INDEX idx_versions_config_created ON versions(configuration_name, created_at DESC);
