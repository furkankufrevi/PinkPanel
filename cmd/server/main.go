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
	"github.com/pinkpanel/pinkpanel/internal/core/ssl"
	"github.com/pinkpanel/pinkpanel/internal/db"
	"github.com/pinkpanel/pinkpanel/internal/logger"
)

//go:embed all:static
var embeddedFiles embed.FS

var version = "dev"

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
	domainHandler := &handlers.DomainHandler{
		DB:          database,
		DomainSvc:   domainSvc,
		DNSSvc:      dnsSvc,
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
		AgentClient: agentClient,
	}

	// SSL service & handler
	sslSvc := &ssl.Service{DB: database}
	sslHandler := &handlers.SSLHandler{
		DB:          database,
		SSLSvc:      sslSvc,
		DomainSvc:   domainSvc,
		AgentClient: agentClient,
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

	// API routes
	api := app.Group("/api")

	// Public routes
	api.Get("/health", healthHandler.Health)
	api.Get("/setup/status", setupHandler.Status)
	api.Post("/setup/admin", setupHandler.CreateAdmin)
	api.Post("/auth/login", middleware.RateLimit(5, time.Minute), authHandler.Login)
	api.Post("/auth/refresh", middleware.RateLimit(10, time.Minute), authHandler.Refresh)

	// Protected routes
	protected := api.Group("", middleware.Auth(jwtManager))
	protected.Get("/health/detailed", healthHandler.HealthDetailed)
	protected.Post("/auth/logout", authHandler.Logout)
	protected.Post("/auth/change-password", authHandler.ChangePassword)

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

	// PHP routes
	protected.Get("/php/versions", phpHandler.GetVersions)
	protected.Get("/domains/:id/php", phpHandler.GetDomainPHP)
	protected.Put("/domains/:id/php", phpHandler.UpdateDomainPHP)

	// SSL routes
	protected.Get("/domains/:id/ssl", sslHandler.GetCertificate)
	protected.Post("/domains/:id/ssl", sslHandler.InstallCertificate)
	protected.Delete("/domains/:id/ssl", sslHandler.DeleteCertificate)
	protected.Put("/domains/:id/ssl/auto-renew", sslHandler.ToggleAutoRenew)

	// Database routes
	protected.Get("/databases", databaseHandler.List)
	protected.Get("/databases/:id", databaseHandler.Get)
	protected.Post("/databases", databaseHandler.Create)
	protected.Delete("/databases/:id", databaseHandler.Delete)
	protected.Post("/databases/:id/users", databaseHandler.CreateUser)
	protected.Delete("/databases/:id/users/:userId", databaseHandler.DeleteUser)

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
	protected.Post("/backups/:id/restore", backupHandler.Restore)

	// Log routes
	protected.Get("/logs/sources", logHandler.Sources)
	protected.Get("/logs/system", logHandler.SystemLogs)
	protected.Get("/domains/:id/logs", logHandler.DomainLogs)

	// Settings routes
	protected.Get("/settings/activity", settingsHandler.ActivityLog)
	protected.Get("/settings/server-info", settingsHandler.ServerInfo)

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
