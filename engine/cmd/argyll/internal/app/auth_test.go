package app_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/cmd/argyll/internal/app"
	"github.com/kode4food/argyll/engine/cmd/argyll/internal/server"
	"github.com/kode4food/argyll/engine/pkg/auth"
)

type reader struct{}

func TestReaderSpacePreview(t *testing.T) {
	h := server.NewServer(nil, nil).WithAuthenticator(reader{}).SetupRoutes()

	for method, status := range map[string]int{
		http.MethodPost:   http.StatusBadRequest,
		http.MethodPut:    http.StatusForbidden,
		http.MethodDelete: http.StatusForbidden,
	} {
		req := httptest.NewRequest(method, "/engine/spaces/preview", strings.NewReader("{"))
		req.Header.Set("Authorization", "Bearer key")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		assert.Equal(t, status, w.Code)
	}
}

func TestLoadConfigAuth(t *testing.T) {
	for _, tc := range []struct {
		name     string
		env      map[string]string
		wantErr  error
		wantAuth bool
	}{
		{
			name: "off by default",
		},
		{
			name: "none",
			env:  map[string]string{"AUTH_PROVIDER": "none"},
		},
		{
			name: "OIDC",
			env: map[string]string{
				"AUTH_PROVIDER": "oidc",
				"AUTH_ISSUER":   "http://idp/realms/argyll",
				"AUTH_JWKS_URL": "http://idp/realms/argyll/certs",
				"AUTH_AUDIENCE": "argyll",
			},
			wantAuth: true,
		},
		{
			name: "OIDC without audience",
			env: map[string]string{
				"AUTH_PROVIDER": "oidc",
				"AUTH_ISSUER":   "http://idp/realms/argyll",
				"AUTH_JWKS_URL": "http://idp/realms/argyll/certs",
			},
			wantErr: auth.ErrOIDCAudienceRequired,
		},
		{
			name:    "unknown provider",
			env:     map[string]string{"AUTH_PROVIDER": "saml"},
			wantErr: auth.ErrInvalidAuthProvider,
		},
		{
			name:    "provider list",
			env:     map[string]string{"AUTH_PROVIDER": "oidc,apikey"},
			wantErr: auth.ErrInvalidAuthProvider,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			cfg, err := app.LoadConfig()
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.wantAuth, cfg.Auth != nil)
		})
	}
}

func (reader) Authenticate(*http.Request) (*auth.Principal, error) {
	return &auth.Principal{Roles: []auth.Role{auth.RoleReader}}, nil
}
