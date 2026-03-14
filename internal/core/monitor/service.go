package monitor

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
)

type SystemSnapshot struct {
	ID          int64   `json:"id"`
	CPUUsage    float64 `json:"cpu_usage"`
	RAMUsed     uint64  `json:"ram_used"`
	RAMTotal    uint64  `json:"ram_total"`
	RAMPercent  float64 `json:"ram_percent"`
	LoadAvg1    float64 `json:"load_avg_1"`
	LoadAvg5    float64 `json:"load_avg_5"`
	LoadAvg15   float64 `json:"load_avg_15"`
	CollectedAt string  `json:"collected_at"`
}

type DomainSnapshot struct {
	ID             int64  `json:"id"`
	DomainID       int64  `json:"domain_id"`
	DiskUsageBytes int64  `json:"disk_usage_bytes"`
	BandwidthBytes int64  `json:"bandwidth_bytes"`
	CollectedAt    string `json:"collected_at"`
}

type Service struct {
	DB          *sql.DB
	AgentClient *agent.Client
	DomainSvc   *domain.Service
	stop        chan struct{}
}

// Start launches background metric collection goroutines.
func (s *Service) Start() {
	s.stop = make(chan struct{})

	// System metrics every 5 minutes
	go func() {
		// Collect immediately on startup
		s.collectSystem()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.collectSystem()
			case <-s.stop:
				return
			}
		}
	}()

	// Domain metrics every hour
	go func() {
		// Collect after a short delay on startup
		time.Sleep(30 * time.Second)
		s.collectDomains()
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.collectDomains()
			case <-s.stop:
				return
			}
		}
	}()

	// Cleanup old data once per day
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.cleanup()
			case <-s.stop:
				return
			}
		}
	}()

	log.Info().Msg("monitor service started (system: 5m, domains: 1h)")
}

// Stop halts all collection goroutines.
func (s *Service) Stop() {
	close(s.stop)
}

// GetSystemHistory returns system metrics for the last N hours.
func (s *Service) GetSystemHistory(hours int) ([]SystemSnapshot, error) {
	if hours <= 0 || hours > 720 {
		hours = 24
	}
	rows, err := s.DB.Query(
		`SELECT id, cpu_usage, ram_used, ram_total, load_avg_1, load_avg_5, load_avg_15, collected_at
		 FROM system_metrics
		 WHERE collected_at >= datetime('now', ?)
		 ORDER BY collected_at ASC`,
		fmt.Sprintf("-%d hours", hours),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []SystemSnapshot
	for rows.Next() {
		var s SystemSnapshot
		if err := rows.Scan(&s.ID, &s.CPUUsage, &s.RAMUsed, &s.RAMTotal, &s.LoadAvg1, &s.LoadAvg5, &s.LoadAvg15, &s.CollectedAt); err != nil {
			return nil, err
		}
		if s.RAMTotal > 0 {
			s.RAMPercent = float64(s.RAMUsed) / float64(s.RAMTotal) * 100
		}
		snapshots = append(snapshots, s)
	}
	return snapshots, rows.Err()
}

// GetSystemCurrent returns the most recent system snapshot.
func (s *Service) GetSystemCurrent() (*SystemSnapshot, error) {
	var snap SystemSnapshot
	err := s.DB.QueryRow(
		`SELECT id, cpu_usage, ram_used, ram_total, load_avg_1, load_avg_5, load_avg_15, collected_at
		 FROM system_metrics ORDER BY id DESC LIMIT 1`,
	).Scan(&snap.ID, &snap.CPUUsage, &snap.RAMUsed, &snap.RAMTotal, &snap.LoadAvg1, &snap.LoadAvg5, &snap.LoadAvg15, &snap.CollectedAt)
	if err != nil {
		return nil, err
	}
	if snap.RAMTotal > 0 {
		snap.RAMPercent = float64(snap.RAMUsed) / float64(snap.RAMTotal) * 100
	}
	return &snap, nil
}

// GetDomainMetrics returns metrics history for a domain.
func (s *Service) GetDomainMetrics(domainID int64, hours int) ([]DomainSnapshot, error) {
	if hours <= 0 || hours > 720 {
		hours = 24
	}
	rows, err := s.DB.Query(
		`SELECT id, domain_id, disk_usage_bytes, bandwidth_bytes, collected_at
		 FROM domain_metrics
		 WHERE domain_id = ? AND collected_at >= datetime('now', ?)
		 ORDER BY collected_at ASC`,
		domainID, fmt.Sprintf("-%d hours", hours),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []DomainSnapshot
	for rows.Next() {
		var d DomainSnapshot
		if err := rows.Scan(&d.ID, &d.DomainID, &d.DiskUsageBytes, &d.BandwidthBytes, &d.CollectedAt); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, d)
	}
	return snapshots, rows.Err()
}

// GetDomainLatest returns the most recent snapshot for a domain.
func (s *Service) GetDomainLatest(domainID int64) (*DomainSnapshot, error) {
	var d DomainSnapshot
	err := s.DB.QueryRow(
		`SELECT id, domain_id, disk_usage_bytes, bandwidth_bytes, collected_at
		 FROM domain_metrics WHERE domain_id = ? ORDER BY id DESC LIMIT 1`, domainID,
	).Scan(&d.ID, &d.DomainID, &d.DiskUsageBytes, &d.BandwidthBytes, &d.CollectedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// GetRecentCPUHistory returns the last N CPU usage values (for sparkline).
func (s *Service) GetRecentCPUHistory(count int) ([]float64, error) {
	rows, err := s.DB.Query(
		`SELECT cpu_usage FROM (
			SELECT cpu_usage, id FROM system_metrics ORDER BY id DESC LIMIT ?
		) sub ORDER BY id ASC`, count,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values []float64
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, rows.Err()
}

// GetRecentRAMHistory returns the last N RAM percent values (for sparkline).
func (s *Service) GetRecentRAMHistory(count int) ([]float64, error) {
	rows, err := s.DB.Query(
		`SELECT ram_used, ram_total FROM (
			SELECT ram_used, ram_total, id FROM system_metrics ORDER BY id DESC LIMIT ?
		) sub ORDER BY id ASC`, count,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values []float64
	for rows.Next() {
		var used, total uint64
		if err := rows.Scan(&used, &total); err != nil {
			return nil, err
		}
		if total > 0 {
			values = append(values, float64(used)/float64(total)*100)
		} else {
			values = append(values, 0)
		}
	}
	return values, rows.Err()
}

func (s *Service) collectSystem() {
	resp, err := s.AgentClient.Call("system_info", nil)
	if err != nil {
		log.Debug().Err(err).Msg("monitor: failed to get system_info")
		return
	}

	data, ok := resp.Result.(map[string]any)
	if !ok {
		return
	}

	cpuUsage, _ := data["cpu_usage"].(float64)

	var ramUsed, ramTotal uint64
	if ram, ok := data["ram"].(map[string]any); ok {
		if v, ok := ram["used"].(float64); ok {
			ramUsed = uint64(v)
		}
		if v, ok := ram["total"].(float64); ok {
			ramTotal = uint64(v)
		}
	}

	var loadAvg1, loadAvg5, loadAvg15 float64
	if la, ok := data["load_avg"].(string); ok {
		fields := strings.Fields(la)
		if len(fields) >= 3 {
			loadAvg1, _ = strconv.ParseFloat(fields[0], 64)
			loadAvg5, _ = strconv.ParseFloat(fields[1], 64)
			loadAvg15, _ = strconv.ParseFloat(fields[2], 64)
		}
	}

	_, err = s.DB.Exec(
		`INSERT INTO system_metrics (cpu_usage, ram_used, ram_total, load_avg_1, load_avg_5, load_avg_15)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		cpuUsage, ramUsed, ramTotal, loadAvg1, loadAvg5, loadAvg15,
	)
	if err != nil {
		log.Error().Err(err).Msg("monitor: failed to store system metrics")
	}
}

func (s *Service) collectDomains() {
	domains, _, err := s.DomainSvc.List("", "", 1, 1000, 0)
	if err != nil {
		log.Error().Err(err).Msg("monitor: failed to list domains")
		return
	}

	for _, dom := range domains {
		var diskBytes int64
		var bwBytes int64

		// Disk usage via agent
		resp, err := s.AgentClient.Call("domain_disk_usage", map[string]any{"path": dom.DocumentRoot})
		if err == nil {
			if result, ok := resp.Result.(map[string]any); ok {
				if v, ok := result["bytes"].(float64); ok {
					diskBytes = int64(v)
				}
			}
		}

		// Bandwidth from nginx log
		logPath := fmt.Sprintf("/var/log/nginx/%s-access.log", dom.Name)
		resp, err = s.AgentClient.Call("domain_bandwidth", map[string]any{"log_path": logPath})
		if err == nil {
			if result, ok := resp.Result.(map[string]any); ok {
				if v, ok := result["bytes"].(float64); ok {
					bwBytes = int64(v)
				}
			}
		}

		if _, err := s.DB.Exec(
			`INSERT INTO domain_metrics (domain_id, disk_usage_bytes, bandwidth_bytes) VALUES (?, ?, ?)`,
			dom.ID, diskBytes, bwBytes,
		); err != nil {
			log.Error().Err(err).Int64("domain_id", dom.ID).Msg("monitor: failed to store domain metrics")
		}
	}
}

func (s *Service) cleanup() {
	res1, _ := s.DB.Exec(`DELETE FROM system_metrics WHERE collected_at < datetime('now', '-30 days')`)
	res2, _ := s.DB.Exec(`DELETE FROM domain_metrics WHERE collected_at < datetime('now', '-90 days')`)
	n1, _ := res1.RowsAffected()
	n2, _ := res2.RowsAffected()
	if n1 > 0 || n2 > 0 {
		log.Info().Int64("system", n1).Int64("domain", n2).Msg("monitor: cleaned old metrics")
	}
}

// MarshalSparklineJSON returns cpu_history and ram_history as JSON arrays.
// Used by the WebSocket hub to augment the broadcast payload.
func (s *Service) MarshalSparklineJSON() (cpuJSON, ramJSON json.RawMessage) {
	cpuHist, err := s.GetRecentCPUHistory(12)
	if err != nil || len(cpuHist) == 0 {
		cpuJSON = []byte("[]")
	} else {
		cpuJSON, _ = json.Marshal(cpuHist)
	}

	ramHist, err := s.GetRecentRAMHistory(12)
	if err != nil || len(ramHist) == 0 {
		ramJSON = []byte("[]")
	} else {
		ramJSON, _ = json.Marshal(ramHist)
	}
	return
}
