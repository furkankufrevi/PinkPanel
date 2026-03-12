-- Email accounts for virtual mailbox hosting
CREATE TABLE IF NOT EXISTS email_accounts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    domain_id   INTEGER NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    address     TEXT    NOT NULL,
    quota_mb    INTEGER NOT NULL DEFAULT 0,
    enabled     INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(domain_id, address)
);
CREATE INDEX IF NOT EXISTS idx_email_accounts_domain_id ON email_accounts(domain_id);
