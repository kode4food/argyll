package worker

import (
	"context"
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
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	w := New(cfg, provider)
	if len(provider.Names()) > 0 {
		w.loadCapabilities(context.Background())
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           w.Handler(),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	slog.Info("Agent worker listening", slog.String("addr", addr))
	return srv.Serve(listener)
}
