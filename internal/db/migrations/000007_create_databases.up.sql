CREATE TABLE IF NOT EXISTS databases (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    domain_id  INTEGER DEFAULT NULL REFERENCES domains(id) ON DELETE SET NULL,
    name       TEXT    NOT NULL UNIQUE,
    type       TEXT    NOT NULL DEFAULT 'mysql' CHECK (type IN ('mysql', 'mariadb')),
    size_bytes INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_databases_domain_id ON databases(domain_id);

CREATE TABLE IF NOT EXISTS database_users (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    database_id INTEGER NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    username    TEXT    NOT NULL,
    host        TEXT    NOT NULL DEFAULT 'localhost',
    permissions TEXT    NOT NULL DEFAULT 'ALL',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(username, host)
);

CREATE INDEX IF NOT EXISTS idx_database_users_database_id ON database_users(database_id);
