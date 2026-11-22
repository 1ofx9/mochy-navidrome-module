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
	if cfg.ModuleID != "mochy-navidrome" {
		t.Errorf("ModuleID = %q, want mochy-navidrome", cfg.ModuleID)
	}
	if cfg.ModuleName != "Navidrome" {
		t.Errorf("ModuleName = %q, want Navidrome", cfg.ModuleName)
	}
}

func TestConfigLoadCustom(t *testing.T) {
	env := map[string]string{
		"PORT":          "9000",
		"PUBLIC_URL":    "https://mochy.example.com/",
		"NAVIDROME_URL": "https://music.example.com/",
		"LOG_LEVEL":     "debug",
		"LOG_FORMAT":    "json",
		"MODULE_ID":     "navidrome-family",
		"MODULE_NAME":   "Navidrome Family",
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
	if cfg.ModuleID != "navidrome-family" {
		t.Errorf("ModuleID = %q, want navidrome-family", cfg.ModuleID)
	}
	if cfg.ModuleName != "Navidrome Family" {
		t.Errorf("ModuleName = %q, want 'Navidrome Family'", cfg.ModuleName)
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

func TestConfigMultiServerShorthand(t *testing.T) {
	env := map[string]string{
		"PUBLIC_URL": "https://mochy.subs-server.site",
		"SERVERS":    "music:Navidrome:https://music.subs-server.site,dolby:Navidrome (Dolby):https://dolby.subs-server.site",
	}
	cfg, err := Load(func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if len(cfg.Instances) != 2 {
		t.Fatalf("len(Instances) = %d, want 2", len(cfg.Instances))
	}

	inst1 := cfg.Instances[0]
	if inst1.Slug != "music" || inst1.ModuleName != "Navidrome" || inst1.DefaultNavidromeURL != "https://music.subs-server.site" || inst1.ModuleID != "mochy-navidrome-music" {
		t.Errorf("unexpected inst1: %+v", inst1)
	}

	inst2 := cfg.Instances[1]
	if inst2.Slug != "dolby" || inst2.ModuleName != "Navidrome (Dolby)" || inst2.DefaultNavidromeURL != "https://dolby.subs-server.site" || inst2.ModuleID != "mochy-navidrome-dolby" {
		t.Errorf("unexpected inst2: %+v", inst2)
	}
}

func TestConfigMultiServerEnvVars(t *testing.T) {
	env := map[string]string{
		"SERVERS":           "music,dolby",
		"SERVER_MUSIC_NAME": "Navidrome Standard",
		"SERVER_MUSIC_URL":  "https://music.subs-server.site",
		"SERVER_DOLBY_NAME": "Navidrome Atmos",
		"SERVER_DOLBY_URL":  "https://dolby.subs-server.site",
		"SERVER_DOLBY_ID":   "navidrome-custom-dolby",
	}
	cfg, err := Load(func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if len(cfg.Instances) != 2 {
		t.Fatalf("len(Instances) = %d, want 2", len(cfg.Instances))
	}

	inst1 := cfg.Instances[0]
	if inst1.Slug != "music" || inst1.ModuleName != "Navidrome Standard" || inst1.DefaultNavidromeURL != "https://music.subs-server.site" || inst1.ModuleID != "mochy-navidrome-music" {
		t.Errorf("unexpected inst1: %+v", inst1)
	}

	inst2 := cfg.Instances[1]
	if inst2.Slug != "dolby" || inst2.ModuleName != "Navidrome Atmos" || inst2.DefaultNavidromeURL != "https://dolby.subs-server.site" || inst2.ModuleID != "navidrome-custom-dolby" {
		t.Errorf("unexpected inst2: %+v", inst2)
	}
}

func TestConfigMultiServerJSON(t *testing.T) {
	env := map[string]string{
		"SERVERS": `[{"slug":"music","name":"Navidrome","url":"https://music.subs-server.site"},{"slug":"dolby","name":"Navidrome (Dolby)","url":"https://dolby.subs-server.site"}]`,
	}
	cfg, err := Load(func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if len(cfg.Instances) != 2 {
		t.Fatalf("len(Instances) = %d, want 2", len(cfg.Instances))
	}
	if cfg.Instances[0].Slug != "music" || cfg.Instances[1].Slug != "dolby" {
		t.Errorf("unexpected instances: %+v", cfg.Instances)
	}
}
