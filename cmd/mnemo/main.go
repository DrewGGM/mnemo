// Command mnemo runs the memory service.
package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DrewGGM/mnemo/internal/config"
	"github.com/DrewGGM/mnemo/internal/httpapi"
)

const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	idleTimeout       = 120 * time.Second
	shutdownTimeout   = 15 * time.Second
)

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stderr))
}

func run(args []string, getenv func(string) string, logOut io.Writer) int {
	logger := slog.New(slog.NewJSONHandler(logOut, nil))
	if len(args) != 1 || args[0] != "serve" {
		logger.Error("usage: mnemo serve")
		return 2
	}
	cfg, err := config.Load(getenv)
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := serve(ctx, cfg, logger); err != nil {
		logger.Error("server stopped", "error", err)
		return 1
	}
	return 0
}

func serve(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	ln, err := new(net.ListenConfig).Listen(ctx, "tcp", cfg.ListenAddr)
	if err != nil {
		return err
	}
	if !isLoopback(ln.Addr()) {
		logger.Warn("listening on a non-loopback address; terminate TLS in a proxy in front of mnemo", "addr", ln.Addr().String())
	}
	srv := &http.Server{
		Handler:           httpapi.NewServer(httpapi.Deps{}),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		IdleTimeout:       idleTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	logger.Info("serving", "addr", ln.Addr().String(), "config", cfg)

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Info("stopped")
	return nil
}

func isLoopback(addr net.Addr) bool {
	tcp, ok := addr.(*net.TCPAddr)
	return ok && tcp.IP.IsLoopback()
}
