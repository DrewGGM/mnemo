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

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/DrewGGM/mnemo"
	"github.com/DrewGGM/mnemo/internal/config"
	"github.com/DrewGGM/mnemo/internal/httpapi"
	"github.com/DrewGGM/mnemo/internal/mcpserver"
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

// mcpSubjectVar names the subject whose memory `mnemo mcp` serves until a
// credential binds it (T-029).
const mcpSubjectVar = "MNEMO_MCP_SUBJECT"

// devTenant is the development tenant inserted by 0001_init.sql, the only
// one `mnemo mcp` serves until a credential names the tenant (T-029).
var devTenant = [16]byte{15: 1}

var errUsage = errors.New("usage: mnemo serve | mnemo migrate | mnemo mcp")

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
		case "mcp":
			err = runMCP(ctx, getenv, logger)
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
	embedder, err := newEmbedder(cfg, logger)
	if err != nil {
		return err
	}
	mem, err := mnemo.Open(ctx, pool, mnemo.Config{Embedder: embedder})
	if err != nil {
		return err
	}
	ready := func(ctx context.Context) error {
		err := store.Ready(ctx, pool)
		if err != nil {
			logger.Warn("not ready", "error", err)
		}
		return err
	}
	return serve(ctx, cfg, httpapi.Deps{Ready: ready, Memory: mem, Logger: logger}, logger)
}

// runMCP serves MCP over stdin and stdout, so every log line goes to
// logger, which writes to stderr; it is also made the default logger, which
// internal/mcpserver logs tool failures with.
func runMCP(ctx context.Context, getenv func(string) string, logger *slog.Logger) error {
	slog.SetDefault(logger)
	cfg, err := config.Load(getenv)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	subject := getenv(mcpSubjectVar)
	if subject == "" {
		return errors.New(mcpSubjectVar + " is required")
	}
	pool, err := store.OpenPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	embedder, err := newEmbedder(cfg, logger)
	if err != nil {
		return err
	}
	mem, err := mnemo.Open(ctx, pool, mnemo.Config{Embedder: embedder})
	if err != nil {
		return err
	}
	err = mcpserver.New(mem.Scope(devTenant, subject)).Run(ctx, &mcp.StdioTransport{})
	if ctx.Err() != nil {
		logger.Info("stopped")
		return nil
	}
	return err
}

// newEmbedder returns the deterministic fake embedder while this build has no
// adapter for a real one, and refuses to start if a real one is configured,
// so a deployment never believes it embeds with a model it is not using.
func newEmbedder(cfg config.Config, logger *slog.Logger) (mnemo.Embedder, error) {
	if cfg.EmbedBaseURL != "" {
		return nil, errors.New("MNEMO_EMBED_BASE_URL is set, but this build has no embedding adapter; unset it to use the fake embedder")
	}
	logger.Warn("MNEMO_EMBED_BASE_URL is not set: embedding with the deterministic fake embedder, which is meant for tests only")
	return mnemo.FakeEmbedder(cfg.EmbedDim), nil
}

func serve(ctx context.Context, cfg config.Config, deps httpapi.Deps, logger *slog.Logger) error {
	ln, err := new(net.ListenConfig).Listen(ctx, "tcp", cfg.ListenAddr)
	if err != nil {
		return err
	}
	if !isLoopback(ln.Addr()) {
		logger.Warn("listening on a non-loopback address, but /v1 has no authentication in this build: anyone who reaches this address can read and write the memory of every subject; publish it on loopback only, and terminate TLS in a proxy in front of mnemo", "addr", ln.Addr().String())
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
