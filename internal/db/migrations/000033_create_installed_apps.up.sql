CREATE TABLE IF NOT EXISTS installed_apps (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    domain_id     INTEGER NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    app_type      TEXT    NOT NULL,
    app_name      TEXT    NOT NULL,
    version       TEXT    NOT NULL DEFAULT '',
    install_path  TEXT    NOT NULL,
    db_name       TEXT    DEFAULT NULL,
    db_user       TEXT    DEFAULT NULL,
    admin_url     TEXT    DEFAULT NULL,
    status        TEXT    NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','installing','completed','failed','updating','uninstalling')),
    error_message TEXT    DEFAULT NULL,
    install_log   TEXT    DEFAULT NULL,
    installed_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_installed_apps_domain ON installed_apps(domain_id);
