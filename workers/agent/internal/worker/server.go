package worker

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultPort       = "8070"
	readHeaderTimeout = 10 * time.Second
	shutdownTimeout   = 170 * time.Second
)

// Serve starts the worker for the selected provider API
func Serve(cfg *Config, provider Provider) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
		w.loadCapabilities(ctx)
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           w.Handler(),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	slog.Info("Agent worker listening", slog.String("addr", addr))
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(listener)
	}()
	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}
	stop()

	shutdown, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	httpErr := make(chan error, 1)
	go func() {
		httpErr <- srv.Shutdown(shutdown)
	}()
	err = w.Shutdown(shutdown)
	return errors.Join(err, <-httpErr)
}
