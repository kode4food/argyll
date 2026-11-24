package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// Authenticator identifies the caller of a request
	Authenticator interface {
		Authenticate(*http.Request) (*Principal, error)
	}

	// Principal is an authenticated caller and the Roles it holds, until Expires when it is set
	Principal struct {
		Expires time.Time
		Roles   []Role
	}

	// Role grants access to a level of operations, each including the ones below it
	Role string

	principalKey struct{}
)

const (
	RoleReader   Role = "reader"
	RoleOperator Role = "operator"
	RoleAdmin    Role = "admin"

	// WebSocketTokenPrefix marks the WebSocket subprotocol that carries a bearer token
	WebSocketTokenPrefix = "bearer."
)

var (
	ErrUnauthenticated = errors.New("authentication required")
	ErrForbidden       = errors.New("not permitted")
)

// roleOrder ranks Roles so that a higher Role includes every lower one
var roleOrder = []Role{RoleReader, RoleOperator, RoleAdmin}

// Handler serves next only for callers holding the Role a request requires
func Handler(next http.Handler, a Authenticator, role Role) http.Handler {
	if a == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := a.Authenticate(r)
		if err != nil || p == nil {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeError(w, http.StatusUnauthorized, ErrUnauthenticated)
			return
		}
		if !p.Allows(role) {
			writeError(w, http.StatusForbidden, forbidden(p, role))
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, p)))
	})
}

// PrincipalFrom returns the caller that Handler authenticated for a request
func PrincipalFrom(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(*Principal)
	return p, ok
}

// Allows reports whether the caller holds the required role or a higher one
func (p *Principal) Allows(required Role) bool {
	need := slices.Index(roleOrder, required)
	if need < 0 {
		return false
	}
	for _, r := range p.Roles {
		if slices.Index(roleOrder, r) >= need {
			return true
		}
	}
	return false
}

func forbidden(p *Principal, required Role) error {
	held := "no role"
	if len(p.Roles) > 0 {
		names := make([]string, len(p.Roles))
		for i, r := range p.Roles {
			names[i] = string(r)
		}
		held = strings.Join(names, ", ")
	}
	return fmt.Errorf("%w: needs the %s role, but this account has %s",
		ErrForbidden, required, held)
}

func knownRoles(names []string) []Role {
	var res []Role
	for _, n := range names {
		if r := Role(strings.TrimSpace(n)); slices.Contains(roleOrder, r) {
			res = append(res, r)
		}
	}
	return res
}

// bearerToken accepts a subprotocol credential because browser upgrades lack custom headers
func bearerToken(r *http.Request) (string, bool) {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if ok && strings.EqualFold(scheme, "Bearer") && strings.TrimSpace(token) != "" {
		return strings.TrimSpace(token), true
	}
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return "", false
	}
	for _, line := range r.Header.Values("Sec-WebSocket-Protocol") {
		for proto := range strings.SplitSeq(line, ",") {
			token, ok := strings.CutPrefix(strings.TrimSpace(proto), WebSocketTokenPrefix)
			if ok && token != "" {
				return token, true
			}
		}
	}
	return "", false
}

func writeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", api.JSONContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.ErrorResponse{
		Error:  err.Error(),
		Status: status,
	})
}
