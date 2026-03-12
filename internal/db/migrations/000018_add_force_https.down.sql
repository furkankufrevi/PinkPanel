-- SQLite doesn't support DROP COLUMN before 3.35.0, so we recreate.
CREATE TABLE ssl_certificates_backup AS SELECT
    id, domain_id, type, cert_path, key_path, chain_path, issuer, domains,
    issued_at, expires_at, auto_renew, created_at, updated_at
FROM ssl_certificates;
DROP TABLE ssl_certificates;
CREATE TABLE ssl_certificates (
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
INSERT INTO ssl_certificates SELECT * FROM ssl_certificates_backup;
DROP TABLE ssl_certificates_backup;
CREATE UNIQUE INDEX IF NOT EXISTS idx_ssl_certificates_domain_id ON ssl_certificates(domain_id);
