package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/db"
)

const githubReleasesURL = "https://api.github.com/repos/furkankufrevi/PinkPanel/releases"

type UpdatesHandler struct {
	DB          *sql.DB
	AgentClient *agent.Client
	Version     string
}

type githubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
	Prerelease  bool   `json:"prerelease"`
	Draft       bool   `json:"draft"`
}

// CheckForUpdates queries GitHub for the latest release and compares with current version.
func (h *UpdatesHandler) CheckForUpdates(c *fiber.Ctx) error {
	client := &http.Client{Timeout: 10 * time.Second}
	// Use /releases (not /releases/latest) because latest skips prereleases
	req, _ := http.NewRequest("GET", githubReleasesURL+"?per_page=5", nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "PinkPanel/"+h.Version)

	resp, err := client.Do(req)
	if err != nil {
		return c.JSON(fiber.Map{
			"current_version":  h.Version,
			"update_available": false,
			"error":            "Failed to check for updates: " + err.Error(),
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return c.JSON(fiber.Map{
			"current_version":  h.Version,
			"update_available": false,
			"error":            fmt.Sprintf("GitHub API returned %d", resp.StatusCode),
		})
	}

	body, _ := io.ReadAll(resp.Body)
	var releases []githubRelease
	if err := json.Unmarshal(body, &releases); err != nil || len(releases) == 0 {
		return c.JSON(fiber.Map{
			"current_version":  h.Version,
			"update_available": false,
		})
	}

	// Find the first non-draft release
	var release *githubRelease
	for i := range releases {
		if !releases[i].Draft {
			release = &releases[i]
			break
		}
	}
	if release == nil {
		return c.JSON(fiber.Map{
			"current_version":  h.Version,
			"update_available": false,
		})
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	updateAvailable := latestVersion != "" && isNewer(latestVersion, h.Version)

	return c.JSON(fiber.Map{
		"current_version":  h.Version,
		"latest_version":   latestVersion,
		"update_available": updateAvailable,
		"release_name":     release.Name,
		"release_notes":    release.Body,
		"release_url":      release.HTMLURL,
		"published_at":     release.PublishedAt,
	})
}

// GetReleases returns recent releases from GitHub.
func (h *UpdatesHandler) GetReleases(c *fiber.Ctx) error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", githubReleasesURL+"?per_page=10", nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "PinkPanel/"+h.Version)

	resp, err := client.Do(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "GITHUB_ERROR",
				"message": "Failed to fetch releases: " + err.Error(),
			},
		})
	}
	defer resp.Body.Close()

	// No releases yet
	if resp.StatusCode == 404 || resp.StatusCode != 200 {
		return c.JSON(fiber.Map{
			"current_version": h.Version,
			"releases":        []fiber.Map{},
		})
	}

	body, _ := io.ReadAll(resp.Body)
	var releases []githubRelease
	json.Unmarshal(body, &releases)

	result := make([]fiber.Map, 0, len(releases))
	for _, r := range releases {
		if r.Draft {
			continue
		}
		ver := strings.TrimPrefix(r.TagName, "v")
		result = append(result, fiber.Map{
			"version":      ver,
			"name":         r.Name,
			"notes":        r.Body,
			"published_at": r.PublishedAt,
			"url":          r.HTMLURL,
			"prerelease":   r.Prerelease,
			"is_current":   ver == h.Version,
			"is_newer":     isNewer(ver, h.Version),
		})
	}

	return c.JSON(fiber.Map{
		"current_version": h.Version,
		"releases":        result,
	})
}

// TriggerUpgrade starts the upgrade process via the agent.
func (h *UpdatesHandler) TriggerUpgrade(c *fiber.Ctx) error {
	adminID, _ := c.Locals("admin_id").(int64)

	// Record upgrade attempt
	h.DB.Exec(
		"INSERT INTO version_history (version, previous_version, status) VALUES ('upgrading', ?, 'in_progress')",
		h.Version,
	)

	resp, err := h.AgentClient.Call("system_upgrade", nil)
	if err != nil {
		h.DB.Exec(
			"UPDATE version_history SET status = 'failed', changelog = ? WHERE status = 'in_progress' ORDER BY id DESC LIMIT 1",
			err.Error(),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "UPGRADE_ERROR",
				"message": "Failed to start upgrade: " + err.Error(),
			},
		})
	}

	db.LogActivity(h.DB, adminID, "system_upgrade", "system", 0, "upgrade started", c.IP())

	return c.JSON(resp.Result)
}

// GetUpgradeStatus returns the current upgrade status and log output.
// It finds the most recent upgrade log file and tails it.
func (h *UpdatesHandler) GetUpgradeStatus(c *fiber.Ctx) error {
	// Check if there's an in_progress upgrade in the DB
	var status string
	var prevVersion *string
	err := h.DB.QueryRow(
		"SELECT status, previous_version FROM version_history ORDER BY id DESC LIMIT 1",
	).Scan(&status, &prevVersion)
	if err != nil {
		status = "idle"
	}

	// Find the most recent upgrade log
	logDir := "/var/log/pinkpanel"
	logContent := ""
	logFile := ""

	entries, err := os.ReadDir(logDir)
	if err == nil {
		var logFiles []string
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "upgrade-") && strings.HasSuffix(e.Name(), ".log") {
				logFiles = append(logFiles, e.Name())
			}
		}
		if len(logFiles) > 0 {
			sort.Strings(logFiles)
			logFile = filepath.Join(logDir, logFiles[len(logFiles)-1])

			// Read with offset support for incremental polling
			offset := c.QueryInt("offset", 0)
			data, err := os.ReadFile(logFile)
			if err == nil {
				content := string(data)
				if offset > 0 && offset < len(content) {
					content = content[offset:]
				} else if offset >= len(content) {
					content = ""
				}
				logContent = content
			}
		}
	}

	// Check if upgrade process is still running
	running := false
	if status == "in_progress" {
		// Check if upgrade.sh is still running
		resp, err := h.AgentClient.Call("exec_command", map[string]string{
			"command": "pgrep -f 'bash.*upgrade.sh' || echo 'not_running'",
		})
		if err == nil {
			if result, ok := resp.Result.(map[string]interface{}); ok {
				if out, ok := result["output"].(string); ok {
					running = !strings.Contains(out, "not_running")
				}
			}
		}
		// If not running, resolve the status
		if !running {
			// Read final log to determine success/failure
			if strings.Contains(logContent, "Upgraded successfully") || strings.Contains(logContent, "is running") {
				h.DB.Exec("UPDATE version_history SET status = 'completed' WHERE status = 'in_progress' ORDER BY id DESC LIMIT 1")
				status = "completed"
			} else if logContent != "" {
				h.DB.Exec("UPDATE version_history SET status = 'failed' WHERE status = 'in_progress' ORDER BY id DESC LIMIT 1")
				status = "failed"
			}
		}
	}

	// Get total log size for offset tracking
	totalSize := 0
	if logFile != "" {
		if info, err := os.Stat(logFile); err == nil {
			totalSize = int(info.Size())
		}
	}

	return c.JSON(fiber.Map{
		"status":     status,
		"running":    running,
		"log":        logContent,
		"log_file":   logFile,
		"total_size": totalSize,
	})
}

// GetUpgradeHistory returns the version upgrade history.
func (h *UpdatesHandler) GetUpgradeHistory(c *fiber.Ctx) error {
	rows, err := h.DB.Query(
		"SELECT id, version, previous_version, changelog, status, created_at FROM version_history ORDER BY id DESC LIMIT 20",
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get upgrade history",
			},
		})
	}
	defer rows.Close()

	type historyEntry struct {
		ID              int64   `json:"id"`
		Version         string  `json:"version"`
		PreviousVersion *string `json:"previous_version"`
		Changelog       *string `json:"changelog"`
		Status          string  `json:"status"`
		CreatedAt       string  `json:"created_at"`
	}

	entries := []historyEntry{}
	for rows.Next() {
		var e historyEntry
		rows.Scan(&e.ID, &e.Version, &e.PreviousVersion, &e.Changelog, &e.Status, &e.CreatedAt)
		entries = append(entries, e)
	}

	return c.JSON(fiber.Map{
		"current_version": h.Version,
		"history":         entries,
	})
}

// isNewer returns true if version a is newer than version b.
func isNewer(a, b string) bool {
	aParts := parseVersion(a)
	bParts := parseVersion(b)
	for i := 0; i < 3; i++ {
		if aParts[i] > bParts[i] {
			return true
		}
		if aParts[i] < bParts[i] {
			return false
		}
	}
	return false
}

func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	// Strip pre-release suffix
	if idx := strings.Index(v, "-"); idx >= 0 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	var result [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		fmt.Sscanf(parts[i], "%d", &result[i])
	}
	return result
}
