package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/1ofx9/mochy-navidrome-module/internal/config"
	"github.com/1ofx9/mochy-navidrome-module/internal/server"
)

var version = "1.2.2"

const shutdownGrace = 10 * time.Second

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stderr))
}

func run(args []string, getenv func(string) string, stderr io.Writer) int {
	flags := flag.NewFlagSet("module", flag.ContinueOnError)
	flags.SetOutput(stderr)

	probe := flags.Bool("healthcheck", false, "probe /health on local port, exit 0 or 1")
	showVersion := flags.Bool("version", false, "print version and exit")
	portFlag := flags.Int("port", 0, "server port (overrides PORT env)")

	if err := flags.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintf(stderr, "mochy-navidrome-module v%s\n", version)
		return 0
	}

	if *probe {
		port := getenv("PORT")
		if port == "" {
			port = "8090"
		}
		return healthcheck("http://127.0.0.1:" + port + "/health")
	}

	cfg, err := config.Load(getenv)
	if err != nil {
		fmt.Fprintf(stderr, "configuration error: %v\n", err)
		return 1
	}
	if *portFlag > 0 {
		cfg.Port = *portFlag
	}

	log := newLogger(cfg.LogFormat, cfg.LogLevel, stderr)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	if err := serve(ctx, cfg, log, func(net.Addr) {}); err != nil {
		log.Error("server stopped with error", "error", err)
		return 1
	}

	return 0
}

func newLogger(format string, level slog.Level, w io.Writer) *slog.Logger {
	if format == "json" {
		opts := &slog.HandlerOptions{Level: level}
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(&prettyHandler{w: w, level: level})
}

type prettyHandler struct {
	w     io.Writer
	level slog.Level
	attrs []slog.Attr
}

func (h *prettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	combined := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	combined = append(combined, h.attrs...)
	combined = append(combined, attrs...)
	return &prettyHandler{w: h.w, level: h.level, attrs: combined}
}

func (h *prettyHandler) WithGroup(_ string) slog.Handler {
	return h
}

func (h *prettyHandler) Handle(_ context.Context, r slog.Record) error {
	timeStr := r.Time.Format("15:04:05")
	levelStr := "INF"
	switch {
	case r.Level >= slog.LevelError:
		levelStr = "ERR"
	case r.Level >= slog.LevelWarn:
		levelStr = "WRN"
	case r.Level < slog.LevelInfo:
		levelStr = "DBG"
	}

	if r.Message == "request" {
		var method, path, status, duration, size, extra string
		r.Attrs(func(a slog.Attr) bool {
			switch a.Key {
			case "method":
				method = a.Value.String()
			case "path":
				path = a.Value.String()
			case "status":
				status = a.Value.String()
			case "duration":
				duration = a.Value.String()
			case "bytes":
				size = a.Value.String()
			case "extra":
				extra = a.Value.String()
			}
			return true
		})

		statusLabel := status
		switch status {
		case "200":
			statusLabel = "200 OK"
		case "206":
			statusLabel = "206 Partial"
		case "304":
			statusLabel = "304 Cache"
		case "400":
			statusLabel = "400 BadReq"
		case "401":
			statusLabel = "401 Unauth"
		case "404":
			statusLabel = "404 NotFound"
		case "500":
			statusLabel = "500 Err"
		case "502":
			statusLabel = "502 BadGW"
		}
		line := fmt.Sprintf("  %s  %s  %-4s %-34s %-11s %7s  %s", timeStr, levelStr, method, path, statusLabel, size, duration)
		if extra != "" {
			line += "  " + extra
		}
		_, err := fmt.Fprintln(h.w, line)
		return err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "  %s  %s  %s", timeStr, levelStr, r.Message)
	for _, a := range h.attrs {
		fmt.Fprintf(&b, " %s=%v", a.Key, a.Value.Any())
	}
	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(&b, " %s=%v", a.Key, a.Value.Any())
		return true
	})
	b.WriteByte('\n')
	_, err := io.WriteString(h.w, b.String())
	return err
}

func healthcheck(url string) int {
	client := &http.Client{Timeout: 3 * time.Second}
	res, err := client.Get(url)
	if err != nil {
		return 1
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func serve(ctx context.Context, cfg config.Config, log *slog.Logger, listening func(net.Addr)) error {
	log.Info("starting mochy-navidrome-module",
		"version", version,
		"port", cfg.Port,
		"public_url", cfg.PublicURL,
		"log_level", cfg.LogLevel.String(),
	)

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(cfg.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", cfg.Port, err)
	}
	defer listener.Close()

	if cfg.DefaultNavidromeURL == "" {
		if discovered := probeNavidrome("http://navidrome:4533", "http://127.0.0.1:4533"); discovered != "" {
			cfg.DefaultNavidromeURL = discovered
			log.Info("auto-discovered navidrome server", "url", discovered)
		}
	}

	srvHandler := server.New(server.Options{
		PublicURL:           cfg.PublicURL,
		DefaultNavidromeURL: cfg.DefaultNavidromeURL,
		Version:             version,
		Log:                 log,
	})

	srv := &http.Server{
		Handler:           srvHandler.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	failed := make(chan error, 1)
	go func() {
		failed <- srv.Serve(listener)
	}()

	manifestLocal := fmt.Sprintf("http://localhost:%d/manifest.json", cfg.Port)
	lanIP := localLANIP()
	var manifestLAN string
	if lanIP != "" {
		manifestLAN = fmt.Sprintf("http://%s:%d/manifest.json", lanIP, cfg.Port)
	}

	if cfg.LogFormat != "json" {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  ▲  MOCHY NAVIDROME MODULE  v%s\n\n", version)
		fmt.Fprintf(os.Stderr, "  ➜  Local:     %s\n", manifestLocal)
		if manifestLAN != "" {
			fmt.Fprintf(os.Stderr, "  ➜  Network:   %s\n", manifestLAN)
		}
		if cfg.DefaultNavidromeURL != "" {
			fmt.Fprintf(os.Stderr, "  ➜  Navidrome: %s\n", cfg.DefaultNavidromeURL)
		}
		if cfg.PublicURL != "" {
			fmt.Fprintf(os.Stderr, "  ➜  Public:    %s/manifest.json\n", strings.TrimRight(cfg.PublicURL, "/"))
		} else {
			fmt.Fprintf(os.Stderr, "  ➜  Public:    https://<your-domain>/manifest.json\n")
		}
		fmt.Fprintf(os.Stderr, "\n  ➜  Mochy:     Settings → Modules → + Add Module\n\n")
	}

	attrs := []any{
		"addr", listener.Addr().String(),
		"local_manifest", manifestLocal,
	}
	if manifestLAN != "" {
		attrs = append(attrs, "lan_manifest", manifestLAN)
	}
	if cfg.DefaultNavidromeURL != "" {
		attrs = append(attrs, "navidrome_url", cfg.DefaultNavidromeURL)
	}
	if cfg.PublicURL != "" {
		attrs = append(attrs, "public_manifest", strings.TrimRight(cfg.PublicURL, "/")+"/manifest.json")
	}
	log.Info("module listening", attrs...)
	listening(listener.Addr())

	select {
	case err := <-failed:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down module")
	graceCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()

	if err := srv.Shutdown(graceCtx); err != nil {
		_ = srv.Close()
	}

	if err := <-failed; !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func localLANIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		if localAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			ip := localAddr.IP
			if !ip.IsLoopback() && ip.To4() != nil {
				return ip.String()
			}
		}
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip := ipnet.IP.To4(); ip != nil {
				return ip.String()
			}
		}
	}
	return ""
}

func probeNavidrome(candidates ...string) string {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	for _, target := range candidates {
		reqURL := strings.TrimRight(target, "/") + "/rest/ping.view?f=json"
		res, err := client.Get(reqURL)
		if err == nil {
			_ = res.Body.Close()
			if res.StatusCode > 0 {
				return target
			}
		}
	}
	return ""
}
