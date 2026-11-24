package app

import (
	"net/http"

	"golang.org/x/oauth2"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/auth"
)

func workerClient() *http.Client {
	c := &http.Client{CheckRedirect: api.SameOriginRedirect}
	if ts := auth.ServiceTokens(); ts != nil {
		c.Transport = &oauth2.Transport{Source: ts}
	}
	return c
}
