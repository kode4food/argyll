package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/auth"
)

func TestServiceTokens(t *testing.T) {
	t.Setenv("ARGYLL_TOKEN_URL", "")
	assert.Nil(t, auth.ServiceTokens())

	t.Setenv("ARGYLL_TOKEN_URL", "http://idp/token")
	assert.NotNil(t, auth.ServiceTokens())
}

func TestLoadAuthenticatorRejectsOtherProviders(t *testing.T) {
	for _, provider := range []string{"apikey", "header", "oidc,apikey"} {
		t.Run(provider, func(t *testing.T) {
			t.Setenv("AUTH_PROVIDER", provider)
			_, err := auth.LoadAuthenticator()
			assert.ErrorIs(t, err, auth.ErrInvalidAuthProvider)
		})
	}
}
