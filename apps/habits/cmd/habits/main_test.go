package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// testHash is a valid bcrypt hash (of "correct horse battery").
const testHash = "$2a$04$ypDoACAigFuzC1hs5JqnDuTbGu.nshUtB0jTojUzhPI/vh3MghUyC"

func TestLoadConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("ADMIN_PASSWORD_HASH", testHash)
	t.Setenv("PORT", "")
	t.Setenv("BASE_URL", "")
	t.Setenv("TRUST_PROXY", "")
	t.Setenv("RATE_LIMIT_LOGIN", "")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.port != "8080" || cfg.baseURL != "https://habits.zazi.cool" || cfg.trustProxy || cfg.loginLimit != 10 {
		t.Errorf("defaults = %+v", cfg)
	}

	t.Setenv("PORT", "9000")
	t.Setenv("BASE_URL", "http://localhost:9000/")
	t.Setenv("TRUST_PROXY", "true")
	t.Setenv("RATE_LIMIT_LOGIN", "3")
	cfg, err = loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.port != "9000" || cfg.baseURL != "http://localhost:9000" || !cfg.trustProxy || cfg.loginLimit != 3 {
		t.Errorf("custom = %+v", cfg)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	tests := []struct{ name, key, value string }{
		{"no database", "DATABASE_URL", ""},
		{"no password hash", "ADMIN_PASSWORD_HASH", ""},
		{"bad password hash", "ADMIN_PASSWORD_HASH", "plaintext"},
		{"bad base url", "BASE_URL", "habits.zazi.cool"},
		{"bad port", "PORT", "http"},
		{"bad trust proxy", "TRUST_PROXY", "maybe"},
		{"zero rate limit", "RATE_LIMIT_LOGIN", "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://x")
			t.Setenv("ADMIN_PASSWORD_HASH", testHash)
			t.Setenv(tt.key, tt.value)
			if _, err := loadConfig(); err == nil {
				t.Error("want an error")
			}
		})
	}
}

func TestServeShutsDownWhenContextIsCancelled(t *testing.T) {
	// Reserve a free port, then release it for serve to bind.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, addr, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			io.WriteString(w, "ok")
		}))
	}()

	var resp *http.Response
	for range 50 {
		if resp, err = http.Get("http://" + addr); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("server never answered: %v", err)
	}
	resp.Body.Close()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("serve returned %v after cancel, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return after cancel")
	}
}
