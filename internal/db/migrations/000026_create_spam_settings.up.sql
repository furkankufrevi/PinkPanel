CREATE TABLE IF NOT EXISTS email_spam_settings (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    domain_id       INTEGER NOT NULL UNIQUE REFERENCES domains(id) ON DELETE CASCADE,
    enabled         INTEGER NOT NULL DEFAULT 0,
    score_threshold REAL    NOT NULL DEFAULT 5.0,
    action          TEXT    NOT NULL DEFAULT 'mark',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS email_spam_lists (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    domain_id  INTEGER NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    list_type  TEXT    NOT NULL,
    entry      TEXT    NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(domain_id, list_type, entry)
);
CREATE INDEX IF NOT EXISTS idx_email_spam_lists_domain ON email_spam_lists(domain_id);
