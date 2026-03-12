CREATE TABLE IF NOT EXISTS backup_schedules (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    domain_id       INTEGER DEFAULT NULL REFERENCES domains(id) ON DELETE CASCADE,
    frequency       TEXT NOT NULL DEFAULT 'daily' CHECK (frequency IN ('daily', 'weekly', 'monthly')),
    time            TEXT NOT NULL DEFAULT '03:00',
    retention_count INTEGER NOT NULL DEFAULT 5,
    enabled         INTEGER NOT NULL DEFAULT 1,
    last_run        DATETIME DEFAULT NULL,
    next_run        DATETIME DEFAULT NULL,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_backup_schedules_enabled ON backup_schedules(enabled);
CREATE INDEX idx_backup_schedules_next_run ON backup_schedules(next_run);
