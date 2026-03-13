-- SQLite doesn't support DROP COLUMN prior to 3.35.0, so we recreate the table.
CREATE TABLE ssl_certificates_backup AS SELECT id, domain_id, type, cert_path, key_path, chain_path, issuer, domains, issued_at, expires_at, auto_renew, force_https, created_at, updated_at FROM ssl_certificates;
DROP TABLE ssl_certificates;
ALTER TABLE ssl_certificates_backup RENAME TO ssl_certificates;
