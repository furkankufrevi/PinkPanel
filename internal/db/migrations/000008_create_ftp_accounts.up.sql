CREATE TABLE IF NOT EXISTS ftp_accounts (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    domain_id INTEGER NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    username  TEXT    NOT NULL UNIQUE,
    home_dir  TEXT    NOT NULL,
    quota_mb  INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ftp_accounts_domain_id ON ftp_accounts(domain_id);
