DROP TABLE IF EXISTS login_attempts;

-- SQLite doesn't support DROP COLUMN in older versions,
-- but modernc/sqlite supports it. Drop the added columns.
-- In practice, rolling back multi-user is destructive.
