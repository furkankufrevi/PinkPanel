CREATE TABLE IF NOT EXISTS backups (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    domain_id  INTEGER DEFAULT NULL REFERENCES domains(id) ON DELETE SET NULL,
    type       TEXT    NOT NULL DEFAULT 'full' CHECK (type IN ('full', 'domain')),
    file_path  TEXT    NOT NULL,
    size_bytes INTEGER NOT NULL DEFAULT 0,
    status     TEXT    NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME DEFAULT NULL
);

CREATE INDEX IF NOT EXISTS idx_backups_domain_id ON backups(domain_id);
CREATE INDEX IF NOT EXISTS idx_backups_status ON backups(status);
CREATE INDEX IF NOT EXISTS idx_backups_created_at ON backups(created_at);
