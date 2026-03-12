-- Add TOTP two-factor authentication columns to admins
ALTER TABLE admins ADD COLUMN totp_secret TEXT DEFAULT NULL;
ALTER TABLE admins ADD COLUMN totp_enabled INTEGER NOT NULL DEFAULT 0;

-- Recovery codes for 2FA (one-time use backup codes)
CREATE TABLE IF NOT EXISTS recovery_codes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    admin_id INTEGER NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,
    used INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_recovery_codes_admin_id ON recovery_codes(admin_id);
