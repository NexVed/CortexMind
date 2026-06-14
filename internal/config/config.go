package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Config is the root configuration for the CORTEX daemon, loaded from
// cortex.yaml and overridable via environment variables.
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	GitHub  GitHubConfig  `mapstructure:"github"`
	Scanner ScannerConfig `mapstructure:"scanner"`
	Search  SearchConfig  `mapstructure:"search"`
	Sync    SyncConfig    `mapstructure:"sync"`
	Env     string        `mapstructure:"env"`
	LogLevel string       `mapstructure:"log_level"`
}

type ServerConfig struct {
	Port    int    `mapstructure:"port"`
	MCPPort int    `mapstructure:"mcp_port"`
	DataDir string `mapstructure:"data_dir"`
}

type GitHubConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
}

type ScannerConfig struct {
	IntervalMinutes   int      `mapstructure:"interval_minutes"`
	MaxFileSizeKB     int      `mapstructure:"max_file_size_kb"`
	IgnoredDirs       []string `mapstructure:"ignored_dirs"`
	IgnoredExtensions []string `mapstructure:"ignored_extensions"`
}

type SearchConfig struct {
	EnableSemantic bool   `mapstructure:"enable_semantic"`
	OllamaURL      string `mapstructure:"ollama_url"`
	EmbeddingModel string `mapstructure:"embedding_model"`
	VectorDBURL    string `mapstructure:"vector_db_url"`
}

type SyncConfig struct {
	AutoSync   bool `mapstructure:"auto_sync"`
	SyncOnPush bool `mapstructure:"sync_on_push"`
}

// C is the process-wide loaded configuration.
var C *Config

// Load reads cortex.yaml (if present), applies defaults, and overlays
// environment variables. It never fails hard — missing config falls back
// to sensible defaults.
func Load() *Config {
	v := viper.New()
	v.SetConfigName("cortex")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath(".cortex")
	if home, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(filepath.Join(home, ".cortex"))
	}

	setDefaults(v)

	// Environment variables: CORTEX_SERVER_PORT, CORTEX_GITHUB_CLIENT_ID, etc.
	v.SetEnvPrefix("CORTEX")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		log.Warn().Err(err).Msg("no cortex.yaml found, using defaults + env")
	} else {
		log.Info().Str("file", v.ConfigFileUsed()).Msg("loaded config")
	}

	// Explicit env bindings for the documented secrets.
	_ = v.BindEnv("github.client_id", "CORTEX_GITHUB_CLIENT_ID")
	_ = v.BindEnv("github.client_secret", "CORTEX_GITHUB_CLIENT_SECRET")
	_ = v.BindEnv("env", "CORTEX_ENV")
	_ = v.BindEnv("log_level", "CORTEX_LOG_LEVEL")
	_ = v.BindEnv("search.ollama_url", "CORTEX_OLLAMA_URL")
	_ = v.BindEnv("server.data_dir", "CORTEX_DATA_DIR")

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal config, using defaults")
		cfg = defaultConfig()
	}

	cfg.Server.DataDir = expandHome(cfg.Server.DataDir)
	C = cfg
	return cfg
}

func setDefaults(v *viper.Viper) {
	d := defaultConfig()
	v.SetDefault("server.port", d.Server.Port)
	v.SetDefault("server.mcp_port", d.Server.MCPPort)
	v.SetDefault("server.data_dir", d.Server.DataDir)
	v.SetDefault("scanner.interval_minutes", d.Scanner.IntervalMinutes)
	v.SetDefault("scanner.max_file_size_kb", d.Scanner.MaxFileSizeKB)
	v.SetDefault("scanner.ignored_dirs", d.Scanner.IgnoredDirs)
	v.SetDefault("scanner.ignored_extensions", d.Scanner.IgnoredExtensions)
	v.SetDefault("search.enable_semantic", d.Search.EnableSemantic)
	v.SetDefault("search.ollama_url", d.Search.OllamaURL)
	v.SetDefault("search.embedding_model", d.Search.EmbeddingModel)
	v.SetDefault("sync.auto_sync", d.Sync.AutoSync)
	v.SetDefault("sync.sync_on_push", d.Sync.SyncOnPush)
	v.SetDefault("env", d.Env)
	v.SetDefault("log_level", d.LogLevel)
}

func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:    8090,
			MCPPort: 8091,
			DataDir: "~/.cortex",
		},
		Scanner: ScannerConfig{
			IntervalMinutes: 30,
			MaxFileSizeKB:   500,
			IgnoredDirs:     []string{"node_modules", ".git", "dist", "build", "target", "__pycache__", ".venv", "vendor"},
			IgnoredExtensions: []string{".lock", ".sum", ".min.js", ".map", ".png", ".jpg", ".jpeg", ".gif",
				".svg", ".ico", ".woff", ".woff2", ".ttf", ".pdf", ".zip", ".exe", ".bin"},
		},
		Search: SearchConfig{
			EnableSemantic: false,
			OllamaURL:      "http://localhost:11434",
			EmbeddingModel: "bge-m3",
			VectorDBURL:    "http://localhost:8123",
		},
		Sync: SyncConfig{
			AutoSync:   true,
			SyncOnPush: true,
		},
		Env:      "development",
		LogLevel: "info",
	}
}

// DataDirPath resolves the data directory, creating it if necessary.
func (c *Config) DataDirPath() string {
	dir := expandHome(c.Server.DataDir)
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	return path
}
