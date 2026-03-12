package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
