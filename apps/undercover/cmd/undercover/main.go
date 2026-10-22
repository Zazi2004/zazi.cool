// Command undercover serves the Undercover game.
//
// Usage:
//
//	undercover           apply migrations and seed pairs, then serve HTTP
//	undercover migrate   apply migrations and seed pairs, then exit
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

	"zazi.cool/undercover/internal/api"
	"zazi.cool/undercover/internal/migrate"
	"zazi.cool/undercover/internal/seed"
	"zazi.cool/undercover/internal/store"
	"zazi.cool/undercover/migrations"
	"zazi.cool/undercover/web"
)

// version is the git SHA of the build, injected with -ldflags "-X main.version=<sha>".
var version = "dev"

// shutdownTimeout bounds how long in-flight requests may take to finish on SIGTERM.
const shutdownTimeout = 10 * time.Second

// drainDelay is how long the server keeps accepting connections after SIGTERM. Traefik
// learns that the pod is gone asynchronously (EndpointSlice update): without this delay
// it can still route requests to a closed listener, which the user sees as a 502.
const drainDelay = 5 * time.Second

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
	st := store.New(pool)

	if err := prepareDatabase(ctx, pool, st); err != nil {
		return err
	}
	if len(args) > 0 && args[0] == "migrate" {
		return nil
	}

	static, err := web.Dist()
	if err != nil {
		return fmt.Errorf("load front: %w", err)
	}
	handler := api.NewHandler(st, api.Config{
		TrustProxy:      cfg.trustProxy,
		IPHashSalt:      cfg.ipHashSalt,
		Version:         version,
		SuggestionLimit: cfg.suggestionLimit,
		ReportLimit:     cfg.reportLimit,
		CheckLimit:      cfg.checkLimit,
		Static:          static,
	})
	return serve(ctx, ":"+cfg.port, handler)
}

func prepareDatabase(ctx context.Context, pool *pgxpool.Pool, st *store.Store) error {
	if err := migrate.Up(ctx, pool, migrations.FS); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	ps, err := seed.Pairs()
	if err != nil {
		return err
	}
	n, err := st.Seed(ctx, ps)
	if err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	slog.Info("database ready", "seed_pairs_added", n)
	return nil
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
		// Longer than the idle timeouts of the proxy in front (Traefik keeps upstream
		// connections 90 s), so the proxy never reuses a connection we just closed.
		IdleTimeout: 120 * time.Second,
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

	slog.Info("shutting down", "drain_delay", drainDelay.String())
	time.Sleep(drainDelay)
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
	port                                     string
	databaseURL                              string
	ipHashSalt                               string
	trustProxy                               bool
	suggestionLimit, reportLimit, checkLimit int
}

// loadConfig reads the configuration from environment variables (see .env.example).
func loadConfig() (config, error) {
	cfg := config{
		port:        getenv("PORT", "8080"),
		databaseURL: os.Getenv("DATABASE_URL"),
		ipHashSalt:  os.Getenv("IP_HASH_SALT"),
	}
	if cfg.databaseURL == "" {
		return config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.ipHashSalt == "" {
		return config{}, errors.New("IP_HASH_SALT is required")
	}

	var err error
	if cfg.trustProxy, err = strconv.ParseBool(getenv("TRUST_PROXY", "false")); err != nil {
		return config{}, fmt.Errorf("TRUST_PROXY: %w", err)
	}
	limits := []struct {
		name  string
		value *int
		def   string
	}{
		{"RATE_LIMIT_SUGGESTIONS", &cfg.suggestionLimit, "60"},
		{"RATE_LIMIT_REPORTS", &cfg.reportLimit, "90"},
		{"RATE_LIMIT_CHECKS", &cfg.checkLimit, "1200"},
	}
	for _, l := range limits {
		n, err := strconv.Atoi(getenv(l.name, l.def))
		if err != nil || n <= 0 {
			return config{}, fmt.Errorf("%s must be a positive integer", l.name)
		}
		*l.value = n
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
