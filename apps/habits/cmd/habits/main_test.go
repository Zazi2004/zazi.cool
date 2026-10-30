package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := loadConfig(); err == nil {
		t.Error("loadConfig without DATABASE_URL: want an error")
	}

	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("PORT", "")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.port != "8080" {
		t.Errorf("default port = %q, want 8080", cfg.port)
	}

	t.Setenv("PORT", "9000")
	if cfg, err := loadConfig(); err != nil || cfg.port != "9000" {
		t.Errorf("PORT=9000: got %q, %v", cfg.port, err)
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
