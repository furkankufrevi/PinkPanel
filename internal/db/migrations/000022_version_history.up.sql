-- Track version upgrade history and changelogs
CREATE TABLE IF NOT EXISTS version_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    version TEXT NOT NULL,
    previous_version TEXT,
    changelog TEXT,
    status TEXT NOT NULL DEFAULT 'completed',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_version_history_version ON version_history(version);
