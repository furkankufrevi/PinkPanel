package handlers

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/core/git"
	"github.com/pinkpanel/pinkpanel/internal/db"
)

type GitHandler struct {
	DB          *sql.DB
	GitSvc      *git.Service
	DomainSvc   *domain.Service
	AgentClient *agent.Client
}

// ListRepos returns git repositories for a domain.
func (h *GitHandler) ListRepos(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	repos, err := h.GitSvc.ListRepos(domainID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if repos == nil {
		repos = []git.Repository{}
	}
	return c.JSON(fiber.Map{"data": repos})
}

// GetRepo returns a single git repository.
func (h *GitHandler) GetRepo(c *fiber.Ctx) error {
	repoID, err := strconv.ParseInt(c.Params("repoId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid repo ID"}})
	}

	repo, err := h.GitSvc.GetRepo(repoID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}
	return c.JSON(repo)
}

// CreateRepo creates a git repository for a domain.
func (h *GitHandler) CreateRepo(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	var req struct {
		Name       string `json:"name"`
		RepoType   string `json:"repo_type"`
		RemoteURL  string `json:"remote_url"`
		Branch     string `json:"branch"`
		DeployPath string `json:"deploy_path"`
		DeployMode string `json:"deploy_mode"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}
	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "name is required"}})
	}

	// Resolve deploy path from domain if not specified
	if req.DeployPath == "" {
		dom, err := h.DomainSvc.GetByID(domainID)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
		}
		req.DeployPath = dom.DocumentRoot
	}

	repo, err := h.GitSvc.CreateRepo(domainID, req.Name, req.RepoType, req.RemoteURL, req.Branch, req.DeployPath, req.DeployMode)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	// For remote repos, clone via agent
	if req.RepoType == "remote" {
		repoWorkDir := fmt.Sprintf("/var/lib/pinkpanel/git/%d/%s", domainID, req.Name)
		branch := req.Branch
		if branch == "" {
			branch = "main"
		}
		if _, err := h.AgentClient.Call("git_clone", map[string]any{
			"url":    req.RemoteURL,
			"path":   repoWorkDir,
			"branch": branch,
		}); err != nil {
			// Rollback DB entry
			h.GitSvc.DeleteRepo(repo.ID)
			return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to clone repository: " + err.Error()}})
		}
	}

	// For local repos, init bare repo via agent
	if req.RepoType == "local" {
		bareRepoPath := fmt.Sprintf("/var/lib/pinkpanel/git/%d/%s.git", domainID, req.Name)
		if _, err := h.AgentClient.Call("git_init_bare", map[string]any{
			"path": bareRepoPath,
		}); err != nil {
			h.GitSvc.DeleteRepo(repo.ID)
			return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to initialize bare repository: " + err.Error()}})
		}

		// Setup post-receive hook for auto-deploy
		if repo.DeployMode == "automatic" {
			if _, err := h.AgentClient.Call("git_setup_hook", map[string]any{
				"repo_path":   bareRepoPath,
				"webhook_url": fmt.Sprintf("http://localhost:8080/api/git/webhook/%s", *repo.WebhookSecret),
			}); err != nil {
				log.Error().Err(err).Msg("failed to setup post-receive hook")
			}
		}
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "create_git_repo", "git", repo.ID, req.Name, c.IP())

	return c.Status(201).JSON(repo)
}

// UpdateRepo updates a git repository's settings.
func (h *GitHandler) UpdateRepo(c *fiber.Ctx) error {
	repoID, err := strconv.ParseInt(c.Params("repoId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid repo ID"}})
	}

	var req struct {
		Branch        string `json:"branch"`
		DeployMode    string `json:"deploy_mode"`
		DeployPath    string `json:"deploy_path"`
		PostDeployCmd string `json:"post_deploy_cmd"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	repo, err := h.GitSvc.UpdateRepo(repoID, req.Branch, req.DeployMode, req.DeployPath, req.PostDeployCmd)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "update_git_repo", "git", repo.ID, repo.Name, c.IP())

	return c.JSON(repo)
}

// DeleteRepo deletes a git repository.
func (h *GitHandler) DeleteRepo(c *fiber.Ctx) error {
	repoID, err := strconv.ParseInt(c.Params("repoId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid repo ID"}})
	}

	repo, err := h.GitSvc.DeleteRepo(repoID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	// Clean up repo directory via agent
	repoDir := fmt.Sprintf("/var/lib/pinkpanel/git/%d/%s", repo.DomainID, repo.Name)
	if repo.RepoType == "local" {
		repoDir = fmt.Sprintf("/var/lib/pinkpanel/git/%d/%s.git", repo.DomainID, repo.Name)
	}
	if _, err := h.AgentClient.Call("file_delete", map[string]any{"path": repoDir}); err != nil {
		log.Error().Err(err).Str("path", repoDir).Msg("failed to clean up git repo directory")
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "delete_git_repo", "git", repoID, repo.Name, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// TriggerDeploy manually triggers a deployment.
func (h *GitHandler) TriggerDeploy(c *fiber.Ctx) error {
	repoID, err := strconv.ParseInt(c.Params("repoId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid repo ID"}})
	}

	repo, err := h.GitSvc.GetRepo(repoID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	if repo.DeployMode == "disabled" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "deployment is disabled for this repository"}})
	}

	dep, err := h.GitSvc.CreateDeployment(repoID, "manual")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "trigger_git_deploy", "git", repo.ID, repo.Name, c.IP())

	// Run deployment async
	go h.runDeploy(repo, dep.ID)

	return c.Status(202).JSON(dep)
}

// ListDeployments returns deployment history for a repository.
func (h *GitHandler) ListDeployments(c *fiber.Ctx) error {
	repoID, err := strconv.ParseInt(c.Params("repoId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid repo ID"}})
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	deployments, err := h.GitSvc.ListDeployments(repoID, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if deployments == nil {
		deployments = []git.Deployment{}
	}
	return c.JSON(fiber.Map{"data": deployments})
}

// WebhookHandler handles incoming webhook requests for auto-deploy.
func (h *GitHandler) WebhookHandler(c *fiber.Ctx) error {
	secret := c.Params("secret")
	if secret == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "missing webhook secret"}})
	}

	repo, err := h.GitSvc.GetRepoByWebhookSecret(secret)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "invalid webhook"}})
	}

	if repo.DeployMode != "automatic" {
		return c.JSON(fiber.Map{"status": "skipped", "message": "automatic deployment is not enabled"})
	}

	dep, err := h.GitSvc.CreateDeployment(repo.ID, "webhook")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}

	// Run deployment async
	go h.runDeploy(repo, dep.ID)

	return c.JSON(fiber.Map{"status": "ok", "deployment_id": dep.ID})
}

// runDeploy executes the deployment process asynchronously.
func (h *GitHandler) runDeploy(repo *git.Repository, deploymentID int64) {
	start := time.Now()

	if err := h.GitSvc.UpdateDeployment(deploymentID, "running", "", "", 0); err != nil {
		log.Error().Err(err).Msg("failed to update deployment status to running")
	}

	var allLogs string
	var commitHash string

	// Step 1: Pull latest (for remote repos)
	if repo.RepoType == "remote" {
		repoWorkDir := fmt.Sprintf("/var/lib/pinkpanel/git/%d/%s", repo.DomainID, repo.Name)
		resp, err := h.AgentClient.Call("git_pull", map[string]any{
			"path":   repoWorkDir,
			"branch": repo.Branch,
		})
		if err != nil {
			duration := time.Since(start).Milliseconds()
			h.GitSvc.UpdateDeployment(deploymentID, "failed", "", "git pull failed: "+err.Error(), duration)
			return
		}
		if resp != nil && resp.Result != nil {
			if m, ok := resp.Result.(map[string]any); ok {
				if l, ok := m["output"].(string); ok {
					allLogs += l + "\n"
				}
			}
		}
	}

	// Step 2: Deploy files
	repoPath := fmt.Sprintf("/var/lib/pinkpanel/git/%d/%s", repo.DomainID, repo.Name)
	if repo.RepoType == "local" {
		repoPath = fmt.Sprintf("/var/lib/pinkpanel/git/%d/%s.git", repo.DomainID, repo.Name)
	}

	postCmd := ""
	if repo.PostDeployCmd != nil {
		postCmd = *repo.PostDeployCmd
	}

	resp, err := h.AgentClient.Call("git_deploy", map[string]any{
		"repo_path":       repoPath,
		"deploy_path":     repo.DeployPath,
		"post_deploy_cmd": postCmd,
	})
	if err != nil {
		duration := time.Since(start).Milliseconds()
		h.GitSvc.UpdateDeployment(deploymentID, "failed", "", allLogs+"deploy failed: "+err.Error(), duration)
		return
	}
	if resp != nil && resp.Result != nil {
		if m, ok := resp.Result.(map[string]any); ok {
			if l, ok := m["output"].(string); ok {
				allLogs += l + "\n"
			}
			if h, ok := m["commit_hash"].(string); ok {
				commitHash = h
			}
		}
	}

	// Step 3: Get latest commit info
	if commitHash == "" {
		logResp, err := h.AgentClient.Call("git_log", map[string]any{
			"path":  repoPath,
			"limit": 1,
		})
		if err == nil && logResp != nil && logResp.Result != nil {
			if m, ok := logResp.Result.(map[string]any); ok {
				if commits, ok := m["commits"].([]any); ok && len(commits) > 0 {
					if c, ok := commits[0].(map[string]any); ok {
						if hash, ok := c["hash"].(string); ok {
							commitHash = hash
						}
					}
				}
			}
		}
	}

	duration := time.Since(start).Milliseconds()
	h.GitSvc.UpdateDeployment(deploymentID, "completed", commitHash, allLogs, duration)
	h.GitSvc.UpdateLastDeploy(repo.ID, commitHash)
}
