// Package config handles configuration loading via Viper.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Agent    AgentConfig    `mapstructure:"agent"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Security SecurityConfig `mapstructure:"security"`
	Paths    PathsConfig    `mapstructure:"paths"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Path         string `mapstructure:"path"`
	WALMode      bool   `mapstructure:"wal_mode"`
	BusyTimeout  int    `mapstructure:"busy_timeout_ms"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

type RedisConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type AgentConfig struct {
	Socket     string `mapstructure:"socket"`
	SecretFile string `mapstructure:"secret_file"`
}

type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	File       string `mapstructure:"file"`
	MaxSizeMB  int    `mapstructure:"max_size_mb"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAgeDays int    `mapstructure:"max_age_days"`
	Compress   bool   `mapstructure:"compress"`
	Console    bool   `mapstructure:"console"`
}

type SecurityConfig struct {
	JWTSecretFile   string        `mapstructure:"jwt_secret_file"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	BcryptCost      int           `mapstructure:"bcrypt_cost"`
}

type PathsConfig struct {
	WebRoot   string `mapstructure:"web_root"`
	Backups   string `mapstructure:"backups"`
	Templates string `mapstructure:"templates"`
	Temp      string `mapstructure:"temp"`
}

func setDefaults() {
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8443)

	viper.SetDefault("database.path", "pinkpanel.db")
	viper.SetDefault("database.wal_mode", true)
	viper.SetDefault("database.busy_timeout_ms", 5000)
	viper.SetDefault("database.max_open_conns", 25)

	viper.SetDefault("redis.enabled", false)
	viper.SetDefault("redis.address", "127.0.0.1:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	viper.SetDefault("agent.socket", "/tmp/pinkpanel-agent/agent.sock")
	viper.SetDefault("agent.secret_file", "")

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.file", "")
	viper.SetDefault("logging.max_size_mb", 100)
	viper.SetDefault("logging.max_backups", 5)
	viper.SetDefault("logging.max_age_days", 30)
	viper.SetDefault("logging.compress", true)
	viper.SetDefault("logging.console", true)

	viper.SetDefault("security.jwt_secret_file", "")
	viper.SetDefault("security.access_token_ttl", "15m")
	viper.SetDefault("security.refresh_token_ttl", "168h")
	viper.SetDefault("security.bcrypt_cost", 12)

	viper.SetDefault("paths.web_root", "/home")
	viper.SetDefault("paths.backups", "backups")
	viper.SetDefault("paths.templates", "embedded")
	viper.SetDefault("paths.temp", "/tmp/pinkpanel")
}

// Load reads configuration from file, environment, and returns a Config.
// configPath is optional; if empty, searches current dir and /etc/pinkpanel/.
func Load(configPath string) (*Config, error) {
	setDefaults()

	viper.SetConfigName("pinkpanel")
	viper.SetConfigType("yml")

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/pinkpanel")
		viper.AddConfigPath("/usr/local/pinkpanel")
	}

	// Environment variable overrides: PINKPANEL_SERVER_PORT=9443
	viper.SetEnvPrefix("PINKPANEL")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
		// Config file not found is OK — use defaults
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}
