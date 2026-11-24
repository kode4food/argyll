package argyll

import (
	"context"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/kode4food/argyll/engine/pkg/api"
)

const (
	// TokenURLEnv names the OAuth2 token endpoint for service-account credentials
	TokenURLEnv = "ARGYLL_TOKEN_URL"

	// ClientIDEnv names the service account's client ID
	ClientIDEnv = "ARGYLL_CLIENT_ID"

	// ClientSecretEnv names the service account's client secret
	ClientSecretEnv = "ARGYLL_CLIENT_SECRET"
)

func engineHTTPClient(timeout time.Duration) *http.Client {
	base := &http.Client{
		Timeout:       timeout,
		CheckRedirect: api.SameOriginRedirect,
	}
	tokenURL := os.Getenv(TokenURLEnv)
	if tokenURL == "" {
		return base
	}
	cfg := clientcredentials.Config{
		ClientID:     os.Getenv(ClientIDEnv),
		ClientSecret: os.Getenv(ClientSecretEnv),
		TokenURL:     tokenURL,
	}
	res := cfg.Client(context.WithValue(context.Background(), oauth2.HTTPClient, base))
	res.Timeout = timeout
	res.CheckRedirect = api.SameOriginRedirect
	return res
}
