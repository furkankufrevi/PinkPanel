package main

import (
	"embed"
	"fmt"
	"flag"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/rs/zerolog"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/api/handlers"
	"github.com/pinkpanel/pinkpanel/internal/api/middleware"
	"github.com/pinkpanel/pinkpanel/internal/auth"
	"github.com/pinkpanel/pinkpanel/internal/config"
	dbpkg "github.com/pinkpanel/pinkpanel/internal/core/database"
	"github.com/pinkpanel/pinkpanel/internal/core/dns"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/core/php"
	"github.com/pinkpanel/pinkpanel/internal/core/backup"
	"github.com/pinkpanel/pinkpanel/internal/core/ftp"
	emailpkg "github.com/pinkpanel/pinkpanel/internal/core/email"
	cronpkg "github.com/pinkpanel/pinkpanel/internal/core/cron"
	"github.com/pinkpanel/pinkpanel/internal/core/monitor"
	"github.com/pinkpanel/pinkpanel/internal/core/redirect"
	gitpkg "github.com/pinkpanel/pinkpanel/internal/core/git"
	sslpkg "github.com/pinkpanel/pinkpanel/internal/core/ssl"
	"github.com/pinkpanel/pinkpanel/internal/core/user"
	"github.com/pinkpanel/pinkpanel/internal/db"
	"github.com/pinkpanel/pinkpanel/internal/logger"
	ws "github.com/pinkpanel/pinkpanel/internal/websocket"

	fiberws "github.com/gofiber/websocket/v2"
)

//go:embed all:static
var embeddedFiles embed.FS

var version = "0.8.3-alpha"

func main() {
	// Parse flags
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// Also check PINKPANEL_CONFIG env var
	cfgFile := *configPath
	if cfgFile == "" {
		cfgFile = os.Getenv("PINKPANEL_CONFIG")
	}

	// Load configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	log := logger.Setup(cfg.Logging)
	log.Info().Str("version", version).Msg("starting PinkPanel")

	// Open database
	database, err := db.Open(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open database")
	}
	defer database.Close()
	log.Info().Str("path", cfg.Database.Path).Msg("database connected")

	// Run migrations
	if err := db.Migrate(database); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}
	log.Info().Msg("database migrations complete")

	// Resolve any stale in_progress upgrade entries (upgrade restarts the service,
	// so if we're starting up, any in_progress upgrade has completed)
	res, _ := database.Exec(
		"UPDATE version_history SET version = ?, status = 'completed' WHERE status = 'in_progress'",
		version,
	)
	if n, _ := res.RowsAffected(); n > 0 {
		log.Info().Int64("count", n).Msg("resolved in_progress upgrade entries")
	}

	// Initialize JWT manager
	jwtManager, err := auth.NewJWTManager(
		cfg.Security.JWTSecretFile,
		cfg.Security.AccessTokenTTL,
		cfg.Security.RefreshTokenTTL,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize JWT manager")
	}
	log.Info().Msg("JWT manager initialized")

	// Connect to agent
	agentClient := agent.NewClient(cfg.Agent.Socket)
	if err := agentClient.Connect(); err != nil {
		log.Warn().Err(err).Msg("agent not reachable (will retry via heartbeat)")
	} else {
		log.Info().Str("socket", cfg.Agent.Socket).Msg("agent connected")
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "PinkPanel",
		DisableStartupMessage: true,
		BodyLimit:             100 * 1024 * 1024, // 100MB for file uploads
	})

	// Global middleware
	app.Use(middleware.SecurityHeaders())
	app.Use(middleware.RequestID())
	app.Use(middleware.RequestLogger(log))

	// Health handler
	healthHandler := &handlers.HealthHandler{
		DB:          database,
		AgentSocket: cfg.Agent.Socket,
		Version:     version,
		StartTime:   time.Now(),
	}

	// Auth handler
	authHandler := &handlers.AuthHandler{
		DB:         database,
		JWTManager: jwtManager,
		BcryptCost: cfg.Security.BcryptCost,
	}

	// Setup handler
	setupHandler := &handlers.SetupHandler{
		DB:         database,
		JWTManager: jwtManager,
		BcryptCost: cfg.Security.BcryptCost,
	}

	// Domain service & handler
	domainSvc := &domain.Service{DB: database}
	dnsSvc := &dns.Service{DB: database}
	sslSvc := &sslpkg.Service{DB: database}
	domainHandler := &handlers.DomainHandler{
		DB:          database,
		DomainSvc:   domainSvc,
		DNSSvc:      dnsSvc,
		SSLSvc:      sslSvc,
		AgentClient: agentClient,
	}

	// DNS handler
	dnsHandler := &handlers.DNSHandler{
		DB:          database,
		DNSSvc:      dnsSvc,
		DomainSvc:   domainSvc,
		AgentClient: agentClient,
	}

	// PHP service & handler
	phpSvc := &php.Service{DB: database}
	phpHandler := &handlers.PHPHandler{
		DB:          database,
		PHPSvc:      phpSvc,
		DomainSvc:   domainSvc,
		SSLSvc:      sslSvc,
		AgentClient: agentClient,
	}

	// SSL handler
	acmeSvc := &sslpkg.ACMEService{
		Email:       "admin@localhost", // will be updated from DB settings
		AgentClient: agentClient,
		DNSSvc:      dnsSvc,
	}
	// Try to load admin email from DB for ACME
	var adminEmail string
	if err := database.QueryRow(`SELECT email FROM admins LIMIT 1`).Scan(&adminEmail); err == nil && adminEmail != "" {
		acmeSvc.Email = adminEmail
	}
	sslHandler := &handlers.SSLHandler{
		DB:          database,
		SSLSvc:      sslSvc,
		DomainSvc:   domainSvc,
		DNSSvc:      dnsSvc,
		AgentClient: agentClient,
		ACMESvc:     acmeSvc,
	}

	// Database service & handler
	dbSvc := &dbpkg.Service{DB: database}
	databaseHandler := &handlers.DatabaseHandler{
		DB:          database,
		DBSvc:       dbSvc,
		AgentClient: agentClient,
	}

	// FTP service & handler
	ftpSvc := &ftp.Service{DB: database}
	ftpHandler := &handlers.FTPHandler{
		DB:          database,
		FTPSvc:      ftpSvc,
		DomainSvc:   domainSvc,
		AgentClient: agentClient,
	}

	// Backup service & handler
	backupSvc := &backup.Service{DB: database}
	scheduleSvc := &backup.ScheduleService{DB: database}
	backupHandler := &handlers.BackupHandler{
		DB:          database,
		BackupSvc:   backupSvc,
		DomainSvc:   domainSvc,
		DBSvc:       dbSvc,
		AgentClient: agentClient,
	}

	// Log handler
	logHandler := &handlers.LogHandler{
		DB:          database,
		DomainSvc:   domainSvc,
		AgentClient: agentClient,
	}

	// Settings handler
	settingsHandler := &handlers.SettingsHandler{
		DB:          database,
		AgentClient: agentClient,
		Version:     version,
	}

	// File handler
	fileHandler := &handlers.FileHandler{
		DB:          database,
		DomainSvc:   domainSvc,
		AgentClient: agentClient,
	}

	// User service & handler
	userSvc := &user.Service{DB: database}
	userHandler := &handlers.UserHandler{
		DB:          database,
		UserSvc:     userSvc,
		AgentClient: agentClient,
		BcryptCost:  cfg.Security.BcryptCost,
	}

	// Email service & handler
	emailSvc := &emailpkg.Service{DB: database}
	emailHandler := &handlers.EmailHandler{
		DB:          database,
		EmailSvc:    emailSvc,
		DomainSvc:   domainSvc,
		DNSSvc:      dnsSvc,
		AgentClient: agentClient,
		ACMESvc:     acmeSvc,
	}

	// Git service & handler
	gitSvc := &gitpkg.Service{DB: database}
	gitHandler := &handlers.GitHandler{
		DB:          database,
		GitSvc:      gitSvc,
		DomainSvc:   domainSvc,
		AgentClient: agentClient,
	}

	// Cron service & handler
	cronSvc := &cronpkg.Service{DB: database}
	cronHandler := &handlers.CronHandler{
		DB:          database,
		CronSvc:     cronSvc,
		DomainSvc:   domainSvc,
		UserSvc:     userSvc,
		AgentClient: agentClient,
	}

	// DNS template service & handler
	dnsTemplateSvc := &dns.TemplateService{DB: database}
	dnsTemplateHandler := &handlers.DNSTemplateHandler{
		DB:          database,
		TemplateSvc: dnsTemplateSvc,
		DNSSvc:      dnsSvc,
		DomainSvc:   domainSvc,
		AgentClient: agentClient,
	}

	// Redirect service & handler
	redirectSvc := &redirect.Service{DB: database}
	redirectHandler := &handlers.RedirectHandler{
		DB:          database,
		RedirectSvc: redirectSvc,
		DomainSvc:   domainSvc,
		AgentClient: agentClient,
	}

	// Monitor service & handler
	monitorSvc := &monitor.Service{
		DB:          database,
		AgentClient: agentClient,
		DomainSvc:   domainSvc,
	}
	monitorHandler := &handlers.MonitorHandler{MonitorSvc: monitorSvc}

	// Security handler (Fail2ban)
	securityHandler := &handlers.SecurityHandler{
		DB:          database,
		AgentClient: agentClient,
	}

	// TOTP handler
	totpHandler := &handlers.TOTPHandler{
		DB:         database,
		JWTManager: jwtManager,
		BcryptCost: cfg.Security.BcryptCost,
	}

	// WebSocket hub for real-time dashboard metrics
	wsHub := ws.NewHub(agentClient)

	// API routes
	api := app.Group("/api")

	// Public routes
	api.Get("/health", healthHandler.Health)
	api.Get("/setup/status", setupHandler.Status)
	api.Post("/setup/admin", setupHandler.CreateAdmin)
	api.Post("/auth/login", middleware.RateLimit(5, time.Minute), authHandler.Login)
	api.Post("/auth/refresh", middleware.RateLimit(10, time.Minute), authHandler.Refresh)
	api.Post("/auth/2fa/verify", middleware.RateLimit(10, time.Minute), totpHandler.Verify)

	// Protected routes
	protected := api.Group("", middleware.Auth(jwtManager))
	protected.Get("/health/detailed", healthHandler.HealthDetailed)
	protected.Post("/auth/logout", authHandler.Logout)
	protected.Post("/auth/change-password", authHandler.ChangePassword)
	protected.Get("/auth/profile", authHandler.Profile)
	protected.Get("/auth/sessions", authHandler.ListSessions)
	protected.Delete("/auth/sessions/:id", authHandler.RevokeSession)

	// 2FA routes
	protected.Get("/auth/2fa/status", totpHandler.Status)
	protected.Post("/auth/2fa/setup", totpHandler.Setup)
	protected.Post("/auth/2fa/enable", totpHandler.Enable)
	protected.Post("/auth/2fa/disable", totpHandler.Disable)
	protected.Post("/auth/2fa/recovery-codes/regenerate", totpHandler.RegenerateRecoveryCodes)

	// User management routes (admin+ only)
	adminOnly := protected.Group("", middleware.RequireRole("super_admin", "admin"))
	adminOnly.Get("/users", userHandler.List)
	adminOnly.Get("/users/:id", userHandler.Get)
	adminOnly.Post("/users", middleware.RequireRole("super_admin"), userHandler.Create)
	adminOnly.Put("/users/:id", userHandler.Update)
	adminOnly.Delete("/users/:id", middleware.RequireRole("super_admin"), userHandler.Delete)
	adminOnly.Post("/users/:id/suspend", userHandler.Suspend)
	adminOnly.Post("/users/:id/activate", userHandler.Activate)
	adminOnly.Post("/users/:id/reset-password", userHandler.ResetPassword)

	// WebSocket route for real-time metrics (uses Upgrade middleware)
	api.Use("/dashboard/live", func(c *fiber.Ctx) error {
		if fiberws.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	api.Get("/dashboard/live", fiberws.New(wsHub.HandleConnection))

	// Terminal WebSocket (auth via query param, not middleware)
	terminalHandler := &handlers.TerminalHandler{
		AgentSocketPath: cfg.Agent.Socket,
		JWTManager:      jwtManager,
		AgentClient:     agentClient,
	}
	api.Use("/terminal/ws", func(c *fiber.Ctx) error {
		if fiberws.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	api.Get("/terminal/ws", fiberws.New(terminalHandler.HandleTerminal))

	// Domain routes
	protected.Get("/domains", domainHandler.List)
	protected.Post("/domains", domainHandler.Create)
	protected.Get("/domains/:id", domainHandler.Get)
	protected.Put("/domains/:id", domainHandler.Update)
	protected.Delete("/domains/:id", domainHandler.Delete)
	protected.Post("/domains/:id/suspend", domainHandler.Suspend)
	protected.Post("/domains/:id/activate", domainHandler.Activate)

	// DNS routes
	protected.Get("/domains/:id/dns", dnsHandler.ListRecords)
	protected.Post("/domains/:id/dns", dnsHandler.CreateRecord)
	protected.Post("/domains/:id/dns/reset", dnsHandler.ResetDefaults)
	protected.Put("/dns/:id", dnsHandler.UpdateRecord)
	protected.Delete("/dns/:id", dnsHandler.DeleteRecord)

	// DNS template routes (import must come before :id to avoid param capture)
	protected.Get("/dns/templates", dnsTemplateHandler.ListTemplates)
	protected.Post("/dns/templates", dnsTemplateHandler.CreateTemplate)
	protected.Post("/dns/templates/import", dnsTemplateHandler.ImportTemplate)
	protected.Get("/dns/templates/:id", dnsTemplateHandler.GetTemplate)
	protected.Put("/dns/templates/:id", dnsTemplateHandler.UpdateTemplate)
	protected.Delete("/dns/templates/:id", dnsTemplateHandler.DeleteTemplate)
	protected.Get("/dns/templates/:id/export", dnsTemplateHandler.ExportTemplate)
	protected.Post("/domains/:id/dns/apply-template", dnsTemplateHandler.ApplyTemplate)
	protected.Post("/domains/:id/dns/save-template", dnsTemplateHandler.SaveAsTemplate)

	// PHP routes
	protected.Get("/php/versions", phpHandler.GetVersions)
	protected.Get("/domains/:id/php", phpHandler.GetDomainPHP)
	protected.Put("/domains/:id/php", phpHandler.UpdateDomainPHP)
	protected.Get("/domains/:id/php/info", phpHandler.GetPHPInfo)

	// SSL routes
	protected.Get("/domains/:id/ssl", sslHandler.GetCertificate)
	protected.Post("/domains/:id/ssl", sslHandler.InstallCertificate)
	protected.Post("/domains/:id/ssl/issue", sslHandler.IssueLetsEncrypt)
	protected.Delete("/domains/:id/ssl", sslHandler.DeleteCertificate)
	protected.Put("/domains/:id/ssl/auto-renew", sslHandler.ToggleAutoRenew)
	protected.Put("/domains/:id/ssl/force-https", sslHandler.ToggleForceHTTPS)
	protected.Put("/domains/:id/ssl/hsts", sslHandler.ToggleHSTS)
	protected.Put("/domains/:id/modsecurity", domainHandler.ToggleModSecurity)

	// Database routes
	protected.Get("/databases", databaseHandler.List)
	protected.Get("/databases/:id", databaseHandler.Get)
	protected.Post("/databases", databaseHandler.Create)
	protected.Delete("/databases/:id", databaseHandler.Delete)
	protected.Post("/databases/:id/users", databaseHandler.CreateUser)
	protected.Delete("/databases/:id/users/:userId", databaseHandler.DeleteUser)
	protected.Post("/databases/:id/phpmyadmin", databaseHandler.PhpMyAdmin)

	// FTP routes
	protected.Get("/ftp", ftpHandler.List)
	protected.Get("/ftp/:id", ftpHandler.Get)
	protected.Post("/ftp", ftpHandler.Create)
	protected.Delete("/ftp/:id", ftpHandler.Delete)
	protected.Put("/ftp/:id/quota", ftpHandler.UpdateQuota)

	// Backup routes
	protected.Get("/backups", backupHandler.List)
	protected.Get("/backups/:id", backupHandler.Get)
	protected.Post("/backups", backupHandler.Create)
	protected.Delete("/backups/:id", backupHandler.Delete)
	protected.Get("/backups/:id/download", backupHandler.Download)
	protected.Post("/backups/:id/restore", backupHandler.Restore)

	// Backup schedule routes
	scheduleHandler := &handlers.BackupScheduleHandler{
		DB:          database,
		ScheduleSvc: scheduleSvc,
	}
	protected.Get("/backup-schedules", scheduleHandler.List)
	protected.Get("/backup-schedules/:id", scheduleHandler.Get)
	protected.Post("/backup-schedules", scheduleHandler.Create)
	protected.Put("/backup-schedules/:id", scheduleHandler.Update)
	protected.Delete("/backup-schedules/:id", scheduleHandler.Delete)

	// Log routes
	protected.Get("/logs/sources", logHandler.Sources)
	protected.Get("/logs/system", logHandler.SystemLogs)
	protected.Get("/domains/:id/logs", logHandler.DomainLogs)
	protected.Get("/domains/:id/logs/download", logHandler.DownloadDomainLog)

	// Settings routes
	protected.Get("/settings/activity", settingsHandler.ActivityLog)
	protected.Get("/settings/server-info", settingsHandler.ServerInfo)

	// Metrics routes
	protected.Get("/metrics/system", monitorHandler.SystemHistory)
	protected.Get("/metrics/system/current", monitorHandler.SystemCurrent)
	protected.Get("/domains/:id/metrics", monitorHandler.DomainMetrics)

	// Updates routes (admin+ only)
	updatesHandler := &handlers.UpdatesHandler{
		DB:          database,
		AgentClient: agentClient,
		Version:     version,
	}
	adminOnly.Get("/updates/check", updatesHandler.CheckForUpdates)
	adminOnly.Get("/updates/releases", updatesHandler.GetReleases)
	adminOnly.Post("/updates/upgrade", updatesHandler.TriggerUpgrade)
	adminOnly.Get("/updates/upgrade/status", updatesHandler.GetUpgradeStatus)
	adminOnly.Get("/updates/history", updatesHandler.GetUpgradeHistory)

	// Security routes (admin+ only)
	adminOnly.Get("/security/fail2ban/status", securityHandler.Fail2banStatus)
	adminOnly.Get("/security/fail2ban/jails/:jail", securityHandler.Fail2banJailStatus)
	adminOnly.Get("/security/fail2ban/banned", securityHandler.Fail2banBannedIPs)
	adminOnly.Post("/security/fail2ban/ban", securityHandler.Fail2banBanIP)
	adminOnly.Post("/security/fail2ban/unban", securityHandler.Fail2banUnbanIP)

	// Email routes
	domainEmail := protected.Group("/domains/:id/email")
	domainEmail.Get("/accounts", emailHandler.ListAccounts)
	domainEmail.Post("/accounts", emailHandler.CreateAccount)
	domainEmail.Delete("/accounts/:accountId", emailHandler.DeleteAccount)
	domainEmail.Put("/accounts/:accountId/quota", emailHandler.UpdateQuota)
	domainEmail.Put("/accounts/:accountId/password", emailHandler.ChangePassword)
	domainEmail.Put("/accounts/:accountId/toggle", emailHandler.ToggleAccount)
	domainEmail.Get("/forwarders", emailHandler.ListForwarders)
	domainEmail.Post("/forwarders", emailHandler.CreateForwarder)
	domainEmail.Delete("/forwarders/:fwdId", emailHandler.DeleteForwarder)
	domainEmail.Get("/dns-records", emailHandler.GetDNSRecords)
	domainEmail.Post("/dns-records", emailHandler.ApplyDNSRecords)
	domainEmail.Post("/accounts/:accountId/webmail", emailHandler.Webmail)
	domainEmail.Get("/spam", emailHandler.GetSpamSettings)
	domainEmail.Put("/spam", emailHandler.UpdateSpamSettings)
	domainEmail.Get("/spam/list/:type", emailHandler.ListSpamEntries)
	domainEmail.Post("/spam/list", emailHandler.AddSpamEntry)
	domainEmail.Delete("/spam/list/:entryId", emailHandler.DeleteSpamEntry)
	domainEmail.Get("/autodiscovery", emailHandler.GetAutodiscoveryStatus)
	domainEmail.Post("/autodiscovery", emailHandler.SetupAutodiscovery)
	domainEmail.Post("/mail-vhost", emailHandler.SetupMailVhost)
	domainEmail.Get("/mail-ssl", emailHandler.GetMailSSLStatus)
	domainEmail.Post("/mail-ssl", emailHandler.ConfigureMailSSL)
	adminOnly.Get("/email/queue", emailHandler.ListQueue)
	adminOnly.Post("/email/queue/flush", emailHandler.FlushQueue)
	adminOnly.Delete("/email/queue/:queueId", emailHandler.DeleteQueueItem)
	adminOnly.Get("/email/clamav", emailHandler.GetClamAVStatus)
	adminOnly.Put("/email/clamav", emailHandler.ToggleClamAV)

	// Git routes
	protected.Get("/git/ssh-key", gitHandler.GetSSHKey)
	protected.Get("/domains/:id/git", gitHandler.ListRepos)
	protected.Post("/domains/:id/git", gitHandler.CreateRepo)
	protected.Get("/domains/:id/git/:repoId", gitHandler.GetRepo)
	protected.Put("/domains/:id/git/:repoId", gitHandler.UpdateRepo)
	protected.Delete("/domains/:id/git/:repoId", gitHandler.DeleteRepo)
	protected.Post("/domains/:id/git/:repoId/deploy", gitHandler.TriggerDeploy)
	protected.Get("/domains/:id/git/:repoId/deployments", gitHandler.ListDeployments)

	// Cron routes
	protected.Get("/domains/:id/crons", cronHandler.List)
	protected.Post("/domains/:id/crons", cronHandler.Create)
	protected.Get("/crons/:id", cronHandler.Get)
	protected.Put("/crons/:id", cronHandler.Update)
	protected.Delete("/crons/:id", cronHandler.Delete)
	protected.Post("/crons/:id/run", cronHandler.RunNow)
	protected.Get("/crons/:id/logs", cronHandler.GetLogs)

	// Redirect routes
	protected.Get("/domains/:id/redirects", redirectHandler.List)
	protected.Post("/domains/:id/redirects", redirectHandler.Create)
	protected.Get("/redirects/:id", redirectHandler.Get)
	protected.Put("/redirects/:id", redirectHandler.Update)
	protected.Delete("/redirects/:id", redirectHandler.Delete)

	// Git webhook (public, no auth)
	api.Post("/git/webhook/:secret", gitHandler.WebhookHandler)

	// File manager routes
	protected.Get("/domains/:id/files", fileHandler.List)
	protected.Get("/domains/:id/files/read", fileHandler.Read)
	protected.Post("/domains/:id/files/save", fileHandler.Save)
	protected.Post("/domains/:id/files/delete", fileHandler.Delete)
	protected.Post("/domains/:id/files/rename", fileHandler.Rename)
	protected.Post("/domains/:id/files/copy", fileHandler.Copy)
	protected.Post("/domains/:id/files/mkdir", fileHandler.CreateDirectory)
	protected.Post("/domains/:id/files/extract", fileHandler.Extract)
	protected.Post("/domains/:id/files/permissions", fileHandler.SetPermissions)
	protected.Post("/domains/:id/files/upload", fileHandler.Upload)
	protected.Get("/domains/:id/files/download", fileHandler.Download)
	protected.Post("/domains/:id/files/compress", fileHandler.Compress)
	protected.Get("/domains/:id/files/search", fileHandler.Search)

	// Global file manager routes (all websites, rooted at /var/www)
	globalFileHandler := &handlers.GlobalFileHandler{
		DB:          database,
		AgentClient: agentClient,
	}
	protected.Get("/files", globalFileHandler.List)
	protected.Get("/files/read", globalFileHandler.Read)
	protected.Post("/files/save", globalFileHandler.Save)
	protected.Post("/files/delete", globalFileHandler.Delete)
	protected.Post("/files/rename", globalFileHandler.Rename)
	protected.Post("/files/mkdir", globalFileHandler.CreateDirectory)
	protected.Post("/files/extract", globalFileHandler.Extract)
	protected.Post("/files/upload", globalFileHandler.Upload)
	protected.Get("/files/download", globalFileHandler.Download)
	protected.Post("/files/compress", globalFileHandler.Compress)
	protected.Get("/files/search", globalFileHandler.Search)

	// Serve embedded frontend
	distFS, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create sub filesystem")
	}

	app.Use("/", filesystem.New(filesystem.Config{
		Root:         http.FS(distFS),
		Browse:       false,
		Index:        "index.html",
		NotFoundFile: "index.html",
	}))

	// Start SSL auto-renewal service
	renewalSvc := &sslpkg.RenewalService{
		SSLSvc:      sslSvc,
		ACMESvc:     acmeSvc,
		AgentClient: agentClient,
	}
	renewalSvc.Start()

	// Start backup scheduler
	backupScheduler := &backup.BackupScheduler{
		ScheduleSvc: scheduleSvc,
		BackupSvc:   backupSvc,
		AgentClient: agentClient,
		DomainDB:    database,
	}
	backupScheduler.Start()

	// Start monitor service
	monitorSvc.Start()

	// Start WebSocket hub (with sparkline data from monitor)
	wsHub.SetMonitorService(monitorSvc)
	wsHub.Start()

	// Agent heartbeat
	stopHeartbeat := make(chan struct{})
	heartbeatCh := agentClient.Heartbeat(30*time.Second, stopHeartbeat)
	go func() {
		for status := range heartbeatCh {
			if !status {
				log.Warn().Msg("agent heartbeat failed")
			}
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-quit
		log.Info().Msg("shutting down gracefully...")
		wsHub.Stop()
		monitorSvc.Stop()
		renewalSvc.Stop()
		backupScheduler.Stop()
		close(stopHeartbeat)
		agentClient.Close()
		if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
			log.Error().Err(err).Msg("error during shutdown")
		}
	}()

	listenAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Info().Str("address", listenAddr).Msg("server listening")

	if err := app.Listen(listenAddr); err != nil {
		log.Fatal().Err(err).Msg("server failed")
	}

	// Cleanup
	log.Info().Msg("PinkPanel stopped")
}

func init() {
	zerolog.TimeFieldFormat = time.RFC3339
}
