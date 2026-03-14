CREATE TABLE IF NOT EXISTS dns_templates (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category    TEXT NOT NULL DEFAULT 'custom',
    is_preset   INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS dns_template_records (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id INTEGER NOT NULL REFERENCES dns_templates(id) ON DELETE CASCADE,
    type        TEXT NOT NULL CHECK (type IN ('A','AAAA','CNAME','MX','TXT','NS','SOA','SRV','CAA')),
    name        TEXT NOT NULL,
    value       TEXT NOT NULL,
    ttl         INTEGER NOT NULL DEFAULT 3600,
    priority    INTEGER DEFAULT NULL
);

CREATE INDEX IF NOT EXISTS idx_dns_template_records_template_id ON dns_template_records(template_id);
