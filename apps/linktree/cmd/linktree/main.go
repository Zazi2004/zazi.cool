// Command linktree serves the link pages of link.zazi.cool and their admin.
//
// Usage:
//
//	linktree           apply migrations, then serve HTTP
//	linktree migrate   apply migrations, then exit
package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
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

	"zazi.cool/linktree/internal/api"
	"zazi.cool/linktree/internal/auth"
	"zazi.cool/linktree/internal/migrate"
	"zazi.cool/linktree/internal/pages"
	"zazi.cool/linktree/internal/render"
	"zazi.cool/linktree/internal/store"
	"zazi.cool/linktree/migrations"
	"zazi.cool/linktree/web"
)

// version is the git SHA of the build, injected with -ldflags "-X main.version=<sha>".
var version = "dev"

const (
	// shutdownTimeout bounds how long in-flight requests may take to finish on SIGTERM.
	shutdownTimeout = 10 * time.Second
	// cleanupEvery is the period of the clean-up of orphan images and old sessions.
	cleanupEvery = time.Hour
	// orphanDelay leaves time to save a page after uploading its images.
	orphanDelay = 24 * time.Hour
)

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

	assets, static, err := loadFront(cfg.viteDevURL)
	if err != nil {
		return err
	}
	shapes, err := pages.LoadShapes(static)
	if err != nil {
		return err
	}
	renderer, err := render.New(assets)
	if err != nil {
		return err
	}
	authManager, err := auth.NewManager(cfg.passwordHash, st, strings.HasPrefix(cfg.publicBaseURL, "https://"))
	if err != nil {
		return err
	}

	handler := api.NewHandler(api.Config{
		Store:          st,
		Auth:           authManager,
		Renderer:       renderer,
		Static:         static,
		Shapes:         shapes,
		PublicBaseURL:  cfg.publicBaseURL,
		DevURL:         cfg.viteDevURL,
		TrustProxy:     cfg.trustProxy,
		LoginLimit:     cfg.loginLimit,
		MaxUploadBytes: int64(cfg.maxUploadMB) << 20,
		Version:        version,
	})
	slog.Info("ready", "shapes", len(shapes), "dev", cfg.viteDevURL != "")

	// The clean-up runs in its own goroutine and stops with ctx; serve waits for
	// the end of the requests in flight before run returns and closes the pool.
	go cleanUp(ctx, st)
	return serve(ctx, ":"+cfg.port, handler)
}

// loadFront returns where the bundles come from and the files served as they are:
// the build embedded in the binary, or, in development, the Vite dev server and
// web/public on disk (`make dev` runs from apps/linktree).
func loadFront(viteDevURL string) (*render.Assets, fs.FS, error) {
	if viteDevURL != "" {
		return render.FromDevServer(viteDevURL), os.DirFS("web/public"), nil
	}
	dist, err := web.Dist()
	if err != nil {
		return nil, nil, fmt.Errorf("load front: %w", err)
	}
	assets, err := render.FromManifest(dist)
	if err != nil {
		return nil, nil, err
	}
	return assets, dist, nil
}

// cleanUp periodically deletes orphan images and expired sessions, until ctx ends.
// Both deletions are idempotent: several replicas could run them safely.
func cleanUp(ctx context.Context, st *store.Store) {
	ticker := time.NewTicker(cleanupEvery)
	defer ticker.Stop()
	for {
		now := time.Now()
		if n, err := st.DeleteOrphanImages(ctx, now.Add(-orphanDelay)); err != nil {
			slog.Error("clean up images", "error", err)
		} else if n > 0 {
			slog.Info("orphan images deleted", "count", n)
		}
		if _, err := st.DeleteExpiredSessions(ctx, now); err != nil {
			slog.Error("clean up sessions", "error", err)
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
		ReadTimeout:       30 * time.Second, // image uploads from a phone can be slow
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

type config struct {
	port          string
	databaseURL   string
	passwordHash  string
	publicBaseURL string
	viteDevURL    string
	trustProxy    bool
	loginLimit    int
	maxUploadMB   int
}

// loadConfig reads the configuration from environment variables (see .env.example).
func loadConfig() (config, error) {
	cfg := config{
		port:          getenv("PORT", "8080"),
		databaseURL:   os.Getenv("DATABASE_URL"),
		passwordHash:  os.Getenv("ADMIN_PASSWORD_HASH"),
		publicBaseURL: strings.TrimSuffix(getenv("PUBLIC_BASE_URL", "http://localhost:8080"), "/"),
		viteDevURL:    os.Getenv("VITE_DEV_URL"),
	}
	if cfg.databaseURL == "" {
		return config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.passwordHash == "" {
		return config{}, errors.New("ADMIN_PASSWORD_HASH is required (see `make hash-password`)")
	}
	if u, err := url.Parse(cfg.publicBaseURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return config{}, fmt.Errorf("PUBLIC_BASE_URL must be an absolute http(s) URL, got %q", cfg.publicBaseURL)
	}

	var err error
	if cfg.trustProxy, err = strconv.ParseBool(getenv("TRUST_PROXY", "false")); err != nil {
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
		n, err := strconv.Atoi(getenv(v.name, v.def))
		if err != nil || n <= 0 {
			return config{}, fmt.Errorf("%s must be a positive integer", v.name)
		}
		*v.value = n
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
