package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/auth"
	"github.com/kode4food/argyll/mcp"
)

const readHeaderTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}

func run() error {
	engine := flag.String("engine", "http://localhost:8080", "Engine base URL")
	listen := flag.String("listen", "", "Address for the Streamable HTTP transport")
	flag.Parse()

	client := &http.Client{CheckRedirect: api.SameOriginRedirect}
	if *listen == "" {
		if ts := auth.ServiceTokens(); ts != nil {
			client.Transport = &oauth2.Transport{Source: ts}
		}
		return mcp.NewServer(*engine, client).Run(context.Background())
	}
	h, err := mcp.NewServer(*engine, client).Handler()
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              *listen,
		Handler:           h,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	return srv.ListenAndServe()
}
