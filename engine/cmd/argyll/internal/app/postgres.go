package app

import (
	"os"

	"github.com/kode4food/timebox/postgres"
)

const (
	DefaultPostgresPrefix = "argyll"
	MaxPostgresMaxConns   = 1024
)

// loadPostgresConfig reads the postgres backend settings from the environment
func loadPostgresConfig() (postgres.Config, error) {
	cfg := postgres.DefaultConfig().With(postgres.Config{
		Prefix: DefaultPostgresPrefix,
	})
	if url := os.Getenv("POSTGRES_URL"); url != "" {
		cfg.URL = url
	}
	if prefix := os.Getenv("POSTGRES_PREFIX"); prefix != "" {
		cfg.Prefix = prefix
	}
	if err := loadEnvInt(loadEnvIntArgs[int32]{
		key: "POSTGRES_MAX_CONNS",
		dst: &cfg.MaxConns,
		min: 0,
		max: MaxPostgresMaxConns,
	}); err != nil {
		return postgres.Config{}, err
	}
	return cfg, cfg.Validate()
}
