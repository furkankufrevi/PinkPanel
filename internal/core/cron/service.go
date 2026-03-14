package cron

import (
	"database/sql"
	"fmt"
	"strings"
)

type CronJob struct {
	ID          int64  `json:"id"`
	DomainID    int64  `json:"domain_id"`
	Schedule    string `json:"schedule"`
	Command     string `json:"command"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CronLog struct {
	ID         int64  `json:"id"`
	CronJobID  int64  `json:"cron_job_id"`
	ExitCode   int    `json:"exit_code"`
	Output     string `json:"output"`
	DurationMs int64  `json:"duration_ms"`
	StartedAt  string `json:"started_at"`
}

type Service struct {
	DB *sql.DB
}

func (s *Service) List(domainID int64) ([]CronJob, error) {
	rows, err := s.DB.Query(
		`SELECT id, domain_id, schedule, command, description, enabled, created_at, updated_at
		 FROM cron_jobs WHERE domain_id = ? ORDER BY id`, domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []CronJob
	for rows.Next() {
		var j CronJob
		if err := rows.Scan(&j.ID, &j.DomainID, &j.Schedule, &j.Command, &j.Description, &j.Enabled, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (s *Service) GetByID(id int64) (*CronJob, error) {
	var j CronJob
	err := s.DB.QueryRow(
		`SELECT id, domain_id, schedule, command, description, enabled, created_at, updated_at
		 FROM cron_jobs WHERE id = ?`, id,
	).Scan(&j.ID, &j.DomainID, &j.Schedule, &j.Command, &j.Description, &j.Enabled, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("cron job not found")
	}
	return &j, nil
}

func (s *Service) Create(domainID int64, schedule, command, description string) (*CronJob, error) {
	if err := validateSchedule(schedule); err != nil {
		return nil, err
	}
	if strings.TrimSpace(command) == "" {
		return nil, fmt.Errorf("command is required")
	}

	res, err := s.DB.Exec(
		`INSERT INTO cron_jobs (domain_id, schedule, command, description) VALUES (?, ?, ?, ?)`,
		domainID, schedule, command, description,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cron job: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetByID(id)
}

func (s *Service) Update(id int64, schedule, command, description *string, enabled *bool) (*CronJob, error) {
	sets := []string{}
	args := []any{}

	if schedule != nil {
		if err := validateSchedule(*schedule); err != nil {
			return nil, err
		}
		sets = append(sets, "schedule = ?")
		args = append(args, *schedule)
	}
	if command != nil {
		if strings.TrimSpace(*command) == "" {
			return nil, fmt.Errorf("command cannot be empty")
		}
		sets = append(sets, "command = ?")
		args = append(args, *command)
	}
	if description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *description)
	}
	if enabled != nil {
		sets = append(sets, "enabled = ?")
		if *enabled {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}

	if len(sets) == 0 {
		return s.GetByID(id)
	}

	sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	_, err := s.DB.Exec(
		fmt.Sprintf("UPDATE cron_jobs SET %s WHERE id = ?", strings.Join(sets, ", ")),
		args...,
	)
	if err != nil {
		return nil, err
	}
	return s.GetByID(id)
}

func (s *Service) Delete(id int64) (*CronJob, error) {
	j, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if _, err := s.DB.Exec(`DELETE FROM cron_jobs WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return j, nil
}

// ListEnabled returns all enabled cron jobs for a domain (used for crontab sync).
func (s *Service) ListEnabled(domainID int64) ([]CronJob, error) {
	rows, err := s.DB.Query(
		`SELECT id, domain_id, schedule, command, description, enabled, created_at, updated_at
		 FROM cron_jobs WHERE domain_id = ? AND enabled = 1 ORDER BY id`, domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []CronJob
	for rows.Next() {
		var j CronJob
		if err := rows.Scan(&j.ID, &j.DomainID, &j.Schedule, &j.Command, &j.Description, &j.Enabled, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (s *Service) ListLogs(jobID int64, limit int) ([]CronLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.DB.Query(
		`SELECT id, cron_job_id, exit_code, output, duration_ms, started_at
		 FROM cron_logs WHERE cron_job_id = ? ORDER BY id DESC LIMIT ?`, jobID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []CronLog
	for rows.Next() {
		var l CronLog
		if err := rows.Scan(&l.ID, &l.CronJobID, &l.ExitCode, &l.Output, &l.DurationMs, &l.StartedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (s *Service) CreateLog(jobID int64, exitCode int, output string, durationMs int64) error {
	_, err := s.DB.Exec(
		`INSERT INTO cron_logs (cron_job_id, exit_code, output, duration_ms) VALUES (?, ?, ?, ?)`,
		jobID, exitCode, output, durationMs,
	)
	return err
}

// validateSchedule checks that a cron schedule has exactly 5 space-separated fields.
func validateSchedule(schedule string) error {
	fields := strings.Fields(schedule)
	if len(fields) != 5 {
		return fmt.Errorf("invalid cron schedule: must have exactly 5 fields (minute hour day month weekday)")
	}
	return nil
}
