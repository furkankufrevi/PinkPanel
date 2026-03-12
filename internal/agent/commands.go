package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// CommandFunc is a function that executes an agent command.
type CommandFunc func(params json.RawMessage) (interface{}, error)

// CommandRegistry holds the allowlist of agent commands.
type CommandRegistry struct {
	commands map[string]CommandFunc
}

// NewCommandRegistry creates a registry with all allowed commands.
func NewCommandRegistry() *CommandRegistry {
	r := &CommandRegistry{
		commands: make(map[string]CommandFunc),
	}
	r.registerBuiltins()
	return r
}

// Execute runs a command by name.
func (r *CommandRegistry) Execute(method string, params json.RawMessage) (interface{}, error) {
	cmd, ok := r.commands[method]
	if !ok {
		return nil, fmt.Errorf("unknown method: %s", method)
	}
	return cmd(params)
}

func (r *CommandRegistry) registerBuiltins() {
	// Core
	r.commands["ping"] = cmdPing
	r.commands["system_info"] = cmdSystemInfo

	// Service management
	r.commands["service_status"] = cmdServiceStatus
	r.commands["service_control"] = cmdServiceControl

	// File operations
	r.commands["file_write"] = cmdFileWrite
	r.commands["file_read"] = cmdFileRead
	r.commands["file_delete"] = cmdFileDelete
	r.commands["file_list"] = cmdFileList
	r.commands["file_rename"] = cmdFileRename
	r.commands["file_copy"] = cmdFileCopy
	r.commands["file_extract"] = cmdFileExtract
	r.commands["file_compress"] = cmdFileCompress
	r.commands["file_search"] = cmdFileSearch
	r.commands["dir_create"] = cmdDirCreate
	r.commands["set_ownership"] = cmdSetOwnership
	r.commands["set_permissions"] = cmdSetPermissions

	// NGINX
	r.commands["nginx_test"] = cmdNginxTest
	r.commands["nginx_reload"] = cmdNginxReload

	// DNS
	r.commands["dns_setup"] = cmdDNSSetup
	r.commands["dns_write_zone"] = cmdDNSWriteZone
	r.commands["dns_add_zone"] = cmdDNSAddZone
	r.commands["dns_remove_zone"] = cmdDNSRemoveZone
	r.commands["dns_reload"] = cmdDNSReload

	// PHP
	r.commands["php_list_versions"] = cmdPHPListVersions
	r.commands["php_write_pool"] = cmdPHPWritePool
	r.commands["php_reload"] = cmdPHPReload
	r.commands["php_info"] = cmdPHPInfo

	// SSL
	r.commands["ssl_write_cert"] = cmdSSLWriteCert
	r.commands["ssl_delete_cert"] = cmdSSLDeleteCert

	// MySQL
	r.commands["mysql_create_db"] = cmdMySQLCreateDB
	r.commands["mysql_drop_db"] = cmdMySQLDropDB
	r.commands["mysql_create_user"] = cmdMySQLCreateUser
	r.commands["mysql_drop_user"] = cmdMySQLDropUser
	r.commands["mysql_grant"] = cmdMySQLGrant
	r.commands["mysql_dump"] = cmdMySQLDump
	r.commands["mysql_restore"] = cmdMySQLRestore
	r.commands["mysql_db_size"] = cmdMySQLDBSize

	// FTP
	r.commands["ftp_create_user"] = cmdFTPCreateUser
	r.commands["ftp_delete_user"] = cmdFTPDeleteUser
	r.commands["ftp_reload"] = cmdFTPReload

	// Backup
	r.commands["backup_create"] = cmdBackupCreate
	r.commands["backup_restore"] = cmdBackupRestore
	r.commands["backup_delete"] = cmdBackupDelete

	// Logs
	r.commands["log_read"] = cmdLogRead

	// System users
	r.commands["user_create"] = cmdUserCreate
	r.commands["user_delete"] = cmdUserDelete
}

// ---------- Param types ----------

type serviceStatusParams struct {
	Service string `json:"service"`
}

type serviceControlParams struct {
	Service string `json:"service"`
	Action  string `json:"action"`
}

type fileWriteParams struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    string `json:"mode"`
}

type fileReadParams struct {
	Path string `json:"path"`
}

type fileDeleteParams struct {
	Path      string `json:"path"`
	Recursive bool   `json:"recursive"`
}

type fileListParams struct {
	Path string `json:"path"`
}

type fileRenameParams struct {
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
}

type fileCopyParams struct {
	Source string `json:"source"`
	Dest   string `json:"dest"`
}

type fileExtractParams struct {
	Archive string `json:"archive"`
	Dest    string `json:"dest"`
}

type fileCompressParams struct {
	Sources []string `json:"sources"`
	Output  string   `json:"output"`
	Format  string   `json:"format"`
}

type fileSearchParams struct {
	Path       string `json:"path"`
	Query      string `json:"query"`
	MaxResults int    `json:"max_results"`
}

type searchResult struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Snippet string `json:"snippet"`
}

type dirCreateParams struct {
	Path  string `json:"path"`
	Owner string `json:"owner"`
	Group string `json:"group"`
}

type ownershipParams struct {
	Path      string `json:"path"`
	Owner     string `json:"owner"`
	Group     string `json:"group"`
	Recursive bool   `json:"recursive"`
}

type permissionsParams struct {
	Path      string `json:"path"`
	Mode      string `json:"mode"`
	Recursive bool   `json:"recursive"`
}

type zoneWriteParams struct {
	Domain  string `json:"domain"`
	Content string `json:"content"`
}

type phpPoolWriteParams struct {
	Version string `json:"version"`
	Domain  string `json:"domain"`
	Content string `json:"content"`
}

type phpReloadParams struct {
	Version string `json:"version"`
}

type sslWriteParams struct {
	Domain string `json:"domain"`
	Cert   string `json:"cert"`
	Key    string `json:"key"`
	Chain  string `json:"chain"`
}

type sslDeleteParams struct {
	Domain string `json:"domain"`
}

type mysqlDBParams struct {
	Name string `json:"name"`
}

type mysqlUserParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
}

type mysqlGrantParams struct {
	Username    string `json:"username"`
	Host        string `json:"host"`
	Database    string `json:"database"`
	Permissions string `json:"permissions"`
}

type mysqlDumpParams struct {
	Database string `json:"database"`
	Output   string `json:"output"`
}

type mysqlRestoreParams struct {
	Database string `json:"database"`
	Input    string `json:"input"`
}

type ftpUserParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
	HomeDir  string `json:"home_dir"`
}

type ftpDeleteParams struct {
	Username string `json:"username"`
}

type backupCreateParams struct {
	SourcePaths []string `json:"source_paths"`
	Databases   []string `json:"databases"`
	Output      string   `json:"output"`
}

type backupRestoreParams struct {
	Archive string `json:"archive"`
	Dest    string `json:"dest"`
}

type backupDeleteParams struct {
	Path string `json:"path"`
}

type logReadParams struct {
	Path   string `json:"path"`
	Lines  int    `json:"lines"`
	Filter string `json:"filter"`
}

// ---------- Result types ----------

type serviceStatusResult struct {
	Service string `json:"service"`
	Active  bool   `json:"active"`
	Status  string `json:"status"`
}

type systemInfoResult struct {
	OS        string        `json:"os"`
	Arch      string        `json:"arch"`
	Hostname  string        `json:"hostname"`
	CPUUsage  float64       `json:"cpu_usage"`
	RAM       ramInfo       `json:"ram"`
	Disk      []diskInfo    `json:"disk"`
	Uptime    string        `json:"uptime"`
	LoadAvg   string        `json:"load_avg"`
}

type ramInfo struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
}

type diskInfo struct {
	Mount      string `json:"mount"`
	Filesystem string `json:"filesystem"`
	Total      string `json:"total"`
	Used       string `json:"used"`
	Available  string `json:"available"`
	UsePercent string `json:"use_percent"`
}

type fileInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	IsDir       bool   `json:"is_dir"`
	Size        int64  `json:"size"`
	Permissions string `json:"permissions"`
	Owner       string `json:"owner"`
	Group       string `json:"group"`
	ModTime     string `json:"mod_time"`
}

// ---------- Validators ----------

// allowedServices is the strict allowlist of services the agent can manage.
var allowedServices = map[string]bool{
	"nginx":    true,
	"apache2":  true,
	"mariadb":  true,
	"mysql":    true,
	"postfix":  true,
	"dovecot":  true,
	"named":    true,
	"bind9":    true,
	"vsftpd":   true,
	"redis":    true,
	"fail2ban": true,
}

var allowedServiceActions = map[string]bool{
	"start":   true,
	"stop":    true,
	"restart": true,
	"reload":  true,
	"status":  true,
}

// safeNameRe validates identifiers (service names, db names, usernames).
var safeNameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// allowedPHPVersionRe validates PHP version strings like "8.3", "8.2".
var allowedPHPVersionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+$`)

func validateServiceName(name string) error {
	if name == "" {
		return fmt.Errorf("service name is required")
	}
	// Allow php-fpm version variants like php8.3-fpm
	if strings.HasPrefix(name, "php") && strings.HasSuffix(name, "-fpm") {
		if safeNameRe.MatchString(name) {
			return nil
		}
	}
	if !allowedServices[name] {
		return fmt.Errorf("service not in allowlist: %s", name)
	}
	return nil
}

// validatePath ensures the path is absolute and doesn't contain traversal.
func validatePath(p string) error {
	if p == "" {
		return fmt.Errorf("path is required")
	}
	if !filepath.IsAbs(p) {
		return fmt.Errorf("path must be absolute: %s", p)
	}
	cleaned := filepath.Clean(p)
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("path traversal not allowed: %s", p)
	}
	return nil
}

func unsupportedOS() (interface{}, error) {
	return map[string]string{"status": "unsupported_os"}, nil
}

// ---------- Core commands ----------

func cmdPing(_ json.RawMessage) (interface{}, error) {
	return "pong", nil
}

func cmdSystemInfo(_ json.RawMessage) (interface{}, error) {
	hostname, _ := exec.Command("hostname").Output()

	result := systemInfoResult{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Hostname: strings.TrimSpace(string(hostname)),
	}

	if runtime.GOOS != "linux" {
		return result, nil
	}

	// CPU usage from /proc/stat snapshot
	result.CPUUsage = getCPUUsage()

	// RAM from /proc/meminfo
	result.RAM = getRAMInfo()

	// Disk usage
	result.Disk = getDiskInfo()

	// Uptime
	if out, err := exec.Command("cat", "/proc/uptime").Output(); err == nil {
		fields := strings.Fields(string(out))
		if len(fields) > 0 {
			if secs, err := strconv.ParseFloat(fields[0], 64); err == nil {
				d := time.Duration(secs) * time.Second
				result.Uptime = d.String()
			}
		}
	}

	// Load average
	if out, err := os.ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(string(out))
		if len(fields) >= 3 {
			result.LoadAvg = strings.Join(fields[:3], " ")
		}
	}

	return result, nil
}

func getCPUUsage() float64 {
	data1, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	idle1, total1 := parseCPUStat(string(data1))

	time.Sleep(200 * time.Millisecond)

	data2, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	idle2, total2 := parseCPUStat(string(data2))

	idleDelta := float64(idle2 - idle1)
	totalDelta := float64(total2 - total1)
	if totalDelta == 0 {
		return 0
	}
	return (1.0 - idleDelta/totalDelta) * 100.0
}

func parseCPUStat(data string) (idle, total uint64) {
	for _, line := range strings.Split(data, "\n") {
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				return
			}
			for i := 1; i < len(fields); i++ {
				v, _ := strconv.ParseUint(fields[i], 10, 64)
				total += v
				if i == 4 { // idle is the 4th value (0-indexed field 4)
					idle = v
				}
			}
			return
		}
	}
	return
}

func getRAMInfo() ramInfo {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return ramInfo{}
	}
	info := ramInfo{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.ParseUint(fields[1], 10, 64)
		val *= 1024 // kB to bytes
		switch fields[0] {
		case "MemTotal:":
			info.Total = val
		case "MemAvailable:":
			info.Free = val
		}
	}
	info.Used = info.Total - info.Free
	return info
}

func getDiskInfo() []diskInfo {
	out, err := exec.Command("df", "-h", "--output=target,source,size,used,avail,pcent").Output()
	if err != nil {
		return nil
	}
	var disks []diskInfo
	lines := strings.Split(string(out), "\n")
	for _, line := range lines[1:] { // skip header
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		// Only include real filesystems
		if !strings.HasPrefix(fields[1], "/dev/") {
			continue
		}
		disks = append(disks, diskInfo{
			Mount:      fields[0],
			Filesystem: fields[1],
			Total:      fields[2],
			Used:       fields[3],
			Available:  fields[4],
			UsePercent: fields[5],
		})
	}
	return disks
}

// ---------- Service commands ----------

func cmdServiceStatus(params json.RawMessage) (interface{}, error) {
	var p serviceStatusParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validateServiceName(p.Service); err != nil {
		return nil, err
	}
	if runtime.GOOS != "linux" {
		return serviceStatusResult{Service: p.Service, Active: false, Status: "unsupported_os"}, nil
	}
	out, err := exec.Command("systemctl", "is-active", p.Service).Output()
	status := strings.TrimSpace(string(out))
	if err != nil {
		return serviceStatusResult{Service: p.Service, Active: false, Status: status}, nil
	}
	return serviceStatusResult{Service: p.Service, Active: status == "active", Status: status}, nil
}

func cmdServiceControl(params json.RawMessage) (interface{}, error) {
	var p serviceControlParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validateServiceName(p.Service); err != nil {
		return nil, err
	}
	if !allowedServiceActions[p.Action] {
		return nil, fmt.Errorf("action not allowed: %s", p.Action)
	}
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}
	out, err := exec.Command("systemctl", p.Action, p.Service).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("systemctl %s %s: %s", p.Action, p.Service, strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

// ---------- File commands ----------

func cmdFileWrite(params json.RawMessage) (interface{}, error) {
	var p fileWriteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.Path); err != nil {
		return nil, err
	}
	mode := os.FileMode(0644)
	if p.Mode != "" {
		m, err := strconv.ParseUint(p.Mode, 8, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid mode: %s", p.Mode)
		}
		mode = os.FileMode(m)
	}
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(p.Path), 0755); err != nil {
		return nil, fmt.Errorf("creating parent directory: %w", err)
	}
	if err := os.WriteFile(p.Path, []byte(p.Content), mode); err != nil {
		return nil, fmt.Errorf("writing file: %w", err)
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdFileRead(params json.RawMessage) (interface{}, error) {
	var p fileReadParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.Path); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p.Path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	return map[string]string{"content": string(data)}, nil
}

func cmdFileDelete(params json.RawMessage) (interface{}, error) {
	var p fileDeleteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.Path); err != nil {
		return nil, err
	}
	if p.Recursive {
		if err := os.RemoveAll(p.Path); err != nil {
			return nil, fmt.Errorf("removing path: %w", err)
		}
	} else {
		if err := os.Remove(p.Path); err != nil {
			return nil, fmt.Errorf("removing file: %w", err)
		}
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdFileList(params json.RawMessage) (interface{}, error) {
	var p fileListParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.Path); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(p.Path)
	if err != nil {
		return nil, fmt.Errorf("listing directory: %w", err)
	}
	var files []fileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{
			Name:        entry.Name(),
			Path:        filepath.Join(p.Path, entry.Name()),
			IsDir:       entry.IsDir(),
			Size:        info.Size(),
			Permissions: info.Mode().String(),
			ModTime:     info.ModTime().Format(time.RFC3339),
		})
	}
	return files, nil
}

func cmdFileRename(params json.RawMessage) (interface{}, error) {
	var p fileRenameParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.OldPath); err != nil {
		return nil, err
	}
	if err := validatePath(p.NewPath); err != nil {
		return nil, err
	}
	if err := os.Rename(p.OldPath, p.NewPath); err != nil {
		return nil, fmt.Errorf("renaming: %w", err)
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdFileCopy(params json.RawMessage) (interface{}, error) {
	var p fileCopyParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.Source); err != nil {
		return nil, err
	}
	if err := validatePath(p.Dest); err != nil {
		return nil, err
	}
	out, err := exec.Command("cp", "-a", p.Source, p.Dest).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("copying: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdFileExtract(params json.RawMessage) (interface{}, error) {
	var p fileExtractParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.Archive); err != nil {
		return nil, err
	}
	if err := validatePath(p.Dest); err != nil {
		return nil, err
	}
	var cmd *exec.Cmd
	switch {
	case strings.HasSuffix(p.Archive, ".tar.gz") || strings.HasSuffix(p.Archive, ".tgz"):
		cmd = exec.Command("tar", "xzf", p.Archive, "-C", p.Dest)
	case strings.HasSuffix(p.Archive, ".tar.bz2"):
		cmd = exec.Command("tar", "xjf", p.Archive, "-C", p.Dest)
	case strings.HasSuffix(p.Archive, ".tar"):
		cmd = exec.Command("tar", "xf", p.Archive, "-C", p.Dest)
	case strings.HasSuffix(p.Archive, ".zip"):
		cmd = exec.Command("unzip", "-o", p.Archive, "-d", p.Dest)
	default:
		return nil, fmt.Errorf("unsupported archive format: %s", filepath.Ext(p.Archive))
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("extracting: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdFileCompress(params json.RawMessage) (interface{}, error) {
	var p fileCompressParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.Output); err != nil {
		return nil, err
	}
	for _, src := range p.Sources {
		if err := validatePath(src); err != nil {
			return nil, err
		}
	}

	var cmd *exec.Cmd
	switch p.Format {
	case "zip":
		args := append([]string{"-r", p.Output}, p.Sources...)
		cmd = exec.Command("zip", args...)
	case "tar.gz":
		args := append([]string{"czf", p.Output}, p.Sources...)
		cmd = exec.Command("tar", args...)
	case "tar.bz2":
		args := append([]string{"cjf", p.Output}, p.Sources...)
		cmd = exec.Command("tar", args...)
	default:
		return nil, fmt.Errorf("unsupported format: %s", p.Format)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("compressing: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdFileSearch(params json.RawMessage) (interface{}, error) {
	var p fileSearchParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.Path); err != nil {
		return nil, err
	}
	if p.Query == "" {
		return nil, fmt.Errorf("query is required")
	}
	maxResults := p.MaxResults
	if maxResults <= 0 {
		maxResults = 100
	}

	cmd := exec.Command("grep", "-rn", "--include=*", "-m", "1", p.Query, p.Path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// grep returns exit code 1 when no matches found
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []searchResult{}, nil
		}
		return nil, fmt.Errorf("searching: %s", strings.TrimSpace(string(out)))
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var results []searchResult
	for _, line := range lines {
		if line == "" {
			continue
		}
		if len(results) >= maxResults {
			break
		}
		// Format: file:line:content
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		lineNum, _ := strconv.Atoi(parts[1])
		snippet := parts[2]
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		results = append(results, searchResult{
			Path:    parts[0],
			Line:    lineNum,
			Snippet: snippet,
		})
	}
	return results, nil
}

func cmdDirCreate(params json.RawMessage) (interface{}, error) {
	var p dirCreateParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.Path); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(p.Path, 0755); err != nil {
		return nil, fmt.Errorf("creating directory: %w", err)
	}
	if p.Owner != "" && runtime.GOOS == "linux" {
		group := p.Group
		if group == "" {
			group = p.Owner
		}
		out, err := exec.Command("chown", "-R", p.Owner+":"+group, p.Path).CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("setting ownership: %s", strings.TrimSpace(string(out)))
		}
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdSetOwnership(params json.RawMessage) (interface{}, error) {
	var p ownershipParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.Path); err != nil {
		return nil, err
	}
	if !safeNameRe.MatchString(p.Owner) {
		return nil, fmt.Errorf("invalid owner: %s", p.Owner)
	}
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}
	group := p.Group
	if group == "" {
		group = p.Owner
	}
	if !safeNameRe.MatchString(group) {
		return nil, fmt.Errorf("invalid group: %s", group)
	}
	args := []string{p.Owner + ":" + group, p.Path}
	if p.Recursive {
		args = append([]string{"-R"}, args...)
	}
	out, err := exec.Command("chown", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("chown: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdSetPermissions(params json.RawMessage) (interface{}, error) {
	var p permissionsParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.Path); err != nil {
		return nil, err
	}
	if !safeNameRe.MatchString(p.Mode) {
		return nil, fmt.Errorf("invalid mode: %s", p.Mode)
	}
	args := []string{p.Mode, p.Path}
	if p.Recursive {
		args = append([]string{"-R"}, args...)
	}
	out, err := exec.Command("chmod", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("chmod: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

// ---------- NGINX commands ----------

func cmdNginxTest(_ json.RawMessage) (interface{}, error) {
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}
	out, err := exec.Command("nginx", "-t").CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		return map[string]interface{}{"valid": false, "output": output}, nil
	}
	return map[string]interface{}{"valid": true, "output": output}, nil
}

func cmdNginxReload(_ json.RawMessage) (interface{}, error) {
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}
	out, err := exec.Command("systemctl", "reload", "nginx").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("nginx reload: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

// ---------- DNS commands ----------

func cmdDNSSetup(_ json.RawMessage) (interface{}, error) {
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}

	optionsConf := `options {
    directory "/var/cache/bind";
    listen-on { any; };
    listen-on-v6 { any; };
    allow-query { any; };
    recursion no;
    allow-recursion { none; };
    dnssec-validation auto;
    version "not disclosed";
};
`
	if err := writeFileSync("/etc/bind/named.conf.options", []byte(optionsConf), 0644); err != nil {
		return nil, fmt.Errorf("writing named.conf.options: %w", err)
	}

	// Ensure zones directory exists with bind-readable permissions
	if err := os.MkdirAll("/etc/bind/zones", 0755); err != nil {
		return nil, fmt.Errorf("creating zones dir: %w", err)
	}

	// Ensure named.conf.local exists
	localPath := "/etc/bind/named.conf.local"
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		if err := writeFileSync(localPath, []byte("// PinkPanel managed zones\n"), 0644); err != nil {
			return nil, fmt.Errorf("creating named.conf.local: %w", err)
		}
	}

	// Generate rndc key if missing
	if _, err := os.Stat("/etc/bind/rndc.key"); os.IsNotExist(err) {
		exec.Command("rndc-confgen", "-a", "-b", "256").CombinedOutput()
	}

	// Ensure main named.conf includes named.conf.local
	namedConf := "/etc/bind/named.conf"
	if data, err := os.ReadFile(namedConf); err == nil {
		if !strings.Contains(string(data), "named.conf.local") {
			f, err := os.OpenFile(namedConf, os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				f.WriteString("\ninclude \"/etc/bind/named.conf.local\";\n")
				f.Sync()
				f.Close()
			}
		}
	}

	// Set ownership so bind can read everything
	exec.Command("chown", "-R", "bind:bind", "/etc/bind/zones").CombinedOutput()

	// Restart BIND
	restartBIND()

	return map[string]string{"status": "ok"}, nil
}

func cmdDNSWriteZone(params json.RawMessage) (interface{}, error) {
	var p zoneWriteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Domain) {
		return nil, fmt.Errorf("invalid domain name: %s", p.Domain)
	}

	if err := os.MkdirAll("/etc/bind/zones", 0755); err != nil {
		return nil, fmt.Errorf("creating zones directory: %w", err)
	}

	zonePath := fmt.Sprintf("/etc/bind/zones/db.%s", p.Domain)

	// Validate zone content before writing to final path by using a temp file
	if checkBin, err := exec.LookPath("named-checkzone"); err == nil {
		tmpPath := zonePath + ".tmp"
		if err := writeFileSync(tmpPath, []byte(p.Content), 0644); err != nil {
			return nil, fmt.Errorf("writing temp zone file: %w", err)
		}
		out, err := exec.Command(checkBin, p.Domain, tmpPath).CombinedOutput()
		os.Remove(tmpPath)
		if err != nil {
			return nil, fmt.Errorf("zone file validation failed for %s: %s", p.Domain, strings.TrimSpace(string(out)))
		}
	}

	// Write final zone file
	if err := writeFileSync(zonePath, []byte(p.Content), 0644); err != nil {
		return nil, fmt.Errorf("writing zone file: %w", err)
	}

	// Set ownership so bind user can read it
	exec.Command("chown", "bind:bind", zonePath).CombinedOutput()

	return map[string]string{"status": "ok", "path": zonePath}, nil
}

func cmdDNSAddZone(params json.RawMessage) (interface{}, error) {
	var p zoneWriteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Domain) {
		return nil, fmt.Errorf("invalid domain name: %s", p.Domain)
	}

	namedConfLocal := "/etc/bind/named.conf.local"
	existing, _ := os.ReadFile(namedConfLocal)

	// Check if zone already exists — match exact zone block pattern
	zoneMarker := fmt.Sprintf("zone \"%s\" {", p.Domain)
	if strings.Contains(string(existing), zoneMarker) {
		return map[string]string{"status": "ok", "detail": "zone already registered"}, nil
	}

	// Append zone block
	zoneBlock := fmt.Sprintf("\nzone \"%s\" {\n    type master;\n    file \"/etc/bind/zones/db.%s\";\n    allow-transfer { none; };\n};\n", p.Domain, p.Domain)

	f, err := os.OpenFile(namedConfLocal, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening named.conf.local: %w", err)
	}
	if _, err := f.WriteString(zoneBlock); err != nil {
		f.Close()
		return nil, fmt.Errorf("writing zone block: %w", err)
	}
	f.Sync()
	f.Close()

	return map[string]string{"status": "ok"}, nil
}

func cmdDNSRemoveZone(params json.RawMessage) (interface{}, error) {
	var p zoneWriteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Domain) {
		return nil, fmt.Errorf("invalid domain name: %s", p.Domain)
	}

	namedConfLocal := "/etc/bind/named.conf.local"
	data, err := os.ReadFile(namedConfLocal)
	if err != nil {
		return nil, fmt.Errorf("reading named.conf.local: %w", err)
	}

	// Remove the zone block using regex for robust matching.
	// Uses .*? (non-greedy with (?s) dotall) to handle nested braces like
	// allow-transfer { none; }; inside the zone block.
	escaped := regexp.QuoteMeta(p.Domain)
	zoneRe := regexp.MustCompile(`(?s)\n*zone\s+"` + escaped + `"\s*\{.*?\n\};\s*`)
	cleaned := zoneRe.ReplaceAllString(string(data), "\n")

	if cleaned != string(data) {
		if err := writeFileSync(namedConfLocal, []byte(cleaned), 0644); err != nil {
			return nil, fmt.Errorf("writing named.conf.local: %w", err)
		}
	}

	// Remove zone file only after config was successfully updated
	zonePath := fmt.Sprintf("/etc/bind/zones/db.%s", p.Domain)
	os.Remove(zonePath)

	return map[string]string{"status": "ok"}, nil
}

func cmdDNSReload(_ json.RawMessage) (interface{}, error) {
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}

	// Validate BIND configuration before attempting reload
	if checkBin, err := exec.LookPath("named-checkconf"); err == nil {
		if out, err := exec.Command(checkBin).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("BIND config validation failed: %s", strings.TrimSpace(string(out)))
		}
	}

	// Try rndc first (fastest, no downtime)
	if out, err := exec.Command("rndc", "reconfig").CombinedOutput(); err == nil {
		// reconfig succeeded — also reload zone data
		if out2, err := exec.Command("rndc", "reload").CombinedOutput(); err != nil {
			return nil, fmt.Errorf("rndc reload failed: %s", strings.TrimSpace(string(out2)))
		}
		return map[string]string{"status": "ok", "method": "rndc"}, nil
	} else {
		// Log rndc failure details before falling back
		fmt.Fprintf(os.Stderr, "rndc reconfig failed (%v): %s, falling back to restart\n", err, strings.TrimSpace(string(out)))
	}

	// Fallback: full restart
	if err := restartBIND(); err != nil {
		return nil, err
	}
	return map[string]string{"status": "ok", "method": "restart"}, nil
}

// restartBIND tries named then bind9 service names.
// Clears any "failed" state first so systemd allows the restart.
func restartBIND() error {
	// Clear failed state so systemd allows restart after repeated crashes
	exec.Command("systemctl", "reset-failed", "named").Run()
	exec.Command("systemctl", "reset-failed", "bind9").Run()

	out, err := exec.Command("systemctl", "restart", "named").CombinedOutput()
	if err == nil {
		return nil
	}
	out2, err2 := exec.Command("systemctl", "restart", "bind9").CombinedOutput()
	if err2 == nil {
		return nil
	}
	return fmt.Errorf("restart bind failed: named: %s, bind9: %s",
		strings.TrimSpace(string(out)), strings.TrimSpace(string(out2)))
}

// writeFileSync writes a file and ensures it's flushed to disk.
func writeFileSync(path string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// ---------- PHP commands ----------

func cmdPHPListVersions(_ json.RawMessage) (interface{}, error) {
	if runtime.GOOS != "linux" {
		return []string{"8.3"}, nil // dev fallback
	}
	// Find installed PHP-FPM versions by scanning /etc/php/
	entries, err := os.ReadDir("/etc/php")
	if err != nil {
		return []string{}, nil
	}
	var versions []string
	for _, e := range entries {
		if e.IsDir() && allowedPHPVersionRe.MatchString(e.Name()) {
			// Verify FPM is actually installed
			fpmPath := fmt.Sprintf("/etc/php/%s/fpm", e.Name())
			if _, err := os.Stat(fpmPath); err == nil {
				versions = append(versions, e.Name())
			}
		}
	}
	return versions, nil
}

func cmdPHPWritePool(params json.RawMessage) (interface{}, error) {
	var p phpPoolWriteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !allowedPHPVersionRe.MatchString(p.Version) {
		return nil, fmt.Errorf("invalid PHP version: %s", p.Version)
	}
	if !safeNameRe.MatchString(p.Domain) {
		return nil, fmt.Errorf("invalid domain: %s", p.Domain)
	}
	poolDir := fmt.Sprintf("/etc/php/%s/fpm/pool.d", p.Version)
	if err := os.MkdirAll(poolDir, 0755); err != nil {
		return nil, fmt.Errorf("creating pool directory: %w", err)
	}
	poolPath := filepath.Join(poolDir, p.Domain+".conf")
	if err := os.WriteFile(poolPath, []byte(p.Content), 0644); err != nil {
		return nil, fmt.Errorf("writing pool config: %w", err)
	}
	return map[string]string{"status": "ok", "path": poolPath}, nil
}

func cmdPHPReload(params json.RawMessage) (interface{}, error) {
	var p phpReloadParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !allowedPHPVersionRe.MatchString(p.Version) {
		return nil, fmt.Errorf("invalid PHP version: %s", p.Version)
	}
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}
	service := fmt.Sprintf("php%s-fpm", p.Version)
	out, err := exec.Command("systemctl", "reload", service).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("php-fpm reload: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdPHPInfo(params json.RawMessage) (interface{}, error) {
	var p struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !allowedPHPVersionRe.MatchString(p.Version) {
		return nil, fmt.Errorf("invalid PHP version: %s", p.Version)
	}

	phpBin := fmt.Sprintf("php%s", p.Version)
	out, err := exec.Command(phpBin, "-i").CombinedOutput()
	if err != nil {
		// Fallback to generic php
		out, err = exec.Command("php", "-i").CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("php -i failed: %s", strings.TrimSpace(string(out)))
		}
	}

	// Also get loaded modules
	modOut, _ := exec.Command(phpBin, "-m").CombinedOutput()
	if len(modOut) == 0 {
		modOut, _ = exec.Command("php", "-m").CombinedOutput()
	}

	return map[string]string{
		"info":       string(out),
		"extensions": string(modOut),
	}, nil
}

// ---------- SSL commands ----------

func cmdSSLWriteCert(params json.RawMessage) (interface{}, error) {
	var p sslWriteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Domain) {
		return nil, fmt.Errorf("invalid domain: %s", p.Domain)
	}
	sslDir := fmt.Sprintf("/usr/local/pinkpanel/data/ssl/%s", p.Domain)
	if err := os.MkdirAll(sslDir, 0700); err != nil {
		return nil, fmt.Errorf("creating ssl directory: %w", err)
	}
	certPath := filepath.Join(sslDir, "cert.pem")
	keyPath := filepath.Join(sslDir, "key.pem")
	if err := os.WriteFile(certPath, []byte(p.Cert), 0644); err != nil {
		return nil, fmt.Errorf("writing cert: %w", err)
	}
	if err := os.WriteFile(keyPath, []byte(p.Key), 0600); err != nil {
		return nil, fmt.Errorf("writing key: %w", err)
	}
	result := map[string]string{
		"status":    "ok",
		"cert_path": certPath,
		"key_path":  keyPath,
	}
	if p.Chain != "" {
		chainPath := filepath.Join(sslDir, "chain.pem")
		if err := os.WriteFile(chainPath, []byte(p.Chain), 0644); err != nil {
			return nil, fmt.Errorf("writing chain: %w", err)
		}
		result["chain_path"] = chainPath
	}
	return result, nil
}

func cmdSSLDeleteCert(params json.RawMessage) (interface{}, error) {
	var p sslDeleteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Domain) {
		return nil, fmt.Errorf("invalid domain: %s", p.Domain)
	}
	sslDir := fmt.Sprintf("/usr/local/pinkpanel/data/ssl/%s", p.Domain)
	if err := os.RemoveAll(sslDir); err != nil {
		return nil, fmt.Errorf("removing ssl directory: %w", err)
	}
	return map[string]string{"status": "ok"}, nil
}

// ---------- MySQL commands ----------

// mysqlDefaultsFile is the legacy path to the MySQL credentials file.
// New installs use unix_socket auth and do not create this file.
const mysqlDefaultsFile = "/etc/pinkpanel/mysql.cnf"

// mysqlArgs prepends authentication flags for mysql/mysqldump commands.
// If the legacy password file exists, it uses --defaults-file for backward
// compatibility. Otherwise it relies on unix_socket auth (agent runs as root).
func mysqlArgs(args ...string) []string {
	if _, err := os.Stat(mysqlDefaultsFile); err == nil {
		return append([]string{"--defaults-file=" + mysqlDefaultsFile}, args...)
	}
	return append([]string{"-u", "root"}, args...)
}

func cmdMySQLCreateDB(params json.RawMessage) (interface{}, error) {
	var p mysqlDBParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Name) {
		return nil, fmt.Errorf("invalid database name: %s", p.Name)
	}
	stmt := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", p.Name)
	out, err := exec.Command("mysql", mysqlArgs("-e", stmt)...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("create database: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdMySQLDropDB(params json.RawMessage) (interface{}, error) {
	var p mysqlDBParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Name) {
		return nil, fmt.Errorf("invalid database name: %s", p.Name)
	}
	stmt := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", p.Name)
	out, err := exec.Command("mysql", mysqlArgs("-e", stmt)...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("drop database: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdMySQLCreateUser(params json.RawMessage) (interface{}, error) {
	var p mysqlUserParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Username) {
		return nil, fmt.Errorf("invalid username: %s", p.Username)
	}
	host := p.Host
	if host == "" {
		host = "localhost"
	}
	if !safeNameRe.MatchString(host) {
		return nil, fmt.Errorf("invalid host: %s", host)
	}
	// Use parameterized approach — write to temp file to avoid password in command line
	stmt := fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%s' IDENTIFIED BY '%s'; FLUSH PRIVILEGES;", p.Username, host, escapeMySQLString(p.Password))
	out, err := exec.Command("mysql", mysqlArgs("-e", stmt)...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("create user: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdMySQLDropUser(params json.RawMessage) (interface{}, error) {
	var p mysqlUserParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Username) {
		return nil, fmt.Errorf("invalid username: %s", p.Username)
	}
	host := p.Host
	if host == "" {
		host = "localhost"
	}
	stmt := fmt.Sprintf("DROP USER IF EXISTS '%s'@'%s'; FLUSH PRIVILEGES;", p.Username, host)
	out, err := exec.Command("mysql", mysqlArgs("-e", stmt)...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("drop user: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdMySQLGrant(params json.RawMessage) (interface{}, error) {
	var p mysqlGrantParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Username) || !safeNameRe.MatchString(p.Database) {
		return nil, fmt.Errorf("invalid username or database name")
	}
	host := p.Host
	if host == "" {
		host = "localhost"
	}
	perms := p.Permissions
	if perms == "" {
		perms = "ALL PRIVILEGES"
	}
	// Validate permissions string
	allowedPerms := map[string]bool{
		"ALL PRIVILEGES": true, "ALL": true,
		"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true,
		"CREATE": true, "DROP": true, "ALTER": true, "INDEX": true,
	}
	for _, perm := range strings.Split(perms, ",") {
		perm = strings.TrimSpace(strings.ToUpper(perm))
		if !allowedPerms[perm] {
			return nil, fmt.Errorf("permission not allowed: %s", perm)
		}
	}
	stmt := fmt.Sprintf("GRANT %s ON `%s`.* TO '%s'@'%s'; FLUSH PRIVILEGES;", perms, p.Database, p.Username, host)
	out, err := exec.Command("mysql", mysqlArgs("-e", stmt)...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("grant: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdMySQLDump(params json.RawMessage) (interface{}, error) {
	var p mysqlDumpParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Database) {
		return nil, fmt.Errorf("invalid database name: %s", p.Database)
	}
	if err := validatePath(p.Output); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(p.Output), 0755); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}
	outFile, err := os.Create(p.Output)
	if err != nil {
		return nil, fmt.Errorf("creating output file: %w", err)
	}
	defer outFile.Close()
	cmd := exec.Command("mysqldump", mysqlArgs("--single-transaction", "--routines", "--triggers", p.Database)...)
	cmd.Stdout = outFile
	if err := cmd.Run(); err != nil {
		os.Remove(p.Output)
		return nil, fmt.Errorf("mysqldump: %w", err)
	}
	stat, _ := outFile.Stat()
	return map[string]interface{}{"status": "ok", "path": p.Output, "size": stat.Size()}, nil
}

func cmdMySQLRestore(params json.RawMessage) (interface{}, error) {
	var p mysqlRestoreParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Database) {
		return nil, fmt.Errorf("invalid database name: %s", p.Database)
	}
	if err := validatePath(p.Input); err != nil {
		return nil, err
	}
	inFile, err := os.Open(p.Input)
	if err != nil {
		return nil, fmt.Errorf("opening input file: %w", err)
	}
	defer inFile.Close()
	cmd := exec.Command("mysql", mysqlArgs(p.Database)...)
	cmd.Stdin = inFile
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("mysql restore: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdMySQLDBSize(params json.RawMessage) (interface{}, error) {
	var p mysqlDBParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Name) {
		return nil, fmt.Errorf("invalid database name: %s", p.Name)
	}
	stmt := fmt.Sprintf("SELECT COALESCE(SUM(data_length + index_length), 0) AS size FROM information_schema.tables WHERE table_schema = '%s';", p.Name)
	out, err := exec.Command("mysql", mysqlArgs("-N", "-e", stmt)...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("db size: %s", strings.TrimSpace(string(out)))
	}
	size, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	return map[string]interface{}{"name": p.Name, "size_bytes": size}, nil
}

// ---------- FTP commands ----------

func cmdFTPCreateUser(params json.RawMessage) (interface{}, error) {
	var p ftpUserParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Username) {
		return nil, fmt.Errorf("invalid username: %s", p.Username)
	}
	if err := validatePath(p.HomeDir); err != nil {
		return nil, err
	}
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}
	// Create system user with no shell, limited to FTP
	out, err := exec.Command("useradd", "-m", "-d", p.HomeDir, "-s", "/usr/sbin/nologin", p.Username).CombinedOutput()
	if err != nil {
		// User might already exist
		if !strings.Contains(string(out), "already exists") {
			return nil, fmt.Errorf("useradd: %s", strings.TrimSpace(string(out)))
		}
	}
	// Set password
	cmd := exec.Command("chpasswd")
	cmd.Stdin = strings.NewReader(fmt.Sprintf("%s:%s", p.Username, p.Password))
	out, err = cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("chpasswd: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdFTPDeleteUser(params json.RawMessage) (interface{}, error) {
	var p ftpDeleteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Username) {
		return nil, fmt.Errorf("invalid username: %s", p.Username)
	}
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}
	out, err := exec.Command("userdel", p.Username).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("userdel: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdFTPReload(_ json.RawMessage) (interface{}, error) {
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}
	out, err := exec.Command("systemctl", "restart", "vsftpd").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("vsftpd restart: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

// ---------- Backup commands ----------

func cmdBackupCreate(params json.RawMessage) (interface{}, error) {
	var p backupCreateParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.Output); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(p.Output), 0755); err != nil {
		return nil, fmt.Errorf("creating backup directory: %w", err)
	}

	// Create a temporary directory to stage the backup
	tmpDir, err := os.MkdirTemp("", "pinkpanel-backup-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Dump databases
	for _, db := range p.Databases {
		if !safeNameRe.MatchString(db) {
			return nil, fmt.Errorf("invalid database name: %s", db)
		}
		dumpPath := filepath.Join(tmpDir, "databases", db+".sql")
		os.MkdirAll(filepath.Join(tmpDir, "databases"), 0755)
		outFile, err := os.Create(dumpPath)
		if err != nil {
			return nil, fmt.Errorf("creating dump file: %w", err)
		}
		cmd := exec.Command("mysqldump", "--single-transaction", "--routines", "--triggers", db)
		cmd.Stdout = outFile
		if err := cmd.Run(); err != nil {
			outFile.Close()
			return nil, fmt.Errorf("mysqldump %s: %w", db, err)
		}
		outFile.Close()
	}

	// Copy source paths into staging area
	for _, src := range p.SourcePaths {
		if err := validatePath(src); err != nil {
			return nil, err
		}
		destDir := filepath.Join(tmpDir, "files")
		os.MkdirAll(destDir, 0755)
		out, err := exec.Command("cp", "-a", src, destDir+"/").CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("copying %s: %s", src, strings.TrimSpace(string(out)))
		}
	}

	// Tar + gzip the staging directory
	out, err := exec.Command("tar", "czf", p.Output, "-C", tmpDir, ".").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("creating archive: %s", strings.TrimSpace(string(out)))
	}

	stat, _ := os.Stat(p.Output)
	size := int64(0)
	if stat != nil {
		size = stat.Size()
	}
	return map[string]interface{}{"status": "ok", "path": p.Output, "size_bytes": size}, nil
}

func cmdBackupRestore(params json.RawMessage) (interface{}, error) {
	var p backupRestoreParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.Archive); err != nil {
		return nil, err
	}
	if err := validatePath(p.Dest); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(p.Dest, 0755); err != nil {
		return nil, fmt.Errorf("creating dest dir: %w", err)
	}
	out, err := exec.Command("tar", "xzf", p.Archive, "-C", p.Dest).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("extracting backup: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdBackupDelete(params json.RawMessage) (interface{}, error) {
	var p backupDeleteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.Path); err != nil {
		return nil, err
	}
	if err := os.Remove(p.Path); err != nil {
		return nil, fmt.Errorf("deleting backup: %w", err)
	}
	return map[string]string{"status": "ok"}, nil
}

// ---------- Log commands ----------

func cmdLogRead(params json.RawMessage) (interface{}, error) {
	var p logReadParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.Path); err != nil {
		return nil, err
	}
	lines := p.Lines
	if lines <= 0 {
		lines = 100
	}
	if lines > 10000 {
		lines = 10000
	}

	args := []string{"-n", strconv.Itoa(lines), p.Path}
	out, err := exec.Command("tail", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("reading log: %s", strings.TrimSpace(string(out)))
	}

	content := string(out)
	if p.Filter != "" {
		var filtered []string
		for _, line := range strings.Split(content, "\n") {
			if strings.Contains(line, p.Filter) {
				filtered = append(filtered, line)
			}
		}
		content = strings.Join(filtered, "\n")
	}

	return map[string]string{"content": content}, nil
}

// ---------- System User Commands ----------

type userCreateParams struct {
	Username string `json:"username"`
	HomeDir  string `json:"home_dir"` // e.g. /home/pp_username
}

func cmdUserCreate(params json.RawMessage) (interface{}, error) {
	var p userCreateParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if p.HomeDir == "" {
		p.HomeDir = "/home/" + p.Username
	}

	// Validate username (alphanumeric + underscore, starts with letter)
	validUser := regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
	if !validUser.MatchString(p.Username) {
		return nil, fmt.Errorf("invalid system username: %s", p.Username)
	}

	// Create system user with home directory, no login shell
	out, err := exec.Command("useradd",
		"--create-home",
		"--home-dir", p.HomeDir,
		"--shell", "/usr/sbin/nologin",
		"--system",
		p.Username,
	).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("useradd failed: %s", strings.TrimSpace(string(out)))
	}

	// Create domains directory under home
	domainsDir := filepath.Join(p.HomeDir, "domains")
	if err := os.MkdirAll(domainsDir, 0755); err != nil {
		return nil, fmt.Errorf("creating domains directory: %w", err)
	}

	// Set ownership
	out, err = exec.Command("chown", "-R", p.Username+":"+p.Username, p.HomeDir).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("chown failed: %s", strings.TrimSpace(string(out)))
	}

	return map[string]string{"status": "created", "username": p.Username, "home_dir": p.HomeDir}, nil
}

type userDeleteParams struct {
	Username   string `json:"username"`
	RemoveHome bool   `json:"remove_home"`
}

func cmdUserDelete(params json.RawMessage) (interface{}, error) {
	var p userDeleteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.Username == "" {
		return nil, fmt.Errorf("username is required")
	}

	// Safety: never delete root, www-data, or other system users
	protected := map[string]bool{"root": true, "www-data": true, "nobody": true, "mysql": true, "postfix": true, "dovecot": true}
	if protected[p.Username] {
		return nil, fmt.Errorf("cannot delete protected system user: %s", p.Username)
	}

	args := []string{p.Username}
	if p.RemoveHome {
		args = append([]string{"--remove"}, args...)
	}

	out, err := exec.Command("userdel", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("userdel failed: %s", strings.TrimSpace(string(out)))
	}

	return map[string]string{"status": "deleted", "username": p.Username}, nil
}

// ---------- Helpers ----------

func escapeMySQLString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}
