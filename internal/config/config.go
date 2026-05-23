package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Proxy     ProxyConfig     `yaml:"proxy"`
	Dashboard DashboardConfig `yaml:"dashboard"`
	License   LicenseConfig   `yaml:"license"`
	AI        AIConfig        `yaml:"ai"`
	Engine    EngineConfig    `yaml:"engine"`
	Storage   StorageConfig   `yaml:"storage"`
	SSH       SSHConfig       `yaml:"ssh"`
	Alert     AlertConfig     `yaml:"alert"`
}

type ProxyConfig struct {
	ListenAddr      string   `yaml:"listen_addr"`
	TLSListenAddr   string   `yaml:"tls_listen_addr"`
	BackendAddr     string   `yaml:"backend_addr"`
	TLSCertFile     string   `yaml:"tls_cert_file"`
	TLSKeyFile      string   `yaml:"tls_key_file"`
	ACMEEnabled     bool     `yaml:"acme_enabled"`
	ACMEEmail       string   `yaml:"acme_email"`
	ACMEDomains     []string `yaml:"acme_domains"`
	DynamicProtect  bool     `yaml:"dynamic_protect"`
	IPTablesEnable  bool     `yaml:"iptables_enable"`
	IPTablesPort    int      `yaml:"iptables_port"`
	ReadTimeout     int      `yaml:"read_timeout"`
	WriteTimeout    int      `yaml:"write_timeout"`
}

type DashboardConfig struct {
	ListenAddr  string   `yaml:"listen_addr"`
	JWTSecret   string   `yaml:"jwt_secret"`
	CORSOrigins []string `yaml:"cors_origins"`
	TLSCertFile string   `yaml:"tls_cert_file"`
	TLSKeyFile  string   `yaml:"tls_key_file"`
}

// HardcodedEd25519PublicKey is the license verification public key.
// It is embedded at compile time and cannot be overridden via config file.
// This prevents attackers from substituting their own key pair in open-source deployments.
const HardcodedEd25519PublicKey = "OFIQFYlYqgk4wD4GFVRtJEQgeP7lnjN3i2CkTGVbJfg"

type LicenseConfig struct {
	CenterURL string `yaml:"center_url"`
	PublicKey string `yaml:"-"` // ignored from YAML; always use HardcodedEd25519PublicKey
	Timeout   int    `yaml:"timeout"`
}

type AIConfig struct {
	Enabled          bool           `yaml:"enabled"`
	Provider         string         `yaml:"provider"`
	AsyncTimeout     int            `yaml:"async_timeout"`
	CacheTTL         int            `yaml:"cache_ttl"`
	MaxRequests      int            `yaml:"max_requests_per_min"`
	FailOpen         bool           `yaml:"fail_open"`
	HighRiskPaths    []string       `yaml:"high_risk_paths"`
	PerIPRate        int            `yaml:"per_ip_rate"`
	PerIPBurst       int            `yaml:"per_ip_burst"`
	CircuitThreshold int            `yaml:"circuit_threshold"`
	CircuitReset     int            `yaml:"circuit_reset"`
	Providers        ProviderConfig `yaml:"providers"`
}

type ProviderConfig struct {
	Claude ClaudeProvider `yaml:"claude"`
	OpenAI OpenAIProvider `yaml:"openai"`
}

type ClaudeProvider struct {
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url"`
}

type OpenAIProvider struct {
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url"`
}

type EngineConfig struct {
	RulesDir         string          `yaml:"rules_dir"`
	Preset           string          `yaml:"preset"`
	ObservationMode  bool            `yaml:"observation_mode"`
	RateLimit        RateLimitConfig `yaml:"rate_limit"`
}

type RateLimitConfig struct {
	RequestsPerMinute int `yaml:"requests_per_minute"`
	BurstSize         int `yaml:"burst_size"`
}

type StorageConfig struct {
	Type             string `yaml:"type"`              // "sqlite" (default) or "mysql"
	Path             string `yaml:"path"`              // SQLite db file path
	DSN              string `yaml:"dsn"`               // MySQL DSN
	MaxOpenConns     int    `yaml:"max_open_conns"`    // default 25 for MySQL
	MaxIdleConns     int    `yaml:"max_idle_conns"`    // default 10 for MySQL
	LogRetentionDays int    `yaml:"log_retention_days"`
}

type SSHConfig struct {
	Enabled    bool   `yaml:"enabled"`
	LogPath    string `yaml:"log_path"`
	MaxFails   int    `yaml:"max_fails"`
	BanMinutes int    `yaml:"ban_minutes"`
}

type AlertConfig struct {
	Enabled     bool        `yaml:"enabled"`
	ThrottleMin int         `yaml:"throttle_minutes"`
	WebhookURL  string      `yaml:"webhook_url"`
	Email       EmailConfig `yaml:"email"`
}

type EmailConfig struct {
	Host     string   `yaml:"host"`
	Port     int      `yaml:"port"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	From     string   `yaml:"from"`
	To       []string `yaml:"to"`
}

func DefaultConfig() *Config {
	return &Config{
		Proxy: ProxyConfig{
			ListenAddr:     ":8080",
			BackendAddr:    "127.0.0.1:80",
			IPTablesEnable: true,
			IPTablesPort:   80,
			ReadTimeout:    30,
			WriteTimeout:   30,
		},
		Dashboard: DashboardConfig{
			ListenAddr:  ":9090",
			JWTSecret:   "ZhiYu-WAF-secret-change-me",
			CORSOrigins: []string{"http://localhost:9090"},
		},
		License: LicenseConfig{
			CenterURL: "https://sq.sreai.cloud:3333",
			PublicKey: HardcodedEd25519PublicKey,
			Timeout:   8,
		},
		AI: AIConfig{
			Enabled:          true,
			Provider:         "openai",
			AsyncTimeout:     5,
			CacheTTL:         300,
			MaxRequests:      60,
			FailOpen:         true,
			PerIPRate:        10,
			PerIPBurst:       2,
			CircuitThreshold: 5,
			CircuitReset:     30,
			HighRiskPaths: []string{
				"/admin",
				"/api/admin",
				"/api/v1/admin",
				"/login",
				"/api/v1/auth/login",
				"/upload",
				"/payment",
				"/pay",
				"/checkout",
			},
			Providers: ProviderConfig{
				Claude: ClaudeProvider{
					Model:   "claude-sonnet-4-20250514",
					BaseURL: "",
				},
				OpenAI: OpenAIProvider{
					Model:   "",
					BaseURL: "",
				},
			},
		},
		Engine: EngineConfig{
			RulesDir: "./configs/rules",
			Preset:   "balanced",
			RateLimit: RateLimitConfig{
				RequestsPerMinute: 60,
				BurstSize:         10,
			},
		},
		Storage: StorageConfig{
			Type:             "sqlite",
			Path:             "./data/zhiyu-waf.db",
			LogRetentionDays: 30,
		},
		SSH: SSHConfig{
			Enabled:    false,
			MaxFails:   5,
			BanMinutes: 30,
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	return cfg, nil
}

// GenerateRandomSecret creates a random 32-byte hex string for JWT signing.
func GenerateRandomSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Validate checks essential config fields.
func (c *Config) Validate() error {
	if c.Proxy.ListenAddr == "" {
		return fmt.Errorf("proxy.listen_addr is required")
	}
	if c.Proxy.BackendAddr == "" {
		return fmt.Errorf("proxy.backend_addr is required")
	}
	if c.Proxy.IPTablesPort <= 0 {
		c.Proxy.IPTablesPort = 80
	}
	if c.Proxy.ReadTimeout <= 0 {
		c.Proxy.ReadTimeout = 30
	}
	if c.Proxy.WriteTimeout <= 0 {
		c.Proxy.WriteTimeout = 30
	}
	if c.Dashboard.JWTSecret == "ZhiYu-WAF-secret-change-me" {
		c.Dashboard.JWTSecret = GenerateRandomSecret()
		log.Println("WARNING: JWT secret was default value, generated random secret (sessions will not persist across restarts)")
	}
	if c.License.CenterURL == "" {
		c.License.CenterURL = "https://sq.sreai.cloud:3333"
	}
	// Always enforce the hardcoded public key — config file cannot override it
	c.License.PublicKey = HardcodedEd25519PublicKey
	if c.License.Timeout <= 0 {
		c.License.Timeout = 8
	}
	return nil
}
