package app_test

import (
	"testing"

	"github.com/kode4food/timebox/postgres"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/cmd/argyll/internal/app"
)

func TestLoadPostgresConfig(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr error
		check   func(*testing.T, app.Config)
	}{
		{
			name: "defaults",
			check: func(t *testing.T, c app.Config) {
				assert.Equal(t, postgres.DefaultURL, c.Postgres.URL)
				assert.Equal(t, app.DefaultPostgresPrefix, c.Postgres.Prefix)
				assert.EqualValues(t,
					postgres.DefaultMaxConns, c.Postgres.MaxConns,
				)
			},
		},
		{
			name: "load_postgres_settings",
			envVars: map[string]string{
				"POSTGRES_URL":       "postgres://db:5432/argyll",
				"POSTGRES_PREFIX":    "custom",
				"POSTGRES_MAX_CONNS": "96",
			},
			check: func(t *testing.T, c app.Config) {
				assert.Equal(t, "postgres://db:5432/argyll", c.Postgres.URL)
				assert.Equal(t, "custom", c.Postgres.Prefix)
				assert.EqualValues(t, 96, c.Postgres.MaxConns)
			},
		},
		{
			name: "invalid_max_conns_errors",
			envVars: map[string]string{
				"POSTGRES_MAX_CONNS": "0",
			},
			wantErr: app.ErrEnvOutOfRange,
		},
		{
			name: "max_conns_beyond_int32_errors",
			envVars: map[string]string{
				"POSTGRES_MAX_CONNS": "4294967297",
			},
			wantErr: app.ErrEnvOutOfRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, tt.envVars)

			cfg, err := app.LoadConfig()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			tt.check(t, cfg)
		})
	}
}
