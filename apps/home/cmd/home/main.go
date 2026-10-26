// Command home serves the homepage of zazi.cool.
//
// Usage:
//
//	home           apply migrations, then serve HTTP
//	home migrate   apply migrations, then exit
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

	"zazi.cool/home/internal/auth"
	"zazi.cool/home/internal/content"
	"zazi.cool/home/internal/migrate"
	"zazi.cool/home/internal/server"
	"zazi.cool/home/migrations"
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
	cfg, err := loadConfig(os.Getenv)
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

	handler, err := server.NewHandler(server.Config{
		DB:             pool,
		Content:        content.NewStore(pool),
		Version:        version,
		PublicBaseURL:  cfg.publicBaseURL,
		MaxUploadBytes: int64(cfg.maxUploadMB) << 20,
	})
	if err != nil {
		return err
	}
	return serve(ctx, ":"+cfg.port, handler)
}

// serve runs the HTTP server until ctx is cancelled, then shuts it down gracefully:
// it stops accepting connections and waits for in-flight requests to finish.
func serve(ctx context.Context, addr string, handler http.Handler) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second, // banner uploads from a phone can be slow
		WriteTimeout:      30 * time.Second,
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

// config is the whole configuration of the app (see .env.example). The admin and
// upload settings are read and checked now, so that a deployment with a wrong
// value fails at startup; the admin (T4) and the banner upload (T6) use them.
type config struct {
	port          string
	databaseURL   string
	passwordHash  string
	publicBaseURL string
	trustProxy    bool
	loginLimit    int
	maxUploadMB   int
}

// loadConfig reads the configuration through getenv (os.Getenv, or a map in tests).
func loadConfig(getenv func(string) string) (config, error) {
	get := func(key, fallback string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return fallback
	}
	cfg := config{
		port:          get("PORT", "8080"),
		databaseURL:   getenv("DATABASE_URL"),
		passwordHash:  getenv("ADMIN_PASSWORD_HASH"),
		publicBaseURL: strings.TrimSuffix(get("PUBLIC_BASE_URL", "http://localhost:8080"), "/"),
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
	if u, err := url.Parse(cfg.publicBaseURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return config{}, fmt.Errorf("PUBLIC_BASE_URL must be an absolute http(s) URL, got %q", cfg.publicBaseURL)
	}
	if n, err := strconv.Atoi(cfg.port); err != nil || n < 1 || n > 65535 {
		return config{}, fmt.Errorf("PORT must be a port number, got %q", cfg.port)
	}

	var err error
	if cfg.trustProxy, err = strconv.ParseBool(get("TRUST_PROXY", "false")); err != nil {
		return config{}, fmt.Errorf("TRUST_PROXY: %w", err)
	}
	ints := []struct {
		name  string
		value *int
		def   string
	}{
		{"RATE_LIMIT_LOGIN", &cfg.loginLimit, "10"},
		{"MAX_UPLOAD_MB", &cfg.maxUploadMB, "5"},
	}
	for _, v := range ints {
		n, err := strconv.Atoi(get(v.name, v.def))
		if err != nil || n <= 0 {
			return config{}, fmt.Errorf("%s must be a positive integer", v.name)
		}
		*v.value = n
	}
	return cfg, nil
}
