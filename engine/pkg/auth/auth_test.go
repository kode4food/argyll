package auth_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/auth"
)

type staticAuth struct {
	principal *auth.Principal
	err       error
}

func TestPrincipalAllows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		roles    []auth.Role
		required auth.Role
		allowed  bool
	}{
		{
			name:     "same role",
			roles:    []auth.Role{auth.RoleOperator},
			required: auth.RoleOperator,
			allowed:  true,
		},
		{
			name:     "higher role includes lower",
			roles:    []auth.Role{auth.RoleAdmin},
			required: auth.RoleReader,
			allowed:  true,
		},
		{
			name:     "lower role excludes higher",
			roles:    []auth.Role{auth.RoleReader},
			required: auth.RoleOperator,
		},
		{
			name:     "no roles",
			required: auth.RoleReader,
		},
		{
			name:     "unknown required role",
			roles:    []auth.Role{auth.RoleAdmin},
			required: auth.Role("superuser"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &auth.Principal{Roles: tc.roles}
			assert.Equal(t, tc.allowed, p.Allows(tc.required))
		})
	}
}

func TestHandler(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p, ok := auth.PrincipalFrom(r.Context()); ok {
			assert.Equal(t, []auth.Role{auth.RoleOperator}, p.Roles)
		}
		w.WriteHeader(http.StatusTeapot)
	})
	operator := staticAuth{principal: &auth.Principal{
		Roles: []auth.Role{auth.RoleOperator},
	}}
	reader := staticAuth{principal: &auth.Principal{
		Roles: []auth.Role{auth.RoleReader},
	}}
	failing := staticAuth{err: errors.New("bad token")}

	for _, tc := range []struct {
		name   string
		authn  auth.Authenticator
		status int
	}{
		{
			name:   "auth disabled",
			status: http.StatusTeapot,
		},
		{
			name:   "failed authentication",
			authn:  failing,
			status: http.StatusUnauthorized,
		},
		{
			name:   "missing principal",
			authn:  staticAuth{},
			status: http.StatusUnauthorized,
		},
		{
			name:   "missing role",
			authn:  reader,
			status: http.StatusForbidden,
		},
		{
			name:   "allowed",
			authn:  operator,
			status: http.StatusTeapot,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			h := auth.Handler(next, tc.authn, auth.RoleOperator)
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/engine", nil))
			assert.Equal(t, tc.status, w.Code)
			if tc.status == http.StatusUnauthorized {
				assert.Equal(t, "Bearer", w.Header().Get("WWW-Authenticate"))
			}
		})
	}
}

func TestForbiddenNamesRoles(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	for msg, roles := range map[string][]auth.Role{
		"needs the admin role, but this account has operator": {auth.RoleOperator},
		"needs the admin role, but this account has no role":  nil,
	} {
		authn := staticAuth{principal: &auth.Principal{Roles: roles}}
		w := httptest.NewRecorder()
		auth.Handler(next, authn, auth.RoleAdmin).ServeHTTP(w, bearer("x"))
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "not permitted: "+msg)
	}
}

func (s staticAuth) Authenticate(*http.Request) (*auth.Principal, error) {
	return s.principal, s.err
}

func bearer(token string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/engine", nil)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return r
}
