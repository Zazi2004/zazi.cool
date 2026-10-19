// Command pastebin serves the pastebin at pb.zazi.cool.
//
// Usage:
//
//	pastebin           apply migrations, then serve HTTP
//	pastebin migrate   apply migrations, then exit
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"zazi.cool/pastebin/internal/migrate"
	"zazi.cool/pastebin/internal/server"
	"zazi.cool/pastebin/migrations"
)

// version is the git SHA of the build, injected with -ldflags "-X main.version=<sha>".
var version = "dev"

// shutdownTimeout bounds how long in-flight requests may take to finish on SIGTERM.
const shutdownTimeout = 10 * time.Second

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(os.Args[1:]); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// ctx is cancelled on SIGINT (Ctrl+C) or SIGTERM (sent by Kubernetes on shutdown).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.databaseURL)
	if err != nil {
		return fmt.Errorf("configure database pool: %w", err)
	}
	defer pool.Close()

	if err := migrate.Up(ctx, pool, migrations.FS); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if len(args) > 0 && args[0] == "migrate" {
		return nil
	}

	return serve(ctx, ":"+cfg.port, server.NewHandler(pool, version, server.NewRateLimiter(cfg.rateLimitCreate, server.RateWindow)))
}

// serve runs the HTTP server until ctx is cancelled, then shuts it down gracefully:
// it stops accepting connections and waits for in-flight requests to finish.
func serve(ctx context.Context, addr string, handler http.Handler) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// ListenAndServe blocks, so it runs in its own goroutine; its result comes back
	// through a channel. The buffer lets the goroutine exit even if nobody reads.
	errc := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", addr, "version", version)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	// After Shutdown, ListenAndServe returns http.ErrServerClosed: that is expected.
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}

type config struct {
	port            string
	databaseURL     string
	rateLimitCreate int
}

// loadConfig reads the configuration from environment variables (see .env.example).
func loadConfig() (config, error) {
	cfg := config{
		port:        getenv("PORT", "8080"),
		databaseURL: os.Getenv("DATABASE_URL"),
	}
	if cfg.databaseURL == "" {
		return config{}, errors.New("DATABASE_URL is required")
	}
	limit, err := strconv.Atoi(getenv("RATE_LIMIT_CREATE", "10"))
	if err != nil || limit < 1 {
		return config{}, fmt.Errorf("RATE_LIMIT_CREATE must be a positive integer, got %q", os.Getenv("RATE_LIMIT_CREATE"))
	}
	cfg.rateLimitCreate = limit
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
