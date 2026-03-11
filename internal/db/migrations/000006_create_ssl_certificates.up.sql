CREATE TABLE IF NOT EXISTS ssl_certificates (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    domain_id  INTEGER NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    type       TEXT    NOT NULL DEFAULT 'letsencrypt' CHECK (type IN ('letsencrypt', 'custom')),
    cert_path  TEXT    NOT NULL,
    key_path   TEXT    NOT NULL,
    chain_path TEXT    DEFAULT NULL,
    issuer     TEXT    DEFAULT NULL,
    domains    TEXT    DEFAULT NULL,
    issued_at  DATETIME DEFAULT NULL,
    expires_at DATETIME NOT NULL,
    auto_renew INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ssl_certificates_domain_id ON ssl_certificates(domain_id);
