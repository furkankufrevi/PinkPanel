-- Email forwarders for virtual alias maps
CREATE TABLE IF NOT EXISTS email_forwarders (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    domain_id       INTEGER NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    source_address  TEXT    NOT NULL,
    destination     TEXT    NOT NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(domain_id, source_address, destination)
);
CREATE INDEX IF NOT EXISTS idx_email_forwarders_domain_id ON email_forwarders(domain_id);
