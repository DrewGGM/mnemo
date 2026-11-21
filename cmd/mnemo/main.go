// Command mnemo runs the memory service.
package main

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/DrewGGM/mnemo/internal/store"
)

const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	idleTimeout       = 120 * time.Second
	shutdownTimeout   = 15 * time.Second
	migrateTimeout    = 5 * time.Minute
)

// adminURLVar is read here and never by config.Load, so serve and mcp cannot
// see the credential of the role that bypasses row level security (ADR-0007).
const adminURLVar = "MNEMO_ADMIN_DATABASE_URL"

var errUsage = errors.New("usage: mnemo serve | mnemo migrate")

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stderr))
}

func run(args []string, getenv func(string) string, logOut io.Writer) int {
	logger := slog.New(slog.NewJSONHandler(logOut, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := errUsage
	if len(args) == 1 {
		switch args[0] {
		case "serve":
			err = runServe(ctx, getenv, logger)
		case "migrate":
			err = runMigrate(ctx, getenv, logger)
		}
	}
	switch {
	case errors.Is(err, errUsage):
		logger.Error(err.Error())
		return 2
	case err != nil:
		logger.Error("mnemo failed", "error", err)
		return 1
	}
	return 0
}

func runMigrate(ctx context.Context, getenv func(string) string, logger *slog.Logger) error {
	adminURL := getenv(adminURLVar)
	if adminURL == "" {
		return errors.New(adminURLVar + " is required")
	}
	ctx, cancel := context.WithTimeout(ctx, migrateTimeout)
	defer cancel()
	if err := store.Migrate(ctx, adminURL); err != nil {
		return err
	}
	logger.Info("schema migrated")
	return nil
}

func runServe(ctx context.Context, getenv func(string) string, logger *slog.Logger) error {
	cfg, err := config.Load(getenv)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	pool, err := store.OpenPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	ready := func(ctx context.Context) error {
		err := store.Ready(ctx, pool)
		if err != nil {
			logger.Warn("not ready", "error", err)
		}
		return err
	}
	return serve(ctx, cfg, httpapi.Deps{Ready: ready}, logger)
}

func serve(ctx context.Context, cfg config.Config, deps httpapi.Deps, logger *slog.Logger) error {
	ln, err := new(net.ListenConfig).Listen(ctx, "tcp", cfg.ListenAddr)
	if err != nil {
		return err
	}
	if !isLoopback(ln.Addr()) {
		logger.Warn("listening on a non-loopback address; terminate TLS in a proxy in front of mnemo", "addr", ln.Addr().String())
	}
	srv := &http.Server{
		Handler:           httpapi.NewServer(deps),
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
