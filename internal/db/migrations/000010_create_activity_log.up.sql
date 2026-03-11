CREATE TABLE IF NOT EXISTS activity_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    admin_id    INTEGER NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
    action      TEXT    NOT NULL,
    target_type TEXT    DEFAULT NULL,
    target_id   INTEGER DEFAULT NULL,
    details     TEXT    DEFAULT NULL,
    ip_address  TEXT    DEFAULT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_activity_log_admin_id ON activity_log(admin_id);
CREATE INDEX IF NOT EXISTS idx_activity_log_created_at ON activity_log(created_at);
CREATE INDEX IF NOT EXISTS idx_activity_log_target ON activity_log(target_type, target_id);
