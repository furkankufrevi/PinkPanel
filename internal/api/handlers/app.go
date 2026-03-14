package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/app"
	dbpkg "github.com/pinkpanel/pinkpanel/internal/core/database"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/core/user"
	"github.com/pinkpanel/pinkpanel/internal/db"
)

type AppHandler struct {
	DB          *sql.DB
	AppSvc      *app.Service
	DomainSvc   *domain.Service
	DBSvc       *dbpkg.Service
	UserSvc     *user.Service
	AgentClient *agent.Client
}

// Catalog returns the list of available apps.
func (h *AppHandler) Catalog(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"data": app.GetCatalog()})
}

// ListInstalled returns installed apps for a domain.
func (h *AppHandler) ListInstalled(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	apps, err := h.AppSvc.ListByDomain(domainID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if apps == nil {
		apps = []app.InstalledApp{}
	}
	return c.JSON(fiber.Map{"data": apps})
}

// Get returns a single installed app with its current status.
func (h *AppHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid app ID"}})
	}
	a, err := h.AppSvc.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}
	return c.JSON(a)
}

// GetLogs returns install log text for an app.
func (h *AppHandler) GetLogs(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid app ID"}})
	}
	a, err := h.AppSvc.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}
	logText := ""
	if a.InstallLog != nil {
		logText = *a.InstallLog
	}
	return c.JSON(fiber.Map{"log": logText})
}

type installAppRequest struct {
	AppType    string `json:"app_type"`
	SiteTitle  string `json:"site_title"`
	AdminUser  string `json:"admin_user"`
	AdminPass  string `json:"admin_pass"`
	AdminEmail string `json:"admin_email"`
	DBName     string `json:"db_name"`
	DBUser     string `json:"db_user"`
	DBPass     string `json:"db_pass"`
	InstallPath string `json:"install_path"`
}

// Install starts an async application installation.
func (h *AppHandler) Install(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	var req installAppRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	// Validate app type
	appDef := app.GetAppDef(req.AppType)
	if appDef == nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "unknown app type: " + req.AppType}})
	}

	// Validate domain
	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	// Determine install path
	installPath := dom.DocumentRoot
	if req.InstallPath != "" {
		installPath = dom.DocumentRoot + "/" + strings.TrimPrefix(req.InstallPath, "/")
	}

	// Auto-generate DB credentials if needed but not provided
	if appDef.NeedsDB {
		if req.DBName == "" {
			req.DBName = sanitizeDBName(req.AppType + "_" + strings.ReplaceAll(dom.Name, ".", "_"))
		}
		if req.DBUser == "" {
			req.DBUser = req.DBName
		}
		if req.DBPass == "" {
			req.DBPass = generateRandomPassword(16)
		}
	}

	// Default site title
	if req.SiteTitle == "" {
		req.SiteTitle = dom.Name
	}

	var dbName, dbUser *string
	if appDef.NeedsDB {
		dbName = &req.DBName
		dbUser = &req.DBUser
	}

	// Create install record
	a, err := h.AppSvc.Create(domainID, req.AppType, appDef.Name, "", installPath, dbName, dbUser)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}

	// Resolve system user for file ownership
	systemUser := h.resolveSystemUser(dom)

	// Start async install
	go h.runInstall(a.ID, appDef, dom, req, systemUser)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "install_app", "installed_app", a.ID, appDef.Name+" on "+dom.Name, c.IP())

	return c.Status(201).JSON(a)
}

// Uninstall removes an installed app.
func (h *AppHandler) Uninstall(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid app ID"}})
	}

	a, err := h.AppSvc.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	dropDB := c.Query("drop_db") == "true"

	// Update status
	h.AppSvc.UpdateStatus(id, "uninstalling", nil)

	go func() {
		// Delete app files
		if _, err := h.AgentClient.Call("file_delete", map[string]any{
			"path":      a.InstallPath,
			"recursive": true,
		}); err != nil {
			log.Error().Err(err).Int64("app_id", id).Msg("failed to delete app files")
		}

		// Drop database if requested
		if dropDB && a.DBName != nil && *a.DBName != "" {
			if _, err := h.AgentClient.Call("mysql_drop_db", map[string]any{
				"name": *a.DBName,
			}); err != nil {
				log.Error().Err(err).Str("db", *a.DBName).Msg("failed to drop app database")
			}
			if a.DBUser != nil && *a.DBUser != "" {
				h.AgentClient.Call("mysql_drop_user", map[string]any{
					"username": *a.DBUser,
					"host":     "localhost",
				})
			}
			// Remove from panel DB
			h.DB.Exec("DELETE FROM databases WHERE name = ?", *a.DBName)
		}

		// Delete app record
		h.AppSvc.Delete(id)
	}()

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "uninstall_app", "installed_app", id, a.AppName, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// Update starts an async app update to latest version.
func (h *AppHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid app ID"}})
	}

	a, err := h.AppSvc.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	if a.Status != "completed" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "can only update completed installations"}})
	}

	appDef := app.GetAppDef(a.AppType)
	if appDef == nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "unknown app type"}})
	}

	dom, err := h.DomainSvc.GetByID(a.DomainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	systemUser := h.resolveSystemUser(dom)

	h.AppSvc.UpdateStatus(id, "updating", nil)

	go h.runUpdate(a, appDef, dom, systemUser)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "update_app", "installed_app", id, a.AppName, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// WPInfo returns WordPress-specific info (version, plugins, themes).
func (h *AppHandler) WPInfo(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid app ID"}})
	}

	a, err := h.AppSvc.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}
	if a.AppType != "wordpress" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "not a WordPress installation"}})
	}

	dom, err := h.DomainSvc.GetByID(a.DomainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	systemUser := h.resolveSystemUser(dom)

	// Get version
	versionResp, err := h.AgentClient.Call("app_wpcli", map[string]any{
		"path":   a.InstallPath,
		"run_as": systemUser,
		"args":   []string{"core", "version"},
	})
	version := ""
	if err == nil {
		if r, ok := versionResp.Result.(map[string]any); ok {
			if v, ok := r["output"].(string); ok {
				version = strings.TrimSpace(v)
			}
		}
	}

	// Get plugins
	pluginsResp, _ := h.AgentClient.Call("app_wpcli", map[string]any{
		"path":   a.InstallPath,
		"run_as": systemUser,
		"args":   []string{"plugin", "list", "--format=json"},
	})
	pluginsJSON := "[]"
	if pluginsResp != nil {
		if r, ok := pluginsResp.Result.(map[string]any); ok {
			if v, ok := r["output"].(string); ok {
				pluginsJSON = v
			}
		}
	}

	// Get themes
	themesResp, _ := h.AgentClient.Call("app_wpcli", map[string]any{
		"path":   a.InstallPath,
		"run_as": systemUser,
		"args":   []string{"theme", "list", "--format=json"},
	})
	themesJSON := "[]"
	if themesResp != nil {
		if r, ok := themesResp.Result.(map[string]any); ok {
			if v, ok := r["output"].(string); ok {
				themesJSON = v
			}
		}
	}

	return c.JSON(fiber.Map{
		"version":      version,
		"plugins_json": pluginsJSON,
		"themes_json":  themesJSON,
	})
}

// WPMaintenance toggles WordPress maintenance mode.
func (h *AppHandler) WPMaintenance(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid app ID"}})
	}

	a, err := h.AppSvc.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}
	if a.AppType != "wordpress" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "not a WordPress installation"}})
	}

	var req struct {
		Enable bool `json:"enable"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	dom, err := h.DomainSvc.GetByID(a.DomainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	systemUser := h.resolveSystemUser(dom)

	action := "deactivate"
	if req.Enable {
		action = "activate"
	}

	resp, err := h.AgentClient.Call("app_wpcli", map[string]any{
		"path":   a.InstallPath,
		"run_as": systemUser,
		"args":   []string{"maintenance-mode", action},
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "wp-cli failed: " + err.Error()}})
	}

	output := ""
	if r, ok := resp.Result.(map[string]any); ok {
		if v, ok := r["output"].(string); ok {
			output = v
		}
	}

	return c.JSON(fiber.Map{"status": "ok", "output": output})
}

// resolveSystemUser returns the Linux system user for a domain.
func (h *AppHandler) resolveSystemUser(dom *domain.Domain) string {
	if dom.AdminID != nil {
		sysUser, err := h.UserSvc.GetSystemUsername(*dom.AdminID)
		if err == nil && sysUser != "" {
			return sysUser
		}
	}
	return "www-data"
}

// runInstall executes the full app installation in background.
func (h *AppHandler) runInstall(appID int64, appDef *app.AppDefinition, dom *domain.Domain, req installAppRequest, systemUser string) {
	logLine := func(msg string) {
		h.AppSvc.AppendLog(appID, msg)
	}
	fail := func(msg string) {
		errMsg := msg
		h.AppSvc.UpdateStatus(appID, "failed", &errMsg)
		logLine("ERROR: " + msg)
	}

	h.AppSvc.UpdateStatus(appID, "installing", nil)
	logLine("Starting installation of " + appDef.Name + "...")

	installPath := dom.DocumentRoot
	if req.InstallPath != "" {
		installPath = dom.DocumentRoot + "/" + strings.TrimPrefix(req.InstallPath, "/")
	}

	// Step 1: Create database if needed
	if appDef.NeedsDB {
		logLine("Creating database " + req.DBName + "...")
		if _, err := h.AgentClient.Call("mysql_create_db", map[string]any{
			"name": req.DBName,
		}); err != nil {
			fail("Failed to create database: " + err.Error())
			return
		}

		logLine("Creating database user " + req.DBUser + "...")
		if _, err := h.AgentClient.Call("mysql_create_user", map[string]any{
			"username": req.DBUser,
			"password": req.DBPass,
			"host":     "localhost",
		}); err != nil {
			fail("Failed to create database user: " + err.Error())
			return
		}

		if _, err := h.AgentClient.Call("mysql_grant", map[string]any{
			"database":    req.DBName,
			"username":    req.DBUser,
			"host":        "localhost",
			"permissions": "ALL PRIVILEGES",
		}); err != nil {
			fail("Failed to grant database privileges: " + err.Error())
			return
		}

		// Track DB in panel
		h.DBSvc.Create(req.DBName, &dom.ID)
	}

	// Step 2: Download and extract
	logLine("Downloading " + appDef.Name + "...")
	if _, err := h.AgentClient.Call("app_download", map[string]any{
		"url":    appDef.DownloadURL,
		"dest":   installPath,
		"format": appDef.ArchiveFormat,
		"subdir": appDef.ExtractSubdir,
	}); err != nil {
		fail("Failed to download: " + err.Error())
		h.cleanupOnFailure(appDef, req)
		return
	}

	// Step 3: App-specific configuration
	switch appDef.Slug {
	case "wordpress":
		h.installWordPress(appID, appDef, dom, req, installPath, systemUser)
		return
	case "phpmyadmin":
		h.installPhpMyAdmin(appID, installPath, systemUser)
		return
	default:
		// Generic apps: just set permissions
	}

	// Step 4: Set permissions
	logLine("Setting file permissions...")
	h.AgentClient.Call("set_ownership", map[string]any{
		"path":      installPath,
		"owner":     systemUser,
		"group":     systemUser,
		"recursive": true,
	})

	logLine("Installation completed successfully!")
	h.AppSvc.UpdateStatus(appID, "completed", nil)
}

func (h *AppHandler) installWordPress(appID int64, appDef *app.AppDefinition, dom *domain.Domain, req installAppRequest, installPath, systemUser string) {
	logLine := func(msg string) {
		h.AppSvc.AppendLog(appID, msg)
	}
	fail := func(msg string) {
		errMsg := msg
		h.AppSvc.UpdateStatus(appID, "failed", &errMsg)
		logLine("ERROR: " + msg)
	}

	// Generate wp-config.php
	logLine("Configuring WordPress...")
	wpConfig := generateWPConfig(req.DBName, req.DBUser, req.DBPass, "localhost")
	if _, err := h.AgentClient.Call("file_write", map[string]any{
		"path":    installPath + "/wp-config.php",
		"content": wpConfig,
		"mode":    "0644",
	}); err != nil {
		fail("Failed to write wp-config.php: " + err.Error())
		h.cleanupOnFailure(appDef, req)
		return
	}

	// Set permissions
	logLine("Setting file permissions...")
	h.AgentClient.Call("set_ownership", map[string]any{
		"path":      installPath,
		"owner":     systemUser,
		"group":     systemUser,
		"recursive": true,
	})

	// Run WP-CLI install
	logLine("Running WordPress installer...")
	wpArgs := []string{
		"core", "install",
		"--url=https://" + dom.Name,
		"--title=" + req.SiteTitle,
		"--admin_user=" + req.AdminUser,
		"--admin_password=" + req.AdminPass,
		"--admin_email=" + req.AdminEmail,
		"--skip-email",
	}
	wpResp, wpErr := h.AgentClient.Call("app_wpcli", map[string]any{
		"path":   installPath,
		"run_as": systemUser,
		"args":   wpArgs,
	})
	// Log wp-cli output regardless of success/failure
	if wpResp != nil {
		if r, ok := wpResp.Result.(map[string]any); ok {
			if v, ok := r["output"].(string); ok && v != "" {
				logLine("wp-cli: " + v)
			}
		}
	}
	if wpErr != nil {
		fail("WordPress install failed: " + wpErr.Error())
		h.cleanupOnFailure(appDef, req)
		return
	}

	// Detect version
	logLine("Detecting WordPress version...")
	versionResp, err := h.AgentClient.Call("app_wpcli", map[string]any{
		"path":   installPath,
		"run_as": systemUser,
		"args":   []string{"core", "version"},
	})
	if err == nil {
		if r, ok := versionResp.Result.(map[string]any); ok {
			if v, ok := r["output"].(string); ok {
				h.AppSvc.UpdateVersion(appID, strings.TrimSpace(v))
			}
		}
	}

	// Set admin URL
	h.AppSvc.SetAdminURL(appID, "https://"+dom.Name+"/wp-admin/")

	logLine("WordPress installed successfully!")
	h.AppSvc.UpdateStatus(appID, "completed", nil)
}

func (h *AppHandler) installPhpMyAdmin(appID int64, installPath, systemUser string) {
	logLine := func(msg string) {
		h.AppSvc.AppendLog(appID, msg)
	}

	// Generate config
	logLine("Configuring phpMyAdmin...")
	blowfishSecret := generateRandomPassword(32)
	config := fmt.Sprintf(`<?php
$cfg['blowfish_secret'] = '%s';
$i = 0;
$i++;
$cfg['Servers'][$i]['auth_type'] = 'cookie';
$cfg['Servers'][$i]['host'] = 'localhost';
$cfg['Servers'][$i]['compress'] = false;
$cfg['Servers'][$i]['AllowNoPassword'] = false;
$cfg['UploadDir'] = '';
$cfg['SaveDir'] = '';
`, blowfishSecret)

	h.AgentClient.Call("file_write", map[string]any{
		"path":    installPath + "/config.inc.php",
		"content": config,
		"mode":    "0644",
	})

	// Set permissions
	logLine("Setting file permissions...")
	h.AgentClient.Call("set_ownership", map[string]any{
		"path":      installPath,
		"owner":     systemUser,
		"group":     systemUser,
		"recursive": true,
	})

	logLine("phpMyAdmin installed successfully!")
	h.AppSvc.UpdateStatus(appID, "completed", nil)
}

func (h *AppHandler) runUpdate(a *app.InstalledApp, appDef *app.AppDefinition, dom *domain.Domain, systemUser string) {
	logLine := func(msg string) {
		h.AppSvc.AppendLog(a.ID, msg)
	}
	fail := func(msg string) {
		errMsg := msg
		h.AppSvc.UpdateStatus(a.ID, "failed", &errMsg)
		logLine("ERROR: " + msg)
	}

	logLine("Starting update of " + appDef.Name + "...")

	if appDef.Slug == "wordpress" && appDef.HasCLI {
		logLine("Updating WordPress core...")
		resp, err := h.AgentClient.Call("app_wpcli", map[string]any{
			"path":   a.InstallPath,
			"run_as": systemUser,
			"args":   []string{"core", "update"},
		})
		if err != nil {
			fail("WordPress update failed: " + err.Error())
			return
		}
		if r, ok := resp.Result.(map[string]any); ok {
			if v, ok := r["output"].(string); ok {
				logLine(v)
			}
		}

		// Update database
		logLine("Updating WordPress database...")
		h.AgentClient.Call("app_wpcli", map[string]any{
			"path":   a.InstallPath,
			"run_as": systemUser,
			"args":   []string{"core", "update-db"},
		})

		// Get new version
		versionResp, err := h.AgentClient.Call("app_wpcli", map[string]any{
			"path":   a.InstallPath,
			"run_as": systemUser,
			"args":   []string{"core", "version"},
		})
		if err == nil {
			if r, ok := versionResp.Result.(map[string]any); ok {
				if v, ok := r["output"].(string); ok {
					h.AppSvc.UpdateVersion(a.ID, strings.TrimSpace(v))
				}
			}
		}
	} else {
		// Generic update: re-download
		logLine("Downloading latest version...")
		if _, err := h.AgentClient.Call("app_download", map[string]any{
			"url":    appDef.DownloadURL,
			"dest":   a.InstallPath,
			"format": appDef.ArchiveFormat,
			"subdir": appDef.ExtractSubdir,
		}); err != nil {
			fail("Download failed: " + err.Error())
			return
		}

		logLine("Setting file permissions...")
		h.AgentClient.Call("set_ownership", map[string]any{
			"path":      a.InstallPath,
			"owner":     systemUser,
			"group":     systemUser,
			"recursive": true,
		})
	}

	logLine("Update completed successfully!")
	h.AppSvc.UpdateStatus(a.ID, "completed", nil)
}

func (h *AppHandler) cleanupOnFailure(appDef *app.AppDefinition, req installAppRequest) {
	if appDef.NeedsDB && req.DBName != "" {
		h.AgentClient.Call("mysql_drop_db", map[string]any{"name": req.DBName})
		if req.DBUser != "" {
			h.AgentClient.Call("mysql_drop_user", map[string]any{
				"username": req.DBUser,
				"host":     "localhost",
			})
		}
	}
}

func generateWPConfig(dbName, dbUser, dbPass, dbHost string) string {
	salts := make([]string, 8)
	saltKeys := []string{
		"AUTH_KEY", "SECURE_AUTH_KEY", "LOGGED_IN_KEY", "NONCE_KEY",
		"AUTH_SALT", "SECURE_AUTH_SALT", "LOGGED_IN_SALT", "NONCE_SALT",
	}
	for i := range salts {
		salts[i] = fmt.Sprintf("define('%s', '%s');", saltKeys[i], generateRandomPassword(64))
	}

	return fmt.Sprintf(`<?php
define('DB_NAME', '%s');
define('DB_USER', '%s');
define('DB_PASSWORD', '%s');
define('DB_HOST', '%s');
define('DB_CHARSET', 'utf8mb4');
define('DB_COLLATE', '');

%s

$table_prefix = 'wp_';

define('WP_DEBUG', false);

if ( ! defined( 'ABSPATH' ) ) {
	define( 'ABSPATH', __DIR__ . '/' );
}

require_once ABSPATH . 'wp-settings.php';
`, dbName, dbUser, dbPass, dbHost, strings.Join(salts, "\n"))
}

func generateRandomPassword(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

func sanitizeDBName(name string) string {
	// Keep only alphanumeric and underscores, max 64 chars
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}
	s := result.String()
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}
