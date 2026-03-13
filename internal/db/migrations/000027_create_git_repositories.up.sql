CREATE TABLE IF NOT EXISTS git_repositories (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    domain_id       INTEGER NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    name            TEXT    NOT NULL,
    repo_type       TEXT    NOT NULL CHECK (repo_type IN ('remote', 'local')),
    remote_url      TEXT    DEFAULT NULL,
    branch          TEXT    NOT NULL DEFAULT 'main',
    deploy_mode     TEXT    NOT NULL DEFAULT 'manual' CHECK (deploy_mode IN ('automatic', 'manual', 'disabled')),
    deploy_path     TEXT    NOT NULL,
    post_deploy_cmd TEXT    DEFAULT NULL,
    webhook_secret  TEXT    DEFAULT NULL,
    last_deploy_at  DATETIME DEFAULT NULL,
    last_commit     TEXT    DEFAULT NULL,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(domain_id, name)
);
CREATE INDEX IF NOT EXISTS idx_git_repos_domain_id ON git_repositories(domain_id);

CREATE TABLE IF NOT EXISTS git_deployments (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id      INTEGER NOT NULL REFERENCES git_repositories(id) ON DELETE CASCADE,
    commit_hash  TEXT    DEFAULT NULL,
    branch       TEXT    DEFAULT NULL,
    status       TEXT    NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    log          TEXT    DEFAULT NULL,
    duration_ms  INTEGER DEFAULT NULL,
    triggered_by TEXT    NOT NULL DEFAULT 'manual',
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_git_deployments_repo_id ON git_deployments(repo_id);
