package worker

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"
)

const (
	defaultPort       = "8070"
	readHeaderTimeout = 10 * time.Second
)

// Serve starts the worker for the selected provider API
func Serve(cfg *Config, provider Provider) error {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	addr := net.JoinHostPort(os.Getenv("HOST"), port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           New(cfg, provider).Handler(),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	slog.Info("Agent worker listening", slog.String("addr", addr))
	return srv.ListenAndServe()
}
