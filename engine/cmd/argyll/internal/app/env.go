package app

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/kode4food/timebox/raft"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/config"
)

// Config holds everything the App runs with: the engine, the raft backend
// beneath it, and the HTTP server that fronts it
type Config struct {
	Engine          *config.Config
	Raft            raft.Config
	APIHost         string
	WebhookBaseURL  string
	LogLevel        string
	APIPort         int
	ShutdownTimeout time.Duration
}

const (
	DefaultAPIHost         = "0.0.0.0"
	DefaultAPIPort         = 8080
	DefaultWebhookBaseURL  = "http://localhost:8080"
	DefaultLogLevel        = "info"
	DefaultShutdownTimeout = 10 * time.Second

	MaxTCPPort          = 65535
	MaxTimeboxCacheSize = 1_024_000
	MaxMemoCacheSize    = 10_240_000
	MaxRetryMaxRetries  = 1000
	MaxStepTimeout      = 365 * 24 * 60 * api.Minute // 1 year in ms
	MaxRetryInitBackoff = 24 * 60 * api.Minute       // 1 day in ms
	MaxRetryMaxBackoff  = MaxRetryInitBackoff
)

var (
	ErrInvalidEnvValue = errors.New("invalid environment value")
	ErrEnvOutOfRange   = errors.New("environment value out of range")
)

// LoadConfig reads the whole App configuration from the environment
func LoadConfig() (Config, error) {
	rc, err := loadRaftConfig()
	if err != nil {
		return Config{}, err
	}
	eng, err := loadEngineConfig(rc)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Engine:          eng,
		Raft:            rc,
		APIHost:         DefaultAPIHost,
		WebhookBaseURL:  DefaultWebhookBaseURL,
		LogLevel:        DefaultLogLevel,
		APIPort:         DefaultAPIPort,
		ShutdownTimeout: DefaultShutdownTimeout,
	}
	if apiHost := os.Getenv("API_HOST"); apiHost != "" {
		cfg.APIHost = apiHost
	}
	if webhookBaseURL := os.Getenv("WEBHOOK_BASE_URL"); webhookBaseURL != "" {
		cfg.WebhookBaseURL = webhookBaseURL
	}
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = logLevel
	}
	if err := loadEnvInt(loadEnvIntArgs[int]{
		key: "API_PORT",
		dst: &cfg.APIPort,
		min: 0,
		max: MaxTCPPort,
	}); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// loadEngineConfig reads the engine settings from the environment, taking the
// node identity and cluster membership from the raft configuration
func loadEngineConfig(rc raft.Config) (*config.Config, error) {
	cfg := config.NewDefaultConfig()
	cfg.NodeID = api.NodeID(rc.LocalID)
	cfg.Nodes = raftNodes(rc.Servers)

	if backoffType := os.Getenv("RETRY_BACKOFF_TYPE"); backoffType != "" {
		cfg.Work.BackoffType = backoffType
	}

	if err := loadEnvInt(loadEnvIntArgs[int]{
		key: "TIMEBOX_CACHE_SIZE",
		dst: &cfg.Timebox.CacheSize,
		min: 0,
		max: MaxTimeboxCacheSize,
	}); err != nil {
		return nil, err
	}
	if err := loadEnvInt(loadEnvIntArgs[int]{
		key: "MEMO_CACHE_SIZE",
		dst: &cfg.MemoCacheSize,
		min: 0,
		max: MaxMemoCacheSize,
	}); err != nil {
		return nil, err
	}
	if err := loadEnvInt(loadEnvIntArgs[int64]{
		key: "STEP_TIMEOUT",
		dst: &cfg.StepTimeout,
		min: 0,
		max: MaxStepTimeout,
	}); err != nil {
		return nil, err
	}

	if err := loadEnvInt(loadEnvIntArgs[int]{
		key: "RETRY_MAX_RETRIES",
		dst: &cfg.Work.MaxRetries,
		min: 0,
		max: MaxRetryMaxRetries,
	}); err != nil {
		return nil, err
	}
	if err := loadEnvInt(loadEnvIntArgs[int64]{
		key: "RETRY_INITIAL_BACKOFF",
		dst: &cfg.Work.InitBackoff,
		min: 0,
		max: MaxRetryInitBackoff,
	}); err != nil {
		return nil, err
	}
	if err := loadEnvInt(loadEnvIntArgs[int64]{
		key: "RETRY_MAX_BACKOFF",
		dst: &cfg.Work.MaxBackoff,
		min: 0,
		max: MaxRetryMaxBackoff,
	}); err != nil {
		return nil, err
	}
	return cfg, nil
}

// loadEnvIntArgs bundles the bounds an environment integer must fall within
type loadEnvIntArgs[T ~int | ~int64] struct {
	key string
	dst *T
	min T
	max T
}

// loadEnvInt reads key from the environment, parses it as an integer, and
// sets *dst if the value is in the range (min, max]
func loadEnvInt[T ~int | ~int64](args loadEnvIntArgs[T]) error {
	s := os.Getenv(args.key)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: %s=%q", ErrInvalidEnvValue, args.key, s)
	}
	tv := T(v)
	if tv <= args.min || tv > args.max {
		return fmt.Errorf("%w: %s=%d, want [%d, %d]",
			ErrEnvOutOfRange, args.key, tv, args.min+1, args.max)
	}
	*args.dst = tv
	return nil
}
