-- Add ModSecurity WAF toggle per domain
ALTER TABLE domains ADD COLUMN modsecurity_enabled INTEGER NOT NULL DEFAULT 0;
