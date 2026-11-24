package auth

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
)

type (
	// OIDC authenticates bearer tokens issued by an OpenID Connect provider
	OIDC struct {
		verifier   *oidc.IDTokenVerifier
		rolesClaim string
	}

	// OIDCConfig names the token issuer and the claims a token must carry
	OIDCConfig struct {
		// Issuer must match each token's iss claim
		Issuer string
		// JWKSURL is where signing keys are fetched, found through discovery when empty
		JWKSURL string
		// Audience must appear in each token's aud claim
		Audience string
		// RolesClaim names the claim holding the caller's Roles
		RolesClaim string
	}
)

// DefaultRolesClaim names the roles array in an access token
const DefaultRolesClaim = "roles"

var (
	ErrOIDCIssuerRequired   = errors.New("OIDC issuer required")
	ErrOIDCAudienceRequired = errors.New("OIDC audience required")
)

var _ Authenticator = (*OIDC)(nil)

// NewOIDC builds an access-token verifier for the configured issuer and audience
func NewOIDC(ctx context.Context, cfg OIDCConfig) (*OIDC, error) {
	switch {
	case cfg.Issuer == "":
		return nil, ErrOIDCIssuerRequired
	case cfg.Audience == "":
		return nil, ErrOIDCAudienceRequired
	}
	verify := &oidc.Config{ClientID: cfg.Audience}
	var verifier *oidc.IDTokenVerifier
	if cfg.JWKSURL != "" {
		keys := oidc.NewRemoteKeySet(ctx, cfg.JWKSURL)
		verifier = oidc.NewVerifier(cfg.Issuer, keys, verify)
	} else {
		p, err := oidc.NewProvider(ctx, cfg.Issuer)
		if err != nil {
			return nil, err
		}
		verifier = p.Verifier(verify)
	}
	return &OIDC{
		verifier:   verifier,
		rolesClaim: cmp.Or(cfg.RolesClaim, DefaultRolesClaim),
	}, nil
}

// Authenticate verifies the bearer token and reads its expiry and roles
func (o *OIDC) Authenticate(r *http.Request) (*Principal, error) {
	raw, ok := bearerToken(r)
	if !ok {
		return nil, ErrUnauthenticated
	}
	token, err := o.verifier.Verify(r.Context(), raw)
	if err != nil {
		return nil, errors.Join(ErrUnauthenticated, err)
	}
	var claims map[string]json.RawMessage
	if err := token.Claims(&claims); err != nil {
		return nil, errors.Join(ErrUnauthenticated, err)
	}
	return &Principal{
		Expires: token.Expiry,
		Roles:   knownRoles(roleNames(claims[o.rolesClaim])),
	}, nil
}

// roleNames rejects the entire claim if any entry is malformed
func roleNames(claim json.RawMessage) []string {
	var entries []*string
	if err := json.Unmarshal(claim, &entries); err != nil {
		return nil
	}
	res := make([]string, 0, len(entries))
	for _, e := range entries {
		if e == nil {
			return nil
		}
		res = append(res, *e)
	}
	return res
}
