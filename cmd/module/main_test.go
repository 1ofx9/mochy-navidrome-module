package main

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/1ofx9/mochy-navidrome-module/internal/config"
)

func TestVersionFlag(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-version"}, func(string) string { return "" }, &buf)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(buf.String(), "mochy-navidrome-module v"+version) {
		t.Fatalf("unexpected version output: %s", buf.String())
	}
}

func TestHealthcheckFlagFailsWhenDown(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-healthcheck"}, func(k string) string {
		if k == "PORT" {
			return "65432"
		}
		return ""
	}, &buf)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 for unreachable server", code)
	}
}

func TestServeAndGracefulShutdown(t *testing.T) {
	cfg := config.Config{
		Port:      0,
		LogLevel:  -4,
		LogFormat: "text",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := newLogger("text", -4, ioDiscard{})

	started := make(chan net.Addr, 1)
	errCh := make(chan error, 1)

	go func() {
		errCh <- serve(ctx, cfg, log, func(addr net.Addr) {
			started <- addr
		})
	}()

	var addr net.Addr
	select {
	case addr = <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("server failed to start within timeout")
	}

	res, err := http.Get("http://" + addr.String() + "/health")
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve error on shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("graceful shutdown timed out")
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
