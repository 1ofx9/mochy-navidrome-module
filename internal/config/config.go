package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port      int
	PublicURL string
	LogLevel  slog.Level
	LogFormat string
}

var logLevels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

func Load(getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}

	var problems []error
	fail := func(format string, args ...any) {
		problems = append(problems, fmt.Errorf(format, args...))
	}
	get := func(key, fallback string) string {
		if value := strings.TrimSpace(getenv(key)); value != "" {
			return value
		}
		return fallback
	}

	var cfg Config

	port, err := strconv.Atoi(get("PORT", "8090"))
	if err != nil || port < 1 || port > 65535 {
		fail("PORT must be between 1 and 65535")
	}
	cfg.Port = port
	cfg.PublicURL = strings.TrimRight(get("PUBLIC_URL", ""), "/")

	level, ok := logLevels[strings.ToLower(get("LOG_LEVEL", "info"))]
	if !ok {
		fail("LOG_LEVEL must be one of debug, info, warn, error")
	}
	cfg.LogLevel = level

	cfg.LogFormat = strings.ToLower(get("LOG_FORMAT", "text"))
	if cfg.LogFormat != "text" && cfg.LogFormat != "json" {
		fail("LOG_FORMAT must be text or json")
	}

	return cfg, errors.Join(problems...)
}
