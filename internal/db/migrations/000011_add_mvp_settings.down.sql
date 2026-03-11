DELETE FROM settings WHERE key IN (
    'panel.hostname', 'panel.port', 'panel.timezone', 'admin.email',
    'nginx.worker_processes', 'php.default_version',
    'backup.retention_days', 'backup.storage_path',
    'log.retention_days', 'log.level'
);
