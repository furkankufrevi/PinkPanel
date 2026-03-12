-- Add role and status to admins table
ALTER TABLE admins ADD COLUMN role TEXT NOT NULL DEFAULT 'super_admin';
ALTER TABLE admins ADD COLUMN status TEXT NOT NULL DEFAULT 'active';

-- Add admin_id to domains
ALTER TABLE domains ADD COLUMN admin_id INTEGER REFERENCES admins(id);
UPDATE domains SET admin_id = (SELECT id FROM admins ORDER BY id LIMIT 1);

-- Add admin_id to databases
ALTER TABLE databases ADD COLUMN admin_id INTEGER REFERENCES admins(id);
UPDATE databases SET admin_id = (SELECT id FROM admins ORDER BY id LIMIT 1);

-- Add admin_id to ftp_accounts
ALTER TABLE ftp_accounts ADD COLUMN admin_id INTEGER REFERENCES admins(id);
UPDATE ftp_accounts SET admin_id = (SELECT id FROM admins ORDER BY id LIMIT 1);

-- Add admin_id to backups
ALTER TABLE backups ADD COLUMN admin_id INTEGER REFERENCES admins(id);
UPDATE backups SET admin_id = (SELECT id FROM admins ORDER BY id LIMIT 1);

-- Add admin_id to backup_schedules
ALTER TABLE backup_schedules ADD COLUMN admin_id INTEGER REFERENCES admins(id);
UPDATE backup_schedules SET admin_id = (SELECT id FROM admins ORDER BY id LIMIT 1);

-- Indexes for admin_id columns
CREATE INDEX IF NOT EXISTS idx_domains_admin_id ON domains(admin_id);
CREATE INDEX IF NOT EXISTS idx_databases_admin_id ON databases(admin_id);
CREATE INDEX IF NOT EXISTS idx_ftp_accounts_admin_id ON ftp_accounts(admin_id);
CREATE INDEX IF NOT EXISTS idx_backups_admin_id ON backups(admin_id);
CREATE INDEX IF NOT EXISTS idx_backup_schedules_admin_id ON backup_schedules(admin_id);

-- Login attempts table for brute force protection
CREATE TABLE IF NOT EXISTS login_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL,
    ip_address TEXT NOT NULL,
    success INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_login_attempts_email ON login_attempts(email, created_at);
CREATE INDEX IF NOT EXISTS idx_login_attempts_ip ON login_attempts(ip_address, created_at);
