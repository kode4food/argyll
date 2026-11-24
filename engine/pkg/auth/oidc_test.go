package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/auth"
)

type identityProvider struct {
	*httptest.Server
	signer jose.Signer
}

const (
	testKeyID    = "test-key"
	testAudience = "argyll"
	testKeyBits  = 2048
)

func TestNewOIDCRequiresSettings(t *testing.T) {
	_, err := auth.NewOIDC(context.Background(), auth.OIDCConfig{Audience: testAudience})
	assert.ErrorIs(t, err, auth.ErrOIDCIssuerRequired)

	_, err = auth.NewOIDC(context.Background(), auth.OIDCConfig{Issuer: "http://idp"})
	assert.ErrorIs(t, err, auth.ErrOIDCAudienceRequired)
}

func TestOIDCAuthenticate(t *testing.T) {
	idp := newIdentityProvider(t)
	for _, tc := range []struct {
		name string
		cfg  auth.OIDCConfig
	}{
		{
			name: "signing keys URL",
			cfg: auth.OIDCConfig{
				Issuer:   idp.URL,
				JWKSURL:  idp.URL + "/keys",
				Audience: testAudience,
			},
		},
		{
			name: "discovery",
			cfg: auth.OIDCConfig{
				Issuer:   idp.URL,
				Audience: testAudience,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, err := auth.NewOIDC(context.Background(), tc.cfg)
			assert.NoError(t, err)

			p, err := o.Authenticate(bearer(idp.token(t, idp.claims("operator", "staff"))))
			assert.NoError(t, err)
			assert.Equal(t, []auth.Role{auth.RoleOperator}, p.Roles)
			assert.WithinDuration(t, time.Now().Add(time.Hour), p.Expires, time.Minute)
		})
	}
}

func TestOIDCRejects(t *testing.T) {
	idp := newIdentityProvider(t)
	o, err := auth.NewOIDC(context.Background(), auth.OIDCConfig{
		Issuer:   idp.URL,
		JWKSURL:  idp.URL + "/keys",
		Audience: testAudience,
	})
	assert.NoError(t, err)

	wrongAudience := idp.claims("admin")
	wrongAudience["aud"] = "other"
	wrongIssuer := idp.claims("admin")
	wrongIssuer["iss"] = "http://elsewhere"
	expired := idp.claims("admin")
	expired["exp"] = time.Now().Add(-time.Hour).Unix()

	for _, tc := range []struct {
		name  string
		token string
	}{
		{
			name: "no token",
		},
		{
			name:  "malformed token",
			token: "not-a-jwt",
		},
		{
			name:  "wrong audience",
			token: idp.token(t, wrongAudience),
		},
		{
			name:  "wrong issuer",
			token: idp.token(t, wrongIssuer),
		},
		{
			name:  "expired",
			token: idp.token(t, expired),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := o.Authenticate(bearer(tc.token))
			assert.ErrorIs(t, err, auth.ErrUnauthenticated)
		})
	}
}

func TestOIDCCustomRolesClaim(t *testing.T) {
	idp := newIdentityProvider(t)
	o, err := auth.NewOIDC(context.Background(), auth.OIDCConfig{
		Issuer:     idp.URL,
		JWKSURL:    idp.URL + "/keys",
		Audience:   testAudience,
		RolesClaim: "groups",
	})
	assert.NoError(t, err)

	claims := idp.claims("admin")
	claims["groups"] = []string{"reader"}
	p, err := o.Authenticate(bearer(idp.token(t, claims)))
	assert.NoError(t, err)
	assert.Equal(t, []auth.Role{auth.RoleReader}, p.Roles)
}

func TestOIDCWebSocketToken(t *testing.T) {
	idp := newIdentityProvider(t)
	o, err := auth.NewOIDC(context.Background(), auth.OIDCConfig{
		Issuer:   idp.URL,
		JWKSURL:  idp.URL + "/keys",
		Audience: testAudience,
	})
	assert.NoError(t, err)
	token := idp.token(t, idp.claims("reader"))

	ws := httptest.NewRequest(http.MethodGet, "/engine/ws", nil)
	ws.Header.Set("Upgrade", "websocket")
	ws.Header.Set("Sec-WebSocket-Protocol", "argyll, "+auth.WebSocketTokenPrefix+token)
	p, err := o.Authenticate(ws)
	assert.NoError(t, err)
	assert.Equal(t, []auth.Role{auth.RoleReader}, p.Roles)

	// The subprotocol carries a token only on a WebSocket upgrade
	plain := httptest.NewRequest(http.MethodGet, "/engine/ws", nil)
	plain.Header.Set("Sec-WebSocket-Protocol", auth.WebSocketTokenPrefix+token)
	_, err = o.Authenticate(plain)
	assert.ErrorIs(t, err, auth.ErrUnauthenticated)
}

func TestOIDCMalformedRolesClaim(t *testing.T) {
	idp := newIdentityProvider(t)
	o, err := auth.NewOIDC(context.Background(), auth.OIDCConfig{
		Issuer:   idp.URL,
		JWKSURL:  idp.URL + "/keys",
		Audience: testAudience,
	})
	assert.NoError(t, err)

	for _, roles := range [][]any{{"admin", 1}, {"admin", nil}} {
		claims := idp.claims()
		claims["roles"] = roles
		p, err := o.Authenticate(bearer(idp.token(t, claims)))
		assert.NoError(t, err)
		assert.Empty(t, p.Roles)
	}
}

func (idp *identityProvider) token(t *testing.T, claims map[string]any) string {
	t.Helper()
	res, err := jwt.Signed(idp.signer).Claims(claims).Serialize()
	assert.NoError(t, err)
	return res
}

func (idp *identityProvider) claims(roles ...string) map[string]any {
	return map[string]any{
		"iss":   idp.URL,
		"aud":   testAudience,
		"sub":   "user-1",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"roles": roles,
	}
}

// newIdentityProvider serves discovery and signing keys, as an OIDC provider does
func newIdentityProvider(t *testing.T) *identityProvider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, testKeyBits)
	assert.NoError(t, err)
	signer, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.RS256,
		Key:       key,
	}, (&jose.SignerOptions{}).WithHeader("kid", testKeyID))
	assert.NoError(t, err)

	idp := &identityProvider{signer: signer}
	mux := http.NewServeMux()
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
			Key:       &key.PublicKey,
			KeyID:     testKeyID,
			Algorithm: string(jose.RS256),
			Use:       "sig",
		}}})
	})
	discovery := func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":   idp.URL,
			"jwks_uri": idp.URL + "/keys",
		})
	}
	mux.HandleFunc("/.well-known/openid-configuration", discovery)
	idp.Server = httptest.NewServer(mux)
	t.Cleanup(idp.Close)
	return idp
}
