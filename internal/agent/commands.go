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
	r.commands["file_symlink"] = cmdFileSymlink

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

	// System upgrade
	r.commands["system_upgrade"] = cmdSystemUpgrade
	r.commands["system_version"] = cmdSystemVersion

	// Fail2ban
	r.commands["fail2ban_status"] = cmdFail2banStatus
	r.commands["fail2ban_jail_status"] = cmdFail2banJailStatus
	r.commands["fail2ban_banned_ips"] = cmdFail2banBannedIPs
	r.commands["fail2ban_ban_ip"] = cmdFail2banBanIP
	r.commands["fail2ban_unban_ip"] = cmdFail2banUnbanIP

	// Email (Postfix + Dovecot)
	r.commands["email_create_account"] = cmdEmailCreateAccount
	r.commands["email_delete_account"] = cmdEmailDeleteAccount
	r.commands["email_change_password"] = cmdEmailChangePassword
	r.commands["email_reload"] = cmdEmailReload
	r.commands["email_update_virtual_maps"] = cmdEmailUpdateVirtualMaps
	r.commands["email_queue_list"] = cmdEmailQueueList
	r.commands["email_queue_flush"] = cmdEmailQueueFlush
	r.commands["email_queue_delete"] = cmdEmailQueueDelete
	r.commands["email_generate_dkim"] = cmdEmailGenerateDKIM

	// SpamAssassin & ClamAV
	r.commands["spam_configure"] = cmdSpamConfigure
	r.commands["spam_status"] = cmdSpamStatus
	r.commands["clamav_configure"] = cmdClamAVConfigure
	r.commands["clamav_status"] = cmdClamAVStatus

	// Mail autodiscovery
	r.commands["email_write_autoconfig"] = cmdEmailWriteAutoconfig

	// Mail SSL
	r.commands["email_configure_ssl"] = cmdEmailConfigureSSL
	r.commands["email_ssl_status"] = cmdEmailSSLStatus

	// Git
	r.commands["git_clone"] = cmdGitClone
	r.commands["git_pull"] = cmdGitPull
	r.commands["git_init_bare"] = cmdGitInitBare
	r.commands["git_deploy"] = cmdGitDeploy
	r.commands["git_log"] = cmdGitLog
	r.commands["git_setup_hook"] = cmdGitSetupHook
	r.commands["git_ssh_key"] = cmdGitSSHKey

	// Cron
	r.commands["cron_sync"] = cmdCronSync
	r.commands["cron_execute"] = cmdCronExecute

	// Monitoring
	r.commands["domain_disk_usage"] = cmdDomainDiskUsage
	r.commands["domain_bandwidth"] = cmdDomainBandwidth

	// App installer
	r.commands["app_download"] = cmdAppDownload
	r.commands["app_wpcli"] = cmdAppWPCLI
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

type fileSymlinkParams struct {
	Target string `json:"target"` // existing file
	Link   string `json:"link"`   // symlink path to create
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

func cmdFileSymlink(params json.RawMessage) (interface{}, error) {
	var p fileSymlinkParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.Target); err != nil {
		return nil, err
	}
	if err := validatePath(p.Link); err != nil {
		return nil, err
	}
	// Remove existing file/symlink at link path
	os.Remove(p.Link)
	if err := os.Symlink(p.Target, p.Link); err != nil {
		return nil, fmt.Errorf("creating symlink: %w", err)
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

// ---------- System Upgrade ----------

func cmdSystemVersion(params json.RawMessage) (interface{}, error) {
	// Read version from file
	version := "unknown"
	data, err := os.ReadFile("/etc/pinkpanel/version")
	if err == nil {
		version = strings.TrimSpace(string(data))
	}
	return map[string]string{"version": version}, nil
}

func cmdSystemUpgrade(params json.RawMessage) (interface{}, error) {
	// Run the upgrade script in the background
	// The script is fetched from GitHub and run as root
	upgradeScript := "/opt/pinkpanel/scripts/upgrade.sh"

	// Check if local script exists, otherwise download
	if _, err := os.Stat(upgradeScript); os.IsNotExist(err) {
		// Download from GitHub
		dlCmd := exec.Command("bash", "-c",
			`curl -fsSL https://raw.githubusercontent.com/furkankufrevi/PinkPanel/master/scripts/upgrade.sh -o /tmp/pinkpanel-upgrade.sh && chmod +x /tmp/pinkpanel-upgrade.sh`)
		if out, err := dlCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("failed to download upgrade script: %s", strings.TrimSpace(string(out)))
		}
		upgradeScript = "/tmp/pinkpanel-upgrade.sh"
	}

	// Run upgrade script, capturing output
	logFile := fmt.Sprintf("/var/log/pinkpanel/upgrade-%s.log", time.Now().Format("20060102-150405"))
	cmd := exec.Command("bash", "-c", fmt.Sprintf("bash %s > %s 2>&1", upgradeScript, logFile))
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start upgrade: %v", err)
	}

	// Wait in background goroutine
	go func() {
		cmd.Wait()
	}()

	return map[string]string{
		"status":   "started",
		"log_file": logFile,
	}, nil
}

// ---------- Fail2ban ----------

func cmdFail2banStatus(params json.RawMessage) (interface{}, error) {
	out, err := exec.Command("fail2ban-client", "status").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("fail2ban-client status failed: %s", strings.TrimSpace(string(out)))
	}

	output := string(out)
	result := map[string]interface{}{
		"raw": strings.TrimSpace(output),
	}

	// Parse jail list
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Jail list:") {
			jailStr := strings.TrimPrefix(line, "Jail list:")
			jailStr = strings.TrimSpace(jailStr)
			jails := []string{}
			for _, j := range strings.Split(jailStr, ",") {
				j = strings.TrimSpace(j)
				if j != "" {
					jails = append(jails, j)
				}
			}
			result["jails"] = jails
		}
	}

	return result, nil
}

type fail2banJailParams struct {
	Jail string `json:"jail"`
}

func cmdFail2banJailStatus(params json.RawMessage) (interface{}, error) {
	var p fail2banJailParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.Jail == "" {
		return nil, fmt.Errorf("jail name is required")
	}
	if !safeNameRe.MatchString(p.Jail) {
		return nil, fmt.Errorf("invalid jail name: %s", p.Jail)
	}

	out, err := exec.Command("fail2ban-client", "status", p.Jail).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("fail2ban-client status %s failed: %s", p.Jail, strings.TrimSpace(string(out)))
	}

	output := strings.TrimSpace(string(out))
	result := map[string]interface{}{
		"jail": p.Jail,
		"raw":  output,
	}

	// Parse key metrics
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Currently failed:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				val, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
				result["currently_failed"] = val
			}
		}
		if strings.Contains(line, "Total failed:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				val, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
				result["total_failed"] = val
			}
		}
		if strings.Contains(line, "Currently banned:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				val, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
				result["currently_banned"] = val
			}
		}
		if strings.Contains(line, "Total banned:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				val, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
				result["total_banned"] = val
			}
		}
		if strings.Contains(line, "Banned IP list:") {
			parts := strings.SplitN(line, ":", 2)
			ips := []string{}
			if len(parts) == 2 {
				for _, ip := range strings.Fields(strings.TrimSpace(parts[1])) {
					if ip != "" {
						ips = append(ips, ip)
					}
				}
			}
			result["banned_ips"] = ips
		}
	}

	return result, nil
}

func cmdFail2banBannedIPs(params json.RawMessage) (interface{}, error) {
	var p fail2banJailParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	jail := p.Jail
	if jail == "" {
		jail = "pinkpanel"
	}
	if !safeNameRe.MatchString(jail) {
		return nil, fmt.Errorf("invalid jail name: %s", jail)
	}

	out, err := exec.Command("fail2ban-client", "status", jail).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("fail2ban-client status %s failed: %s", jail, strings.TrimSpace(string(out)))
	}

	ips := []string{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Banned IP list:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				for _, ip := range strings.Fields(strings.TrimSpace(parts[1])) {
					if ip != "" {
						ips = append(ips, ip)
					}
				}
			}
		}
	}

	return map[string]interface{}{"jail": jail, "banned_ips": ips}, nil
}

// ipRe validates IPv4 and IPv6 addresses loosely.
var ipRe = regexp.MustCompile(`^[0-9a-fA-F.:]+$`)

type fail2banIPParams struct {
	Jail string `json:"jail"`
	IP   string `json:"ip"`
}

func cmdFail2banBanIP(params json.RawMessage) (interface{}, error) {
	var p fail2banIPParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.IP == "" {
		return nil, fmt.Errorf("ip is required")
	}
	if !ipRe.MatchString(p.IP) {
		return nil, fmt.Errorf("invalid IP address: %s", p.IP)
	}
	jail := p.Jail
	if jail == "" {
		jail = "pinkpanel"
	}
	if !safeNameRe.MatchString(jail) {
		return nil, fmt.Errorf("invalid jail name: %s", jail)
	}

	out, err := exec.Command("fail2ban-client", "set", jail, "banip", p.IP).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("fail2ban ban failed: %s", strings.TrimSpace(string(out)))
	}

	return map[string]string{"status": "banned", "ip": p.IP, "jail": jail}, nil
}

func cmdFail2banUnbanIP(params json.RawMessage) (interface{}, error) {
	var p fail2banIPParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.IP == "" {
		return nil, fmt.Errorf("ip is required")
	}
	if !ipRe.MatchString(p.IP) {
		return nil, fmt.Errorf("invalid IP address: %s", p.IP)
	}
	jail := p.Jail
	if jail == "" {
		jail = "pinkpanel"
	}
	if !safeNameRe.MatchString(jail) {
		return nil, fmt.Errorf("invalid jail name: %s", jail)
	}

	out, err := exec.Command("fail2ban-client", "set", jail, "unbanip", p.IP).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("fail2ban unban failed: %s", strings.TrimSpace(string(out)))
	}

	return map[string]string{"status": "unbanned", "ip": p.IP, "jail": jail}, nil
}

// ---------- Email (Postfix + Dovecot) commands ----------

const (
	dovecotUsersFile        = "/etc/dovecot/users"
	postfixVirtualDomains   = "/etc/postfix/virtual-mailbox-domains"
	postfixVirtualMailboxes = "/etc/postfix/virtual-mailbox-maps"
	postfixVirtualAliases   = "/etc/postfix/virtual"
	vmailDir                = "/var/mail/vhosts"
)

type emailAccountParams struct {
	Domain   string `json:"domain"`
	Address  string `json:"address"`
	Password string `json:"password"`
	QuotaMB  int64  `json:"quota_mb"`
}

type emailDeleteAccountParams struct {
	Domain  string `json:"domain"`
	Address string `json:"address"`
}

type emailChangePasswordParams struct {
	Domain   string `json:"domain"`
	Address  string `json:"address"`
	Password string `json:"password"`
}

type emailVirtualMapsParams struct {
	// Domains is a list of domain names that have email enabled
	Domains []string `json:"domains"`
	// Mailboxes maps "user@domain" -> "domain/user/"
	Mailboxes map[string]string `json:"mailboxes"`
	// Aliases maps "source@domain" -> "dest@other.com"
	Aliases map[string]string `json:"aliases"`
}

type emailQueueDeleteParams struct {
	QueueID string `json:"queue_id"`
}

type emailDKIMParams struct {
	Domain string `json:"domain"`
}

func cmdEmailCreateAccount(params json.RawMessage) (interface{}, error) {
	var p emailAccountParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Domain) {
		return nil, fmt.Errorf("invalid domain: %s", p.Domain)
	}
	if !safeNameRe.MatchString(p.Address) {
		return nil, fmt.Errorf("invalid address: %s", p.Address)
	}
	if p.Password == "" {
		return nil, fmt.Errorf("password is required")
	}
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}

	// Hash password using doveadm
	hashOut, err := exec.Command("doveadm", "pw", "-s", "SHA512-CRYPT", "-p", p.Password).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("doveadm pw: %s", strings.TrimSpace(string(hashOut)))
	}
	passHash := strings.TrimSpace(string(hashOut))

	fullAddr := fmt.Sprintf("%s@%s", p.Address, p.Domain)
	maildir := fmt.Sprintf("%s/%s/%s/Maildir", vmailDir, p.Domain, p.Address)

	// Create maildir
	if err := os.MkdirAll(maildir, 0700); err != nil {
		return nil, fmt.Errorf("creating maildir: %w", err)
	}
	// Set ownership to vmail (uid/gid 5000)
	exec.Command("chown", "-R", "vmail:vmail", fmt.Sprintf("%s/%s/%s", vmailDir, p.Domain, p.Address)).Run()

	// Add to dovecot users file: user@domain:{hash}::5000:5000::/var/mail/vhosts/domain/user::userdb_quota_rule=*:storage=QuotaMB
	userLine := fmt.Sprintf("%s:%s::5000:5000::%s/%s/%s", fullAddr, passHash, vmailDir, p.Domain, p.Address)
	if p.QuotaMB > 0 {
		userLine += fmt.Sprintf("::userdb_quota_rule=*:storage=%dM", p.QuotaMB)
	}

	// Read existing file, remove old entry if any, append new
	existing, _ := os.ReadFile(dovecotUsersFile)
	lines := strings.Split(string(existing), "\n")
	var newLines []string
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, fullAddr+":") {
			continue
		}
		newLines = append(newLines, line)
	}
	newLines = append(newLines, userLine)
	if err := os.WriteFile(dovecotUsersFile, []byte(strings.Join(newLines, "\n")+"\n"), 0640); err != nil {
		return nil, fmt.Errorf("writing dovecot users: %w", err)
	}
	exec.Command("chown", "dovecot:dovecot", dovecotUsersFile).Run()

	return map[string]string{"status": "ok", "address": fullAddr}, nil
}

func cmdEmailDeleteAccount(params json.RawMessage) (interface{}, error) {
	var p emailDeleteAccountParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Domain) || !safeNameRe.MatchString(p.Address) {
		return nil, fmt.Errorf("invalid domain or address")
	}
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}

	fullAddr := fmt.Sprintf("%s@%s", p.Address, p.Domain)

	// Remove from dovecot users file
	existing, _ := os.ReadFile(dovecotUsersFile)
	lines := strings.Split(string(existing), "\n")
	var newLines []string
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, fullAddr+":") {
			continue
		}
		newLines = append(newLines, line)
	}
	os.WriteFile(dovecotUsersFile, []byte(strings.Join(newLines, "\n")+"\n"), 0640)

	// Remove maildir
	maildirPath := fmt.Sprintf("%s/%s/%s", vmailDir, p.Domain, p.Address)
	os.RemoveAll(maildirPath)

	return map[string]string{"status": "ok", "address": fullAddr}, nil
}

func cmdEmailChangePassword(params json.RawMessage) (interface{}, error) {
	var p emailChangePasswordParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Domain) || !safeNameRe.MatchString(p.Address) {
		return nil, fmt.Errorf("invalid domain or address")
	}
	if p.Password == "" {
		return nil, fmt.Errorf("password is required")
	}
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}

	// Hash new password
	hashOut, err := exec.Command("doveadm", "pw", "-s", "SHA512-CRYPT", "-p", p.Password).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("doveadm pw: %s", strings.TrimSpace(string(hashOut)))
	}
	passHash := strings.TrimSpace(string(hashOut))
	fullAddr := fmt.Sprintf("%s@%s", p.Address, p.Domain)

	// Read file and replace the password for this user
	existing, _ := os.ReadFile(dovecotUsersFile)
	lines := strings.Split(string(existing), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, fullAddr+":") {
			parts := strings.SplitN(line, ":", 3)
			if len(parts) >= 3 {
				lines[i] = fullAddr + ":" + passHash + ":" + parts[2]
			} else {
				lines[i] = fullAddr + ":" + passHash + "::5000:5000::" + vmailDir + "/" + p.Domain + "/" + p.Address
			}
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("email account not found in dovecot users file")
	}

	os.WriteFile(dovecotUsersFile, []byte(strings.Join(lines, "\n")), 0640)
	return map[string]string{"status": "ok", "address": fullAddr}, nil
}

func cmdEmailReload(_ json.RawMessage) (interface{}, error) {
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}
	exec.Command("systemctl", "reload", "postfix").CombinedOutput()
	exec.Command("systemctl", "reload", "dovecot").CombinedOutput()
	return map[string]string{"status": "ok"}, nil
}

func cmdEmailUpdateVirtualMaps(params json.RawMessage) (interface{}, error) {
	var p emailVirtualMapsParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}

	// Write virtual-mailbox-domains
	var domainLines []string
	for _, d := range p.Domains {
		if !safeNameRe.MatchString(d) {
			continue
		}
		domainLines = append(domainLines, d+" OK")
	}
	os.WriteFile(postfixVirtualDomains, []byte(strings.Join(domainLines, "\n")+"\n"), 0644)

	// Write virtual-mailbox-maps
	var mbLines []string
	for addr, path := range p.Mailboxes {
		mbLines = append(mbLines, addr+" "+path)
	}
	os.WriteFile(postfixVirtualMailboxes, []byte(strings.Join(mbLines, "\n")+"\n"), 0644)

	// Write virtual aliases
	var aliasLines []string
	for src, dst := range p.Aliases {
		aliasLines = append(aliasLines, src+" "+dst)
	}
	os.WriteFile(postfixVirtualAliases, []byte(strings.Join(aliasLines, "\n")+"\n"), 0644)

	// Postmap the hash files
	exec.Command("postmap", postfixVirtualMailboxes).Run()
	exec.Command("postmap", postfixVirtualAliases).Run()

	// Reload postfix
	exec.Command("systemctl", "reload", "postfix").Run()

	return map[string]string{"status": "ok"}, nil
}

func cmdEmailQueueList(_ json.RawMessage) (interface{}, error) {
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}
	out, err := exec.Command("postqueue", "-j").CombinedOutput()
	if err != nil {
		// Empty queue returns error sometimes
		if strings.TrimSpace(string(out)) == "" {
			return map[string]interface{}{"queue": []interface{}{}}, nil
		}
		return nil, fmt.Errorf("postqueue: %s", strings.TrimSpace(string(out)))
	}

	// Parse JSON lines output
	var items []interface{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		var item interface{}
		if err := json.Unmarshal([]byte(line), &item); err == nil {
			items = append(items, item)
		}
	}
	return map[string]interface{}{"queue": items}, nil
}

func cmdEmailQueueFlush(_ json.RawMessage) (interface{}, error) {
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}
	out, err := exec.Command("postqueue", "-f").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("postqueue flush: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdEmailQueueDelete(params json.RawMessage) (interface{}, error) {
	var p emailQueueDeleteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	// Queue IDs are hex strings
	if !regexp.MustCompile(`^[A-Fa-f0-9]+$`).MatchString(p.QueueID) {
		return nil, fmt.Errorf("invalid queue ID: %s", p.QueueID)
	}
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}
	out, err := exec.Command("postsuper", "-d", p.QueueID).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("postsuper: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"status": "ok"}, nil
}

func cmdEmailGenerateDKIM(params json.RawMessage) (interface{}, error) {
	var p emailDKIMParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Domain) {
		return nil, fmt.Errorf("invalid domain: %s", p.Domain)
	}
	if runtime.GOOS != "linux" {
		return unsupportedOS()
	}

	keyDir := fmt.Sprintf("/etc/opendkim/keys/%s", p.Domain)
	pubKey := filepath.Join(keyDir, "mail.txt")

	// If key already exists, ensure tables are populated and return the public key
	if _, err := os.Stat(pubKey); err == nil {
		// Always ensure OpenDKIM tables are up-to-date (fixes empty tables)
		updateOpenDKIMTables(p.Domain)
		data, _ := os.ReadFile(pubKey)
		return map[string]string{
			"status":     "ok",
			"public_key": extractDKIMPublicKey(string(data)),
			"selector":   "mail",
		}, nil
	}

	// Generate new DKIM key
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return nil, fmt.Errorf("creating DKIM key directory: %w", err)
	}

	out, err := exec.Command("opendkim-genkey", "-b", "2048", "-d", p.Domain, "-D", keyDir, "-s", "mail").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("opendkim-genkey: %s", strings.TrimSpace(string(out)))
	}

	// Set ownership
	exec.Command("chown", "-R", "opendkim:opendkim", keyDir).Run()

	// Update OpenDKIM key table and signing table
	updateOpenDKIMTables(p.Domain)

	// Read and return public key
	data, _ := os.ReadFile(pubKey)
	return map[string]string{
		"status":     "ok",
		"public_key": extractDKIMPublicKey(string(data)),
		"selector":   "mail",
	}, nil
}

// extractDKIMPublicKey extracts the p= value from opendkim-genkey output.
func extractDKIMPublicKey(raw string) string {
	// The file contains a TXT record like:
	// mail._domainkey IN TXT ( "v=DKIM1; k=rsa; p=MIIBIj..." )
	// We need to extract just the record value
	raw = strings.ReplaceAll(raw, "\n", "")
	raw = strings.ReplaceAll(raw, "\t", "")

	// Find content between quotes and concatenate
	var parts []string
	inQuote := false
	current := ""
	for _, ch := range raw {
		if ch == '"' {
			if inQuote {
				parts = append(parts, current)
				current = ""
			}
			inQuote = !inQuote
			continue
		}
		if inQuote {
			current += string(ch)
		}
	}
	return strings.Join(parts, "")
}

// updateOpenDKIMTables adds a domain to the OpenDKIM key and signing tables.
func updateOpenDKIMTables(domain string) {
	keyTable := "/etc/opendkim/key.table"
	signingTable := "/etc/opendkim/signing.table"

	// Key table: mail._domainkey.domain domain:mail:/etc/opendkim/keys/domain/mail.private
	entry := fmt.Sprintf("mail._domainkey.%s %s:mail:/etc/opendkim/keys/%s/mail.private", domain, domain, domain)
	appendIfMissing(keyTable, entry, domain)

	// Signing table: *@domain mail._domainkey.domain
	sigEntry := fmt.Sprintf("*@%s mail._domainkey.%s", domain, domain)
	appendIfMissing(signingTable, sigEntry, domain)

	// Reload opendkim
	exec.Command("systemctl", "reload", "opendkim").Run()
}

// appendIfMissing adds a line to a file if no line containing the search string exists.
func appendIfMissing(filePath, line, search string) {
	existing, _ := os.ReadFile(filePath)
	if strings.Contains(string(existing), search) {
		return
	}
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(line + "\n")
}

// ---------- SpamAssassin ----------

func cmdSpamConfigure(params json.RawMessage) (interface{}, error) {
	var p struct {
		Domain         string   `json:"domain"`
		Enabled        bool     `json:"enabled"`
		ScoreThreshold float64  `json:"score_threshold"`
		Action         string   `json:"action"`
		Whitelist      []string `json:"whitelist"`
		Blacklist      []string `json:"blacklist"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.Domain == "" {
		return nil, fmt.Errorf("invalid params: domain required")
	}

	// Write per-domain SpamAssassin config
	os.MkdirAll("/etc/spamassassin/local.d", 0755)
	var cfg strings.Builder
	cfg.WriteString(fmt.Sprintf("# PinkPanel spam config for %s\n", p.Domain))
	cfg.WriteString(fmt.Sprintf("required_score %.1f\n", p.ScoreThreshold))

	for _, addr := range p.Whitelist {
		cfg.WriteString(fmt.Sprintf("whitelist_from %s\n", addr))
	}
	for _, addr := range p.Blacklist {
		cfg.WriteString(fmt.Sprintf("blacklist_from %s\n", addr))
	}

	cfgPath := fmt.Sprintf("/etc/spamassassin/local.d/%s.cf", p.Domain)
	if err := os.WriteFile(cfgPath, []byte(cfg.String()), 0644); err != nil {
		return nil, fmt.Errorf("writing spam config: %w", err)
	}

	// Write Dovecot sieve rule based on action
	sieveDir := "/var/lib/dovecot/sieve"
	os.MkdirAll(sieveDir, 0755)

	var sieve string
	switch p.Action {
	case "junk":
		sieve = `require ["fileinto", "mailbox"];
if header :contains "X-Spam-Flag" "YES" {
    fileinto :create "Junk";
    stop;
}
`
	case "delete":
		sieve = `require ["fileinto"];
if header :contains "X-Spam-Flag" "YES" {
    discard;
    stop;
}
`
	default: // "mark" — headers only, no filing
		sieve = `# Spam is marked with X-Spam-Flag header only
`
	}

	sievePath := sieveDir + "/spam-to-junk.sieve"
	if err := os.WriteFile(sievePath, []byte(sieve), 0644); err != nil {
		return nil, fmt.Errorf("writing sieve rule: %w", err)
	}

	// Compile sieve
	exec.Command("sievec", sievePath).Run()

	// Fix ownership
	exec.Command("chown", "-R", "vmail:vmail", sieveDir).Run()

	// Reload SpamAssassin
	exec.Command("systemctl", "reload", "spamassassin").Run()

	return map[string]string{"status": "ok"}, nil
}

func cmdSpamStatus(params json.RawMessage) (interface{}, error) {
	result := map[string]interface{}{
		"spamassassin_running": false,
		"spamass_milter_running": false,
	}

	if err := exec.Command("systemctl", "is-active", "--quiet", "spamassassin").Run(); err == nil {
		result["spamassassin_running"] = true
	}
	if err := exec.Command("systemctl", "is-active", "--quiet", "spamass-milter").Run(); err == nil {
		result["spamass_milter_running"] = true
	}

	// Get version
	out, err := exec.Command("spamassassin", "--version").CombinedOutput()
	if err == nil {
		result["version"] = strings.TrimSpace(string(out))
	}

	return result, nil
}

// ---------- ClamAV ----------

func cmdClamAVConfigure(params json.RawMessage) (interface{}, error) {
	var p struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params")
	}

	if p.Enabled {
		// Write clamav-milter config
		os.MkdirAll("/var/spool/postfix/clamav", 0755)
		exec.Command("chown", "clamav:postfix", "/var/spool/postfix/clamav").Run()

		milterConf := `PidFile /var/run/clamav/clamav-milter.pid
MilterSocket /var/spool/postfix/clamav/clamav-milter.sock
MilterSocketMode 660
MilterSocketGroup postfix
FixStaleSocket true
User clamav
ClamdSocket unix:/run/clamav/clamd.ctl
OnInfected Reject
LogInfected Basic
LogClean Off
`
		if err := os.WriteFile("/etc/clamav/clamav-milter.conf", []byte(milterConf), 0644); err != nil {
			return nil, fmt.Errorf("writing clamav-milter config: %w", err)
		}

		// Add ClamAV milter to Postfix if not present
		out, _ := exec.Command("postconf", "-h", "smtpd_milters").CombinedOutput()
		currentMilters := strings.TrimSpace(string(out))
		clamMilter := "unix:/var/spool/postfix/clamav/clamav-milter.sock"
		if !strings.Contains(currentMilters, "clamav") {
			newMilters := currentMilters
			if newMilters != "" {
				newMilters += ", "
			}
			newMilters += clamMilter
			exec.Command("postconf", "-e", "smtpd_milters = "+newMilters).Run()
		}

		exec.Command("systemctl", "enable", "--now", "clamav-daemon").Run()
		exec.Command("systemctl", "enable", "--now", "clamav-freshclam").Run()
		exec.Command("systemctl", "restart", "clamav-milter").Run()
		exec.Command("postfix", "reload").Run()
	} else {
		// Remove ClamAV milter from Postfix
		out, _ := exec.Command("postconf", "-h", "smtpd_milters").CombinedOutput()
		currentMilters := strings.TrimSpace(string(out))
		// Remove clamav socket from milters list
		parts := strings.Split(currentMilters, ",")
		var filtered []string
		for _, p := range parts {
			if !strings.Contains(strings.TrimSpace(p), "clamav") {
				filtered = append(filtered, strings.TrimSpace(p))
			}
		}
		exec.Command("postconf", "-e", "smtpd_milters = "+strings.Join(filtered, ", ")).Run()

		exec.Command("systemctl", "stop", "clamav-milter").Run()
		exec.Command("postfix", "reload").Run()
	}

	return map[string]string{"status": "ok"}, nil
}

func cmdClamAVStatus(params json.RawMessage) (interface{}, error) {
	result := map[string]interface{}{
		"clamav_running":    false,
		"freshclam_running": false,
		"milter_running":    false,
		"enabled":           false,
	}

	if err := exec.Command("systemctl", "is-active", "--quiet", "clamav-daemon").Run(); err == nil {
		result["clamav_running"] = true
	}
	if err := exec.Command("systemctl", "is-active", "--quiet", "clamav-freshclam").Run(); err == nil {
		result["freshclam_running"] = true
	}
	if err := exec.Command("systemctl", "is-active", "--quiet", "clamav-milter").Run(); err == nil {
		result["milter_running"] = true
		result["enabled"] = true
	}

	// Get DB version info
	out, err := exec.Command("clamscan", "--version").CombinedOutput()
	if err == nil {
		result["version"] = strings.TrimSpace(string(out))
	}

	return result, nil
}

// ---------- Mail Autodiscovery ----------

func cmdEmailWriteAutoconfig(params json.RawMessage) (interface{}, error) {
	var p struct {
		Domain   string `json:"domain"`
		Hostname string `json:"hostname"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.Domain == "" {
		return nil, fmt.Errorf("invalid params: domain required")
	}
	if p.Hostname == "" {
		p.Hostname = "mail." + p.Domain
	}

	// Thunderbird autoconfig XML
	autoconfigDir := fmt.Sprintf("/var/www/autoconfig/%s/mail", p.Domain)
	os.MkdirAll(autoconfigDir, 0755)

	thunderbirdXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<clientConfig version="1.1">
  <emailProvider id="%s">
    <domain>%s</domain>
    <displayName>%s Mail</displayName>
    <incomingServer type="imap">
      <hostname>%s</hostname>
      <port>993</port>
      <socketType>SSL</socketType>
      <authentication>password-cleartext</authentication>
      <username>%%EMAILADDRESS%%</username>
    </incomingServer>
    <incomingServer type="imap">
      <hostname>%s</hostname>
      <port>143</port>
      <socketType>STARTTLS</socketType>
      <authentication>password-cleartext</authentication>
      <username>%%EMAILADDRESS%%</username>
    </incomingServer>
    <outgoingServer type="smtp">
      <hostname>%s</hostname>
      <port>587</port>
      <socketType>STARTTLS</socketType>
      <authentication>password-cleartext</authentication>
      <username>%%EMAILADDRESS%%</username>
    </outgoingServer>
  </emailProvider>
</clientConfig>
`, p.Domain, p.Domain, p.Domain, p.Hostname, p.Hostname, p.Hostname)

	if err := os.WriteFile(autoconfigDir+"/config-v1.1.xml", []byte(thunderbirdXML), 0644); err != nil {
		return nil, fmt.Errorf("writing autoconfig XML: %w", err)
	}

	// Outlook autodiscover XML
	autodiscoverDir := fmt.Sprintf("/var/www/autodiscover/%s", p.Domain)
	os.MkdirAll(autodiscoverDir, 0755)

	outlookXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Autodiscover xmlns="http://schemas.microsoft.com/exchange/autodiscover/responseschema/2006">
  <Response xmlns="http://schemas.microsoft.com/exchange/autodiscover/outlook/responseschema/2006a">
    <Account>
      <AccountType>email</AccountType>
      <Action>settings</Action>
      <Protocol>
        <Type>IMAP</Type>
        <Server>%s</Server>
        <Port>993</Port>
        <SSL>on</SSL>
        <LoginName>%%EMAILADDRESS%%</LoginName>
      </Protocol>
      <Protocol>
        <Type>SMTP</Type>
        <Server>%s</Server>
        <Port>587</Port>
        <Encryption>TLS</Encryption>
        <LoginName>%%EMAILADDRESS%%</LoginName>
      </Protocol>
    </Account>
  </Response>
</Autodiscover>
`, p.Hostname, p.Hostname)

	if err := os.WriteFile(autodiscoverDir+"/autodiscover.xml", []byte(outlookXML), 0644); err != nil {
		return nil, fmt.Errorf("writing autodiscover XML: %w", err)
	}

	// Fix ownership
	exec.Command("chown", "-R", "www-data:www-data", "/var/www/autoconfig").Run()
	exec.Command("chown", "-R", "www-data:www-data", "/var/www/autodiscover").Run()

	// Create NGINX snippet for this domain's autoconfig
	phpSock := findPHPSocket()
	_ = phpSock // Not needed for static XML

	nginxSnippet := fmt.Sprintf(`# Mail autodiscovery for %s
location /.well-known/autoconfig/mail/config-v1.1.xml {
    alias /var/www/autoconfig/%s/mail/config-v1.1.xml;
    default_type application/xml;
}

location /autodiscover/autodiscover.xml {
    alias /var/www/autodiscover/%s/autodiscover.xml;
    default_type application/xml;
}

location /Autodiscover/Autodiscover.xml {
    alias /var/www/autodiscover/%s/autodiscover.xml;
    default_type application/xml;
}
`, p.Domain, p.Domain, p.Domain, p.Domain)

	snippetPath := fmt.Sprintf("/etc/nginx/snippets/autoconfig-%s.conf", p.Domain)
	if err := os.WriteFile(snippetPath, []byte(nginxSnippet), 0644); err != nil {
		return nil, fmt.Errorf("writing nginx snippet: %w", err)
	}

	// Include snippet in domain's vhost if not already
	vhostPath := fmt.Sprintf("/etc/nginx/sites-available/%s", p.Domain)
	if data, err := os.ReadFile(vhostPath); err == nil {
		content := string(data)
		includeDirective := fmt.Sprintf("include snippets/autoconfig-%s.conf;", p.Domain)
		if !strings.Contains(content, includeDirective) {
			content = strings.Replace(content, "server_name ", includeDirective+"\n    server_name ", 1)
			os.WriteFile(vhostPath, []byte(content), 0644)
		}
	}

	exec.Command("nginx", "-t").Run()
	exec.Command("systemctl", "reload", "nginx").Run()

	return map[string]string{"status": "ok"}, nil
}

func findPHPSocket() string {
	matches, _ := filepath.Glob("/run/php/php*-fpm.sock")
	if len(matches) > 0 {
		return matches[0]
	}
	return "/run/php/php-fpm.sock"
}

// ---------- Mail SSL ----------

func cmdEmailConfigureSSL(params json.RawMessage) (interface{}, error) {
	var p struct {
		Domain string `json:"domain"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Domain) {
		return nil, fmt.Errorf("invalid domain: %s", p.Domain)
	}

	sslDir := fmt.Sprintf("/usr/local/pinkpanel/data/ssl/%s", p.Domain)
	certPath := filepath.Join(sslDir, "cert.pem")
	keyPath := filepath.Join(sslDir, "key.pem")
	chainPath := filepath.Join(sslDir, "chain.pem")

	// Check cert exists
	if _, err := os.Stat(certPath); err != nil {
		return nil, fmt.Errorf("SSL certificate not found for %s — issue an SSL certificate first", p.Domain)
	}

	// If chain exists, build fullchain (cert + chain)
	fullchainPath := filepath.Join(sslDir, "fullchain.pem")
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("reading cert: %w", err)
	}
	fullchain := string(certData)
	if chainData, err := os.ReadFile(chainPath); err == nil {
		fullchain += "\n" + string(chainData)
	}
	if err := os.WriteFile(fullchainPath, []byte(fullchain), 0644); err != nil {
		return nil, fmt.Errorf("writing fullchain: %w", err)
	}

	// Rebuild multi-domain mail SSL config (scans all mail.* cert dirs)
	if err := rebuildMailSSL(); err != nil {
		return nil, err
	}

	return map[string]string{
		"status":    "ok",
		"cert_path": fullchainPath,
		"key_path":  keyPath,
	}, nil
}

// rebuildMailSSL scans all mail.* SSL cert directories and rebuilds:
// - Postfix: default cert (first domain) + tls_server_sni_maps for per-domain SNI
// - Dovecot: default cert + local_name blocks for per-domain SNI
func rebuildMailSSL() error {
	sslBase := "/usr/local/pinkpanel/data/ssl"

	// Find all mail.* directories that have certs
	type mailCert struct {
		domain    string // e.g. "mail.example.com"
		fullchain string
		key       string
	}
	var certs []mailCert

	entries, err := os.ReadDir(sslBase)
	if err != nil {
		return fmt.Errorf("reading ssl dir: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "mail.") {
			continue
		}
		dir := filepath.Join(sslBase, e.Name())
		fc := filepath.Join(dir, "fullchain.pem")
		key := filepath.Join(dir, "key.pem")
		cert := filepath.Join(dir, "cert.pem")
		// Build fullchain if missing but cert exists
		if _, err := os.Stat(fc); err != nil {
			if _, err2 := os.Stat(cert); err2 != nil {
				continue // no cert at all
			}
			cData, _ := os.ReadFile(cert)
			full := string(cData)
			if chain, err3 := os.ReadFile(filepath.Join(dir, "chain.pem")); err3 == nil {
				full += "\n" + string(chain)
			}
			os.WriteFile(fc, []byte(full), 0644)
		}
		if _, err := os.Stat(key); err != nil {
			continue // no key
		}
		certs = append(certs, mailCert{domain: e.Name(), fullchain: fc, key: key})
	}

	if len(certs) == 0 {
		return nil // no mail certs, nothing to configure
	}

	// Use first cert as default
	defaultCert := certs[0]

	// ── Postfix: default cert + SNI map ──
	postfixCmds := [][]string{
		{"postconf", "-e", fmt.Sprintf("smtpd_tls_cert_file=%s", defaultCert.fullchain)},
		{"postconf", "-e", fmt.Sprintf("smtpd_tls_key_file=%s", defaultCert.key)},
		{"postconf", "-e", "smtpd_tls_security_level=may"},
		{"postconf", "-e", "smtp_tls_security_level=may"},
		{"postconf", "-e", "tls_server_sni_maps=hash:/etc/postfix/sni_maps"},
	}
	for _, args := range postfixCmds {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("postconf failed: %s: %w", strings.TrimSpace(string(out)), err)
		}
	}

	// Write SNI map: one line per mail domain
	var sniLines []string
	for _, mc := range certs {
		sniLines = append(sniLines, fmt.Sprintf("%s %s %s", mc.domain, mc.fullchain, mc.key))
	}
	sniContent := strings.Join(sniLines, "\n") + "\n"
	if err := os.WriteFile("/etc/postfix/sni_maps", []byte(sniContent), 0644); err != nil {
		return fmt.Errorf("writing sni_maps: %w", err)
	}

	// Build the hash db
	if out, err := exec.Command("postmap", "-F", "hash:/etc/postfix/sni_maps").CombinedOutput(); err != nil {
		return fmt.Errorf("postmap sni_maps: %s: %w", strings.TrimSpace(string(out)), err)
	}

	// ── Dovecot: default cert + local_name SNI blocks ──
	var dovecotConf strings.Builder
	dovecotConf.WriteString(fmt.Sprintf(`ssl = required
ssl_cert = <%s
ssl_key = <%s
ssl_min_protocol = TLSv1.2
`, defaultCert.fullchain, defaultCert.key))

	// Add local_name blocks for each mail domain
	for _, mc := range certs {
		dovecotConf.WriteString(fmt.Sprintf(`
local_name %s {
  ssl_cert = <%s
  ssl_key = <%s
}
`, mc.domain, mc.fullchain, mc.key))
	}

	if err := os.WriteFile("/etc/dovecot/conf.d/10-ssl.conf", []byte(dovecotConf.String()), 0644); err != nil {
		return fmt.Errorf("writing dovecot ssl config: %w", err)
	}

	// Reload services
	exec.Command("systemctl", "reload", "postfix").Run()
	exec.Command("systemctl", "reload", "dovecot").Run()

	return nil
}

func cmdEmailSSLStatus(params json.RawMessage) (interface{}, error) {
	var p struct {
		Domain string `json:"domain"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if !safeNameRe.MatchString(p.Domain) {
		return nil, fmt.Errorf("invalid domain: %s", p.Domain)
	}

	sslDir := fmt.Sprintf("/usr/local/pinkpanel/data/ssl/%s", p.Domain)
	certPath := filepath.Join(sslDir, "cert.pem")
	fullchainPath := filepath.Join(sslDir, "fullchain.pem")

	hasCert := false
	if _, err := os.Stat(certPath); err == nil {
		hasCert = true
	}
	mailSSL := false
	if _, err := os.Stat(fullchainPath); err == nil {
		mailSSL = true
	}

	return map[string]any{
		"has_ssl_cert": hasCert,
		"mail_ssl":     mailSSL,
	}, nil
}

// ---------- Helpers ----------

// ---------- Git commands ----------

type gitCloneParams struct {
	URL    string `json:"url"`
	Path   string `json:"path"`
	Branch string `json:"branch"`
}

func cmdGitClone(params json.RawMessage) (interface{}, error) {
	var p gitCloneParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.URL == "" || p.Path == "" {
		return nil, fmt.Errorf("url and path are required")
	}
	if err := validatePath(p.Path); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(p.Path), 0755); err != nil {
		return nil, fmt.Errorf("creating parent directory: %w", err)
	}

	isSSH := strings.HasPrefix(p.URL, "git@") || strings.HasPrefix(p.URL, "ssh://")

	args := []string{}
	// For HTTPS URLs, prevent git from rewriting to SSH via insteadOf config
	if !isSSH {
		args = append(args, "-c", "url.ssh://git@github.com/.insteadOf=''",
			"-c", "url.git@github.com:.insteadOf=''")
	}
	args = append(args, "clone")
	if p.Branch != "" {
		args = append(args, "--branch", p.Branch)
	}
	args = append(args, "--single-branch", p.URL, p.Path)

	cmd := exec.Command("git", args...)
	if isSSH {
		cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=accept-new")
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git clone failed: %s", strings.TrimSpace(string(out)))
	}
	return map[string]any{"output": strings.TrimSpace(string(out))}, nil
}

type gitPullParams struct {
	Path   string `json:"path"`
	Branch string `json:"branch"`
}

func cmdGitPull(params json.RawMessage) (interface{}, error) {
	var p gitPullParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.Path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if err := validatePath(p.Path); err != nil {
		return nil, err
	}

	// Check if remote uses SSH to decide env
	remoteOut, _ := exec.Command("git", "-C", p.Path, "remote", "get-url", "origin").Output()
	remoteURL := strings.TrimSpace(string(remoteOut))
	isSSH := strings.HasPrefix(remoteURL, "git@") || strings.HasPrefix(remoteURL, "ssh://")

	args := []string{"-C", p.Path, "pull", "origin"}
	if p.Branch != "" {
		args = append(args, p.Branch)
	}

	cmd := exec.Command("git", args...)
	if isSSH {
		cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=accept-new")
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git pull failed: %s", strings.TrimSpace(string(out)))
	}
	return map[string]any{"output": strings.TrimSpace(string(out))}, nil
}

type gitInitBareParams struct {
	Path string `json:"path"`
}

func cmdGitInitBare(params json.RawMessage) (interface{}, error) {
	var p gitInitBareParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.Path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if err := validatePath(p.Path); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(p.Path, 0755); err != nil {
		return nil, fmt.Errorf("creating directory: %w", err)
	}

	out, err := exec.Command("git", "init", "--bare", p.Path).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git init --bare failed: %s", strings.TrimSpace(string(out)))
	}
	return map[string]any{"output": strings.TrimSpace(string(out))}, nil
}

type gitDeployParams struct {
	RepoPath      string `json:"repo_path"`
	DeployPath    string `json:"deploy_path"`
	PostDeployCmd string `json:"post_deploy_cmd"`
}

func cmdGitDeploy(params json.RawMessage) (interface{}, error) {
	var p gitDeployParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.RepoPath == "" || p.DeployPath == "" {
		return nil, fmt.Errorf("repo_path and deploy_path are required")
	}
	if err := validatePath(p.RepoPath); err != nil {
		return nil, err
	}
	if err := validatePath(p.DeployPath); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(p.DeployPath, 0755); err != nil {
		return nil, fmt.Errorf("creating deploy directory: %w", err)
	}

	var allOutput string

	// Use rsync to deploy files from the working tree
	// For bare repos, we need to use git archive; for working trees, rsync
	isBare := false
	if _, err := os.Stat(filepath.Join(p.RepoPath, "HEAD")); err == nil {
		if _, err := os.Stat(filepath.Join(p.RepoPath, ".git")); os.IsNotExist(err) {
			isBare = true
		}
	}

	if isBare {
		// For bare repos, use git archive to extract files
		cmd := exec.Command("git", "-C", p.RepoPath, "archive", "HEAD")
		tarCmd := exec.Command("tar", "-xf", "-", "-C", p.DeployPath)
		pipe, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("creating pipe: %w", err)
		}
		tarCmd.Stdin = pipe
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("git archive failed to start: %w", err)
		}
		tarOut, tarErr := tarCmd.CombinedOutput()
		if err := cmd.Wait(); err != nil {
			return nil, fmt.Errorf("git archive failed: %w", err)
		}
		if tarErr != nil {
			return nil, fmt.Errorf("tar extract failed: %s", strings.TrimSpace(string(tarOut)))
		}
		allOutput += "Extracted files from bare repo\n"
	} else {
		// For working trees, rsync excluding .git
		out, err := exec.Command("rsync", "-a", "--delete", "--exclude", ".git", p.RepoPath+"/", p.DeployPath+"/").CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("rsync failed: %s", strings.TrimSpace(string(out)))
		}
		allOutput += "Synced files to deploy path\n"
	}

	// Get latest commit hash
	var commitHash string
	gitDir := p.RepoPath
	if !isBare {
		gitDir = filepath.Join(p.RepoPath, ".git")
	}
	if hashOut, err := exec.Command("git", "--git-dir", gitDir, "rev-parse", "HEAD").Output(); err == nil {
		commitHash = strings.TrimSpace(string(hashOut))
	}

	// Run post-deploy command
	if p.PostDeployCmd != "" {
		cmd := exec.Command("bash", "-c", p.PostDeployCmd)
		cmd.Dir = p.DeployPath
		out, err := cmd.CombinedOutput()
		output := strings.TrimSpace(string(out))
		allOutput += "Post-deploy: " + output + "\n"
		if err != nil {
			return map[string]any{
				"output":      allOutput + "Post-deploy command failed: " + err.Error(),
				"commit_hash": commitHash,
			}, fmt.Errorf("post-deploy command failed: %s", output)
		}
	}

	return map[string]any{
		"output":      allOutput,
		"commit_hash": commitHash,
	}, nil
}

type gitLogParams struct {
	Path  string `json:"path"`
	Limit int    `json:"limit"`
}

func cmdGitLog(params json.RawMessage) (interface{}, error) {
	var p gitLogParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.Path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if err := validatePath(p.Path); err != nil {
		return nil, err
	}
	if p.Limit <= 0 {
		p.Limit = 10
	}

	// Use a custom format for easy parsing
	format := "%H|%an|%ae|%aI|%s"
	out, err := exec.Command("git", "-C", p.Path, "log",
		fmt.Sprintf("--max-count=%d", p.Limit),
		"--format="+format,
	).Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	var commits []map[string]string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 5 {
			continue
		}
		commits = append(commits, map[string]string{
			"hash":    parts[0],
			"author":  parts[1],
			"email":   parts[2],
			"date":    parts[3],
			"message": parts[4],
		})
	}

	return map[string]any{"commits": commits}, nil
}

type gitSetupHookParams struct {
	RepoPath   string `json:"repo_path"`
	WebhookURL string `json:"webhook_url"`
}

func cmdGitSetupHook(params json.RawMessage) (interface{}, error) {
	var p gitSetupHookParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.RepoPath == "" || p.WebhookURL == "" {
		return nil, fmt.Errorf("repo_path and webhook_url are required")
	}
	if err := validatePath(p.RepoPath); err != nil {
		return nil, err
	}

	hookDir := filepath.Join(p.RepoPath, "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		return nil, fmt.Errorf("creating hooks directory: %w", err)
	}

	hookContent := fmt.Sprintf(`#!/bin/bash
# PinkPanel auto-deploy hook
curl -s -X POST "%s" > /dev/null 2>&1 &
`, p.WebhookURL)

	hookPath := filepath.Join(hookDir, "post-receive")
	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		return nil, fmt.Errorf("writing post-receive hook: %w", err)
	}

	return map[string]any{"status": "ok"}, nil
}

func cmdGitSSHKey(_ json.RawMessage) (interface{}, error) {
	keyPath := "/root/.ssh/id_ed25519"
	pubPath := keyPath + ".pub"

	// Generate key if it doesn't exist
	if _, err := os.Stat(pubPath); os.IsNotExist(err) {
		if err := os.MkdirAll("/root/.ssh", 0700); err != nil {
			return nil, fmt.Errorf("creating .ssh directory: %w", err)
		}
		out, err := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-C", "pinkpanel@server").CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("generating SSH key: %s", strings.TrimSpace(string(out)))
		}
	}

	pubKey, err := os.ReadFile(pubPath)
	if err != nil {
		return nil, fmt.Errorf("reading public key: %w", err)
	}

	return map[string]any{"public_key": strings.TrimSpace(string(pubKey))}, nil
}

// ---------- Monitoring commands ----------

type domainDiskUsageParams struct {
	Path string `json:"path"`
}

func cmdDomainDiskUsage(params json.RawMessage) (interface{}, error) {
	var p domainDiskUsageParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.Path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if err := validatePath(p.Path); err != nil {
		return nil, err
	}

	// Use du -sb with a 30-second timeout
	cmd := exec.Command("du", "-sb", p.Path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return map[string]any{"bytes": 0}, nil
	}

	fields := strings.Fields(string(out))
	if len(fields) < 1 {
		return map[string]any{"bytes": 0}, nil
	}
	bytes, _ := strconv.ParseInt(fields[0], 10, 64)
	return map[string]any{"bytes": bytes}, nil
}

type domainBandwidthParams struct {
	LogPath string `json:"log_path"`
}

func cmdDomainBandwidth(params json.RawMessage) (interface{}, error) {
	var p domainBandwidthParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.LogPath == "" {
		return nil, fmt.Errorf("log_path is required")
	}
	// Security: only allow reading from /var/log/nginx/
	if !strings.HasPrefix(p.LogPath, "/var/log/nginx/") {
		return nil, fmt.Errorf("log_path must be under /var/log/nginx/")
	}

	// Sum bytes_sent (field 10 in combined log format)
	cmd := exec.Command("awk", `{sum += $10} END {print sum+0}`, p.LogPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return map[string]any{"bytes": 0}, nil
	}

	bytes, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	return map[string]any{"bytes": bytes}, nil
}

// ---------- Cron commands ----------

type cronSyncParams struct {
	User string `json:"user"`
	Jobs []struct {
		ID       int64  `json:"id"`
		Schedule string `json:"schedule"`
		Command  string `json:"command"`
	} `json:"jobs"`
}

func cmdCronSync(params json.RawMessage) (interface{}, error) {
	var p cronSyncParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.User == "" {
		return nil, fmt.Errorf("user is required")
	}

	// Build crontab content
	var lines []string
	lines = append(lines, "# PinkPanel managed crontab - do not edit manually")
	for _, job := range p.Jobs {
		// Each line: schedule command
		lines = append(lines, fmt.Sprintf("%s %s", job.Schedule, job.Command))
	}
	lines = append(lines, "") // trailing newline
	crontab := strings.Join(lines, "\n")

	// Write via crontab -u USER -
	cmd := exec.Command("crontab", "-u", p.User, "-")
	cmd.Stdin = strings.NewReader(crontab)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("crontab sync failed: %s", strings.TrimSpace(string(out)))
	}

	return map[string]any{"status": "ok", "job_count": len(p.Jobs)}, nil
}

type cronExecuteParams struct {
	User    string `json:"user"`
	Command string `json:"command"`
}

func cmdCronExecute(params json.RawMessage) (interface{}, error) {
	var p cronExecuteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.User == "" || p.Command == "" {
		return nil, fmt.Errorf("user and command are required")
	}

	start := time.Now()

	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		// macOS: su doesn't support -s flag the same way
		cmd = exec.Command("su", "-", p.User, "-c", p.Command)
	} else {
		cmd = exec.Command("su", "-s", "/bin/bash", "-c", p.Command, p.User)
	}

	out, err := cmd.CombinedOutput()
	durationMs := time.Since(start).Milliseconds()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to execute command: %w", err)
		}
	}

	// Limit output to 64KB
	output := string(out)
	if len(output) > 65536 {
		output = output[:65536] + "\n... (output truncated)"
	}

	return map[string]any{
		"exit_code":   exitCode,
		"output":      output,
		"duration_ms": durationMs,
	}, nil
}

func escapeMySQLString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

// ---------- App installer commands ----------

type appDownloadParams struct {
	URL    string `json:"url"`
	Dest   string `json:"dest"`
	Format string `json:"format"`
	Subdir string `json:"subdir"`
}

func cmdAppDownload(params json.RawMessage) (interface{}, error) {
	var p appDownloadParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.URL == "" {
		return nil, fmt.Errorf("url is required")
	}
	if err := validatePath(p.Dest); err != nil {
		return nil, err
	}

	// Determine temp file extension
	ext := "tar.gz"
	if p.Format == "zip" {
		ext = "zip"
	}
	tmpFile := fmt.Sprintf("/tmp/app-download-%d.%s", time.Now().UnixNano(), ext)
	defer os.Remove(tmpFile)

	// Download
	out, err := exec.Command("wget", "-q", "-O", tmpFile, p.URL).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("download failed: %s", strings.TrimSpace(string(out)))
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(p.Dest, 0755); err != nil {
		return nil, fmt.Errorf("creating destination: %w", err)
	}

	// Extract
	if p.Format == "zip" {
		out, err = exec.Command("unzip", "-o", "-q", tmpFile, "-d", p.Dest).CombinedOutput()
	} else {
		out, err = exec.Command("tar", "-xzf", tmpFile, "-C", p.Dest).CombinedOutput()
	}
	if err != nil {
		return nil, fmt.Errorf("extract failed: %s", strings.TrimSpace(string(out)))
	}

	// Move subdir contents up if specified
	subdir := p.Subdir
	if subdir == "auto" {
		// Auto-detect: find the single top-level directory
		entries, _ := os.ReadDir(p.Dest)
		dirs := []string{}
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, e.Name())
			}
		}
		if len(dirs) == 1 {
			subdir = dirs[0]
		} else {
			subdir = ""
		}
	}

	if subdir != "" {
		subdirPath := filepath.Join(p.Dest, subdir)
		if info, err := os.Stat(subdirPath); err == nil && info.IsDir() {
			// Move contents from subdir to dest
			entries, _ := os.ReadDir(subdirPath)
			for _, e := range entries {
				src := filepath.Join(subdirPath, e.Name())
				dst := filepath.Join(p.Dest, e.Name())
				if err := os.Rename(src, dst); err != nil {
					// Fallback to cp + rm for cross-device moves
					cpOut, cpErr := exec.Command("cp", "-a", src, dst).CombinedOutput()
					if cpErr != nil {
						return nil, fmt.Errorf("moving %s: %s", e.Name(), strings.TrimSpace(string(cpOut)))
					}
					os.RemoveAll(src)
				}
			}
			os.Remove(subdirPath)
		}
	}

	return map[string]string{"status": "ok"}, nil
}

type appWPCLIParams struct {
	Path  string   `json:"path"`
	Args  []string `json:"args"`
	RunAs string   `json:"run_as"`
}

func cmdAppWPCLI(params json.RawMessage) (interface{}, error) {
	var p appWPCLIParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := validatePath(p.Path); err != nil {
		return nil, err
	}
	if p.RunAs == "" {
		p.RunAs = "www-data"
	}
	if !safeNameRe.MatchString(p.RunAs) {
		return nil, fmt.Errorf("invalid run_as user: %s", p.RunAs)
	}

	wpBin := "/usr/local/bin/wp"

	// Auto-install wp-cli if missing
	if _, err := os.Stat(wpBin); os.IsNotExist(err) {
		dlOut, dlErr := exec.Command("wget", "-q", "-O", wpBin,
			"https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar").CombinedOutput()
		if dlErr != nil {
			return nil, fmt.Errorf("failed to download wp-cli: %s", strings.TrimSpace(string(dlOut)))
		}
		os.Chmod(wpBin, 0755)
	}

	// Build command: sudo -u {run_as} wp --path={path} {args...}
	cmdArgs := []string{"-u", p.RunAs, wpBin, "--path=" + p.Path}
	cmdArgs = append(cmdArgs, p.Args...)

	cmd := exec.Command("sudo", cmdArgs...)
	output, err := cmd.CombinedOutput()
	outStr := strings.TrimSpace(string(output))
	if err != nil {
		exitCode := 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return map[string]any{
			"output":    outStr,
			"exit_code": exitCode,
		}, fmt.Errorf("wp-cli exited with code %d: %s", exitCode, outStr)
	}

	return map[string]any{
		"output":    outStr,
		"exit_code": 0,
	}, nil
}
