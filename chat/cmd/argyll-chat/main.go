package main

import (
	"cmp"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kode4food/argyll/chat"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/auth"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

const (
	defaultPort       = "8072"
	readHeaderTimeout = 10 * time.Second
)

var (
	ErrInvalidAgentWorkers = errors.New("invalid AGENT_WORKERS")
)

func main() {
	if err := run(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}

func run() error {
	workers, err := loadAgentWorkers()
	if err != nil {
		return err
	}
	authn, err := auth.LoadAuthenticator()
	if err != nil {
		return err
	}
	client := &http.Client{
		Timeout:       chat.DefaultTimeout,
		CheckRedirect: api.SameOriginRedirect,
	}
	handler := chat.NewServer(workers, client).Handler(authn)
	port := cmp.Or(os.Getenv("PORT"), defaultPort)
	srv := &http.Server{
		Addr:              net.JoinHostPort(os.Getenv("HOST"), port),
		Handler:           auth.CORS(handler),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	return srv.ListenAndServe()
}

func loadAgentWorkers() (builtins.AgentWorkers, error) {
	res := builtins.AgentWorkers{}
	for entry := range strings.SplitSeq(os.Getenv("AGENT_WORKERS"), ",") {
		if strings.TrimSpace(entry) == "" {
			continue
		}
		provider, base, ok := strings.Cut(entry, "=")
		provider = strings.TrimSpace(provider)
		base = strings.TrimRight(strings.TrimSpace(base), "/")
		if !ok || provider == "" || base == "" {
			return nil, fmt.Errorf("%w: %q", ErrInvalidAgentWorkers, entry)
		}
		res[api.ProviderID(provider)] = base
	}
	return res, nil
}
