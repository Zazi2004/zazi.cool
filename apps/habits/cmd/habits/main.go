// Command habits serves the habit tracker at habits.zazi.cool.
//
// Usage:
//
//	habits           apply migrations, then serve HTTP
//	habits migrate   apply migrations, then exit
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"zazi.cool/habits/internal/auth"
	"zazi.cool/habits/internal/migrate"
	"zazi.cool/habits/internal/server"
	"zazi.cool/habits/internal/store"
	"zazi.cool/habits/migrations"
)

// version is the git SHA of the build, injected with -ldflags "-X main.version=<sha>".
var version = "dev"

// shutdownTimeout bounds how long in-flight requests may take to finish on SIGTERM.
const shutdownTimeout = 10 * time.Second

// cleanupEvery is the period of the purge of expired sessions.
const cleanupEvery = time.Hour

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

	st := store.New(pool)
	sessions := auth.NewManager(cfg.passwordHash, st, strings.HasPrefix(cfg.baseURL, "https://"))
	go cleanUp(ctx, sessions)

	return serve(ctx, ":"+cfg.port, server.NewHandler(server.Config{
		Store:          st,
		Auth:           sessions,
		Version:        version,
		BaseURL:        cfg.baseURL,
		TrustProxy:     cfg.trustProxy,
		LoginLimit:     cfg.loginLimit,
		Location:       cfg.location,
		EditWindowDays: cfg.editWindowDays,
	}))
}

// cleanUp deletes the expired sessions now and then every cleanupEvery, until ctx
// ends. The deletion is idempotent: running it twice is harmless.
func cleanUp(ctx context.Context, sessions *auth.Manager) {
	ticker := time.NewTicker(cleanupEvery)
	defer ticker.Stop()
	for {
		if n, err := sessions.DeleteExpired(ctx, time.Now()); err != nil {
			slog.Error("clean up sessions", "error", err)
		} else if n > 0 {
			slog.Info("expired sessions deleted", "count", n)
		}

		// select waits for whichever comes first: the next tick or the shutdown.
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
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
	port         string
	databaseURL  string
	passwordHash string
	baseURL      string
	trustProxy   bool
	loginLimit   int
	// location defines the "day" (TZ_NAME); editWindowDays how far back days can be edited.
	location       *time.Location
	editWindowDays int
}

// loadConfig reads the configuration from environment variables (see .env.example).
func loadConfig() (config, error) {
	cfg := config{
		port:         getenv("PORT", "8080"),
		databaseURL:  os.Getenv("DATABASE_URL"),
		passwordHash: os.Getenv("ADMIN_PASSWORD_HASH"),
		baseURL:      strings.TrimSuffix(getenv("BASE_URL", "https://habits.zazi.cool"), "/"),
	}
	if cfg.databaseURL == "" {
		return config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.passwordHash == "" {
		return config{}, errors.New("ADMIN_PASSWORD_HASH is required (see `make hash-password`)")
	}
	if err := auth.CheckHash(cfg.passwordHash); err != nil {
		return config{}, err
	}
	if u, err := url.Parse(cfg.baseURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return config{}, fmt.Errorf("BASE_URL must be an absolute http(s) URL, got %q", cfg.baseURL)
	}
	if n, err := strconv.Atoi(cfg.port); err != nil || n < 1 || n > 65535 {
		return config{}, fmt.Errorf("PORT must be a port number, got %q", cfg.port)
	}

	var err error
	if cfg.trustProxy, err = strconv.ParseBool(getenv("TRUST_PROXY", "false")); err != nil {
		return config{}, fmt.Errorf("TRUST_PROXY: %w", err)
	}
	if cfg.loginLimit, err = strconv.Atoi(getenv("RATE_LIMIT_LOGIN", "10")); err != nil || cfg.loginLimit <= 0 {
		return config{}, errors.New("RATE_LIMIT_LOGIN must be a positive integer")
	}
	if cfg.location, err = time.LoadLocation(getenv("TZ_NAME", "Europe/Paris")); err != nil {
		return config{}, fmt.Errorf("TZ_NAME: %w", err)
	}
	if cfg.editWindowDays, err = strconv.Atoi(getenv("EDIT_WINDOW_DAYS", "14")); err != nil || cfg.editWindowDays < 0 {
		return config{}, errors.New("EDIT_WINDOW_DAYS must be a non-negative integer")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
