package config

import (
	"log/slog"
	"testing"
)

func TestConfigLoadDefaults(t *testing.T) {
	cfg, err := Load(func(string) string { return "" })
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if cfg.Port != 8090 {
		t.Errorf("Port = %d, want 8090", cfg.Port)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want Info", cfg.LogLevel)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("LogFormat = %q, want text", cfg.LogFormat)
	}
}

func TestConfigLoadCustom(t *testing.T) {
	env := map[string]string{
		"PORT":          "9000",
		"PUBLIC_URL":    "https://mochy.example.com/",
		"NAVIDROME_URL": "https://music.example.com/",
		"LOG_LEVEL":     "debug",
		"LOG_FORMAT":    "json",
	}
	cfg, err := Load(func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if cfg.Port != 9000 {
		t.Errorf("Port = %d, want 9000", cfg.Port)
	}
	if cfg.PublicURL != "https://mochy.example.com" {
		t.Errorf("PublicURL = %q, want 'https://mochy.example.com'", cfg.PublicURL)
	}
	if cfg.DefaultNavidromeURL != "https://music.example.com" {
		t.Errorf("DefaultNavidromeURL = %q, want 'https://music.example.com'", cfg.DefaultNavidromeURL)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want Debug", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want json", cfg.LogFormat)
	}
}

func TestConfigErrors(t *testing.T) {
	_, err := Load(func(k string) string {
		if k == "PORT" {
			return "invalid"
		}
		return ""
	})
	if err == nil {
		t.Errorf("expected error for invalid PORT")
	}

	_, err = Load(func(k string) string {
		if k == "LOG_LEVEL" {
			return "verbose"
		}
		return ""
	})
	if err == nil {
		t.Errorf("expected error for invalid LOG_LEVEL")
	}
}
