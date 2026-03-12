package backup

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
)

// Schedule represents a backup schedule configuration.
type Schedule struct {
	ID             int64   `json:"id"`
	DomainID       *int64  `json:"domain_id"`
	Frequency      string  `json:"frequency"`
	Time           string  `json:"time"`
	RetentionCount int     `json:"retention_count"`
	Enabled        bool    `json:"enabled"`
	LastRun        *string `json:"last_run"`
	NextRun        *string `json:"next_run"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// ScheduleService manages backup schedules.
type ScheduleService struct {
	DB *sql.DB
}

func (s *ScheduleService) List() ([]Schedule, error) {
	rows, err := s.DB.Query(`SELECT id, domain_id, frequency, time, retention_count, enabled, last_run, next_run, created_at, updated_at FROM backup_schedules ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []Schedule
	for rows.Next() {
		var sc Schedule
		var enabled int
		if err := rows.Scan(&sc.ID, &sc.DomainID, &sc.Frequency, &sc.Time, &sc.RetentionCount, &enabled, &sc.LastRun, &sc.NextRun, &sc.CreatedAt, &sc.UpdatedAt); err != nil {
			return nil, err
		}
		sc.Enabled = enabled == 1
		schedules = append(schedules, sc)
	}
	return schedules, rows.Err()
}

func (s *ScheduleService) GetByID(id int64) (*Schedule, error) {
	var sc Schedule
	var enabled int
	err := s.DB.QueryRow(
		`SELECT id, domain_id, frequency, time, retention_count, enabled, last_run, next_run, created_at, updated_at FROM backup_schedules WHERE id = ?`, id,
	).Scan(&sc.ID, &sc.DomainID, &sc.Frequency, &sc.Time, &sc.RetentionCount, &enabled, &sc.LastRun, &sc.NextRun, &sc.CreatedAt, &sc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("schedule not found")
	}
	sc.Enabled = enabled == 1
	return &sc, nil
}

func (s *ScheduleService) Create(domainID *int64, frequency, timeStr string, retentionCount int) (*Schedule, error) {
	if frequency != "daily" && frequency != "weekly" && frequency != "monthly" {
		return nil, fmt.Errorf("invalid frequency: must be daily, weekly, or monthly")
	}
	if retentionCount < 1 {
		retentionCount = 5
	}
	if timeStr == "" {
		timeStr = "03:00"
	}

	nextRun := calcNextRun(frequency, timeStr)

	res, err := s.DB.Exec(
		`INSERT INTO backup_schedules (domain_id, frequency, time, retention_count, enabled, next_run) VALUES (?, ?, ?, ?, 1, ?)`,
		domainID, frequency, timeStr, retentionCount, nextRun.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetByID(id)
}

func (s *ScheduleService) Update(id int64, frequency, timeStr string, retentionCount int, enabled bool) (*Schedule, error) {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	nextRun := calcNextRun(frequency, timeStr)
	_, err := s.DB.Exec(
		`UPDATE backup_schedules SET frequency = ?, time = ?, retention_count = ?, enabled = ?, next_run = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		frequency, timeStr, retentionCount, enabledInt, nextRun.Format(time.RFC3339), id,
	)
	if err != nil {
		return nil, err
	}
	return s.GetByID(id)
}

func (s *ScheduleService) Delete(id int64) error {
	result, err := s.DB.Exec(`DELETE FROM backup_schedules WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("schedule not found")
	}
	return nil
}

func (s *ScheduleService) MarkRun(id int64, frequency, timeStr string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	nextRun := calcNextRun(frequency, timeStr)
	_, err := s.DB.Exec(
		`UPDATE backup_schedules SET last_run = ?, next_run = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		now, nextRun.Format(time.RFC3339), id,
	)
	return err
}

// GetDueSchedules returns schedules whose next_run is in the past and are enabled.
func (s *ScheduleService) GetDueSchedules() ([]Schedule, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := s.DB.Query(
		`SELECT id, domain_id, frequency, time, retention_count, enabled, last_run, next_run, created_at, updated_at
		 FROM backup_schedules WHERE enabled = 1 AND next_run <= ?`, now,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []Schedule
	for rows.Next() {
		var sc Schedule
		var enabled int
		if err := rows.Scan(&sc.ID, &sc.DomainID, &sc.Frequency, &sc.Time, &sc.RetentionCount, &enabled, &sc.LastRun, &sc.NextRun, &sc.CreatedAt, &sc.UpdatedAt); err != nil {
			return nil, err
		}
		sc.Enabled = enabled == 1
		schedules = append(schedules, sc)
	}
	return schedules, rows.Err()
}

func calcNextRun(frequency, timeStr string) time.Time {
	now := time.Now().UTC()
	hour, min := 3, 0
	fmt.Sscanf(timeStr, "%d:%d", &hour, &min)

	next := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, time.UTC)

	switch frequency {
	case "daily":
		if next.Before(now) {
			next = next.Add(24 * time.Hour)
		}
	case "weekly":
		if next.Before(now) {
			next = next.Add(24 * time.Hour)
		}
		// Advance to next Sunday
		for next.Weekday() != time.Sunday {
			next = next.Add(24 * time.Hour)
		}
	case "monthly":
		if next.Before(now) || next.Day() != 1 {
			// First of next month
			next = time.Date(now.Year(), now.Month()+1, 1, hour, min, 0, 0, time.UTC)
		}
	}
	return next
}

// BackupScheduler runs scheduled backups in the background.
type BackupScheduler struct {
	ScheduleSvc *ScheduleService
	BackupSvc   *Service
	AgentClient *agent.Client
	DomainDB    *sql.DB // for querying domain info
	stop        chan struct{}
}

// Start begins the scheduler loop.
func (bs *BackupScheduler) Start() {
	bs.stop = make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				bs.runDueSchedules()
			case <-bs.stop:
				return
			}
		}
	}()
	log.Info().Msg("backup scheduler started (checks every minute)")
}

// Stop halts the scheduler.
func (bs *BackupScheduler) Stop() {
	if bs.stop != nil {
		close(bs.stop)
	}
}

func (bs *BackupScheduler) runDueSchedules() {
	schedules, err := bs.ScheduleSvc.GetDueSchedules()
	if err != nil {
		log.Error().Err(err).Msg("failed to get due backup schedules")
		return
	}

	for _, sc := range schedules {
		go bs.executeSchedule(sc)
	}
}

func (bs *BackupScheduler) executeSchedule(sc Schedule) {
	logger := log.With().Int64("schedule_id", sc.ID).Str("frequency", sc.Frequency).Logger()
	logger.Info().Msg("executing scheduled backup")

	// Determine backup type and gather paths
	var sourcePaths []string
	var databases []string
	backupType := "full"
	timestamp := time.Now().UTC().Format("20060102-150405")
	var fileName string

	if sc.DomainID != nil {
		backupType = "domain"
		var domName, docRoot string
		err := bs.DomainDB.QueryRow(`SELECT name, document_root FROM domains WHERE id = ?`, *sc.DomainID).Scan(&domName, &docRoot)
		if err != nil {
			logger.Error().Err(err).Msg("domain not found for scheduled backup")
			return
		}
		sourcePaths = append(sourcePaths, filepath.Dir(docRoot))
		// Get domain databases
		rows, _ := bs.DomainDB.Query(`SELECT name FROM databases WHERE domain_id = ?`, *sc.DomainID)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var dbName string
				rows.Scan(&dbName)
				databases = append(databases, dbName)
			}
		}
		fileName = fmt.Sprintf("scheduled-domain-%s-%s.tar.gz", domName, timestamp)
	} else {
		// Full backup
		rows, _ := bs.DomainDB.Query(`SELECT document_root FROM domains WHERE parent_id IS NULL`)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var docRoot string
				rows.Scan(&docRoot)
				sourcePaths = append(sourcePaths, filepath.Dir(docRoot))
			}
		}
		dbRows, _ := bs.DomainDB.Query(`SELECT name FROM databases`)
		if dbRows != nil {
			defer dbRows.Close()
			for dbRows.Next() {
				var dbName string
				dbRows.Scan(&dbName)
				databases = append(databases, dbName)
			}
		}
		fileName = fmt.Sprintf("scheduled-full-%s.tar.gz", timestamp)
	}

	filePath := filepath.Join("/var/backups/pinkpanel", fileName)

	// Create backup record
	b, err := bs.BackupSvc.Create(sc.DomainID, backupType, filePath)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create backup record")
		return
	}

	bs.BackupSvc.UpdateStatus(b.ID, "running", 0)

	resp, err := bs.AgentClient.Call("backup_create", map[string]any{
		"source_paths": sourcePaths,
		"databases":    databases,
		"output":       filePath,
	})
	if err != nil {
		logger.Error().Err(err).Msg("scheduled backup failed")
		bs.BackupSvc.UpdateStatus(b.ID, "failed", 0)
		return
	}

	var sizeBytes int64
	if result, ok := resp.Result.(map[string]any); ok {
		if sz, ok := result["size_bytes"].(float64); ok {
			sizeBytes = int64(sz)
		}
	}
	bs.BackupSvc.UpdateStatus(b.ID, "completed", sizeBytes)

	// Mark schedule as run
	bs.ScheduleSvc.MarkRun(sc.ID, sc.Frequency, sc.Time)

	// Enforce retention: delete oldest backups beyond retention_count
	bs.enforceRetention(sc)

	logger.Info().Str("file", fileName).Msg("scheduled backup completed")
}

func (bs *BackupScheduler) enforceRetention(sc Schedule) {
	// Get all completed backups for this scope, ordered newest first
	query := `SELECT id, file_path FROM backups WHERE status = 'completed' AND file_path LIKE '%scheduled-%'`
	var args []any
	if sc.DomainID != nil {
		query += ` AND domain_id = ?`
		args = append(args, *sc.DomainID)
	} else {
		query += ` AND domain_id IS NULL AND type = 'full'`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := bs.DomainDB.Query(query, args...)
	if err != nil {
		return
	}
	defer rows.Close()

	var idx int
	for rows.Next() {
		var id int64
		var filePath string
		rows.Scan(&id, &filePath)
		idx++
		if idx > sc.RetentionCount {
			// Delete this backup
			bs.BackupSvc.Delete(id)
			bs.AgentClient.Call("file_delete", map[string]any{"path": filePath, "recursive": false})
			log.Info().Int64("backup_id", id).Msg("deleted old backup (retention policy)")
		}
	}
}
