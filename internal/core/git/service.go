package git

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"regexp"
)

var safeNameRe = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

type Repository struct {
	ID            int64   `json:"id"`
	DomainID      int64   `json:"domain_id"`
	Name          string  `json:"name"`
	RepoType      string  `json:"repo_type"`
	RemoteURL     *string `json:"remote_url"`
	Branch        string  `json:"branch"`
	DeployMode    string  `json:"deploy_mode"`
	DeployPath    string  `json:"deploy_path"`
	PostDeployCmd *string `json:"post_deploy_cmd"`
	WebhookSecret *string `json:"webhook_secret"`
	LastDeployAt  *string `json:"last_deploy_at"`
	LastCommit    *string `json:"last_commit"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type Deployment struct {
	ID          int64   `json:"id"`
	RepoID      int64   `json:"repo_id"`
	CommitHash  *string `json:"commit_hash"`
	Branch      *string `json:"branch"`
	Status      string  `json:"status"`
	Log         *string `json:"log"`
	DurationMs  *int64  `json:"duration_ms"`
	TriggeredBy string  `json:"triggered_by"`
	CreatedAt   string  `json:"created_at"`
}

type Service struct {
	DB *sql.DB
}

func (s *Service) ListRepos(domainID int64) ([]Repository, error) {
	rows, err := s.DB.Query(
		`SELECT id, domain_id, name, repo_type, remote_url, branch, deploy_mode, deploy_path, post_deploy_cmd, webhook_secret, last_deploy_at, last_commit, created_at, updated_at
		 FROM git_repositories WHERE domain_id = ? ORDER BY name`, domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []Repository
	for rows.Next() {
		var r Repository
		if err := rows.Scan(&r.ID, &r.DomainID, &r.Name, &r.RepoType, &r.RemoteURL, &r.Branch, &r.DeployMode, &r.DeployPath, &r.PostDeployCmd, &r.WebhookSecret, &r.LastDeployAt, &r.LastCommit, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

func (s *Service) GetRepo(id int64) (*Repository, error) {
	var r Repository
	err := s.DB.QueryRow(
		`SELECT id, domain_id, name, repo_type, remote_url, branch, deploy_mode, deploy_path, post_deploy_cmd, webhook_secret, last_deploy_at, last_commit, created_at, updated_at
		 FROM git_repositories WHERE id = ?`, id,
	).Scan(&r.ID, &r.DomainID, &r.Name, &r.RepoType, &r.RemoteURL, &r.Branch, &r.DeployMode, &r.DeployPath, &r.PostDeployCmd, &r.WebhookSecret, &r.LastDeployAt, &r.LastCommit, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("git repository not found")
	}
	return &r, nil
}

func (s *Service) CreateRepo(domainID int64, name, repoType, remoteURL, branch, deployPath, deployMode string) (*Repository, error) {
	if !safeNameRe.MatchString(name) {
		return nil, fmt.Errorf("invalid repository name: only alphanumeric, underscore, dot, and hyphen allowed")
	}
	if len(name) < 1 || len(name) > 64 {
		return nil, fmt.Errorf("repository name must be 1-64 characters")
	}
	if repoType != "remote" && repoType != "local" {
		return nil, fmt.Errorf("repo_type must be 'remote' or 'local'")
	}
	if repoType == "remote" && remoteURL == "" {
		return nil, fmt.Errorf("remote_url is required for remote repositories")
	}
	if deployPath == "" {
		return nil, fmt.Errorf("deploy_path is required")
	}
	// Don't default branch — let git use the remote's default (HEAD)
	if deployMode == "" {
		deployMode = "manual"
	}

	// Generate webhook secret
	secret := generateWebhookSecret()

	var remotePtr *string
	if remoteURL != "" {
		remotePtr = &remoteURL
	}

	res, err := s.DB.Exec(
		`INSERT INTO git_repositories (domain_id, name, repo_type, remote_url, branch, deploy_mode, deploy_path, webhook_secret) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		domainID, name, repoType, remotePtr, branch, deployMode, deployPath, secret,
	)
	if err != nil {
		return nil, fmt.Errorf("repository name already exists for this domain")
	}
	id, _ := res.LastInsertId()
	return s.GetRepo(id)
}

func (s *Service) UpdateRepo(id int64, branch, deployMode, deployPath, postDeployCmd string) (*Repository, error) {
	if deployMode != "" && deployMode != "automatic" && deployMode != "manual" && deployMode != "disabled" {
		return nil, fmt.Errorf("deploy_mode must be 'automatic', 'manual', or 'disabled'")
	}

	// Build dynamic update
	sets := []string{}
	args := []any{}

	if branch != "" {
		sets = append(sets, "branch = ?")
		args = append(args, branch)
	}
	if deployMode != "" {
		sets = append(sets, "deploy_mode = ?")
		args = append(args, deployMode)
	}
	if deployPath != "" {
		sets = append(sets, "deploy_path = ?")
		args = append(args, deployPath)
	}
	// post_deploy_cmd can be empty string (to clear it)
	sets = append(sets, "post_deploy_cmd = ?")
	args = append(args, nilIfEmpty(postDeployCmd))

	sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := "UPDATE git_repositories SET "
	for i, s := range sets {
		if i > 0 {
			query += ", "
		}
		query += s
	}
	query += " WHERE id = ?"

	if _, err := s.DB.Exec(query, args...); err != nil {
		return nil, fmt.Errorf("updating repository: %w", err)
	}
	return s.GetRepo(id)
}

func (s *Service) DeleteRepo(id int64) (*Repository, error) {
	r, err := s.GetRepo(id)
	if err != nil {
		return nil, err
	}
	if _, err := s.DB.Exec(`DELETE FROM git_repositories WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Service) UpdateLastDeploy(id int64, commitHash string) error {
	var commitPtr *string
	if commitHash != "" {
		commitPtr = &commitHash
	}
	_, err := s.DB.Exec(
		`UPDATE git_repositories SET last_deploy_at = CURRENT_TIMESTAMP, last_commit = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		commitPtr, id,
	)
	return err
}

func (s *Service) ListDeployments(repoID int64, limit int) ([]Deployment, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.DB.Query(
		`SELECT id, repo_id, commit_hash, branch, status, log, duration_ms, triggered_by, created_at
		 FROM git_deployments WHERE repo_id = ? ORDER BY created_at DESC LIMIT ?`, repoID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deployments []Deployment
	for rows.Next() {
		var d Deployment
		if err := rows.Scan(&d.ID, &d.RepoID, &d.CommitHash, &d.Branch, &d.Status, &d.Log, &d.DurationMs, &d.TriggeredBy, &d.CreatedAt); err != nil {
			return nil, err
		}
		deployments = append(deployments, d)
	}
	return deployments, rows.Err()
}

func (s *Service) CreateDeployment(repoID int64, triggeredBy string) (*Deployment, error) {
	res, err := s.DB.Exec(
		`INSERT INTO git_deployments (repo_id, triggered_by) VALUES (?, ?)`,
		repoID, triggeredBy,
	)
	if err != nil {
		return nil, fmt.Errorf("creating deployment: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetDeployment(id)
}

func (s *Service) GetDeployment(id int64) (*Deployment, error) {
	var d Deployment
	err := s.DB.QueryRow(
		`SELECT id, repo_id, commit_hash, branch, status, log, duration_ms, triggered_by, created_at
		 FROM git_deployments WHERE id = ?`, id,
	).Scan(&d.ID, &d.RepoID, &d.CommitHash, &d.Branch, &d.Status, &d.Log, &d.DurationMs, &d.TriggeredBy, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("deployment not found")
	}
	return &d, nil
}

func (s *Service) UpdateDeployment(id int64, status, commitHash, logOutput string, durationMs int64) error {
	var commitPtr, logPtr *string
	if commitHash != "" {
		commitPtr = &commitHash
	}
	if logOutput != "" {
		logPtr = &logOutput
	}
	var durPtr *int64
	if durationMs > 0 {
		durPtr = &durationMs
	}
	_, err := s.DB.Exec(
		`UPDATE git_deployments SET status = ?, commit_hash = ?, log = ?, duration_ms = ? WHERE id = ?`,
		status, commitPtr, logPtr, durPtr, id,
	)
	return err
}

func (s *Service) GetRepoByWebhookSecret(secret string) (*Repository, error) {
	var r Repository
	err := s.DB.QueryRow(
		`SELECT id, domain_id, name, repo_type, remote_url, branch, deploy_mode, deploy_path, post_deploy_cmd, webhook_secret, last_deploy_at, last_commit, created_at, updated_at
		 FROM git_repositories WHERE webhook_secret = ?`, secret,
	).Scan(&r.ID, &r.DomainID, &r.Name, &r.RepoType, &r.RemoteURL, &r.Branch, &r.DeployMode, &r.DeployPath, &r.PostDeployCmd, &r.WebhookSecret, &r.LastDeployAt, &r.LastCommit, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("repository not found for webhook")
	}
	return &r, nil
}

func generateWebhookSecret() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
