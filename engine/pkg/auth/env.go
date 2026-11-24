package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/kode4food/argyll/engine/pkg/api"
)

const (
	authNone            = "none"
	authOIDC            = "oidc"
	defaultTokenTimeout = 30 * time.Second
)

var (
	ErrInvalidAuthProvider = errors.New("invalid AUTH_PROVIDER")
)

// LoadAuthenticator configures OIDC from the environment or returns nil when disabled
func LoadAuthenticator() (Authenticator, error) {
	switch provider := strings.TrimSpace(os.Getenv("AUTH_PROVIDER")); provider {
	case "", authNone:
		return nil, nil
	case authOIDC:
		return NewOIDC(context.Background(), OIDCConfig{
			Issuer:     os.Getenv("AUTH_ISSUER"),
			JWKSURL:    os.Getenv("AUTH_JWKS_URL"),
			Audience:   os.Getenv("AUTH_AUDIENCE"),
			RolesClaim: os.Getenv("AUTH_ROLES_CLAIM"),
		})
	default:
		return nil, fmt.Errorf("%w: %q", ErrInvalidAuthProvider, provider)
	}
}

// ServiceTokens loads OAuth client credentials or returns nil when unconfigured
func ServiceTokens() oauth2.TokenSource {
	tokenURL := os.Getenv("ARGYLL_TOKEN_URL")
	if tokenURL == "" {
		return nil
	}
	cfg := clientcredentials.Config{
		ClientID:     os.Getenv("ARGYLL_CLIENT_ID"),
		ClientSecret: os.Getenv("ARGYLL_CLIENT_SECRET"),
		TokenURL:     tokenURL,
	}
	cl := &http.Client{
		Timeout:       defaultTokenTimeout,
		CheckRedirect: api.SameOriginRedirect,
	}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, cl)
	return cfg.TokenSource(ctx)
}
