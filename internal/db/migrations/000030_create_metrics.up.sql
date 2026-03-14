CREATE TABLE IF NOT EXISTS system_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cpu_usage REAL NOT NULL,
    ram_used INTEGER NOT NULL,
    ram_total INTEGER NOT NULL,
    load_avg_1 REAL NOT NULL,
    load_avg_5 REAL NOT NULL,
    load_avg_15 REAL NOT NULL,
    collected_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_system_metrics_collected_at ON system_metrics(collected_at);

CREATE TABLE IF NOT EXISTS domain_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    domain_id INTEGER NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    disk_usage_bytes INTEGER NOT NULL DEFAULT 0,
    bandwidth_bytes INTEGER NOT NULL DEFAULT 0,
    collected_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_domain_metrics_domain_collected ON domain_metrics(domain_id, collected_at);
