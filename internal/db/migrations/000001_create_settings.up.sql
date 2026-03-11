CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Default settings
INSERT OR IGNORE INTO settings (key, value) VALUES ('panel.name', 'PinkPanel');
INSERT OR IGNORE INTO settings (key, value) VALUES ('panel.version', '0.1.0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('panel.setup_complete', 'false');
