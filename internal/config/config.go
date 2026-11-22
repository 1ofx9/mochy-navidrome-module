package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type ServerInstance struct {
	Slug                string `json:"slug"`
	ModuleID            string `json:"id,omitempty"`
	ModuleName          string `json:"name,omitempty"`
	DefaultNavidromeURL string `json:"url,omitempty"`
}

type Config struct {
	Port                int
	PublicURL           string
	DefaultNavidromeURL string
	ModuleID            string
	ModuleName          string
	LogLevel            slog.Level
	LogFormat           string
	Instances           []ServerInstance
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

	navURL := get("NAVIDROME_URL", "")
	if navURL == "" {
		navURL = get("DEFAULT_NAVIDROME_URL", "")
	}
	cfg.DefaultNavidromeURL = strings.TrimRight(navURL, "/")

	cfg.ModuleID = get("MODULE_ID", "mochy-navidrome")
	cfg.ModuleName = get("MODULE_NAME", "Navidrome")

	level, ok := logLevels[strings.ToLower(get("LOG_LEVEL", "info"))]
	if !ok {
		fail("LOG_LEVEL must be one of debug, info, warn, error")
	}
	cfg.LogLevel = level

	cfg.LogFormat = strings.ToLower(get("LOG_FORMAT", "text"))
	if cfg.LogFormat != "text" && cfg.LogFormat != "json" {
		fail("LOG_FORMAT must be text or json")
	}

	instances, err := parseServerInstances(get("SERVERS", get("INSTANCES", "")), getenv)
	if err != nil {
		fail("failed to parse multi-server configuration: %v", err)
	}
	cfg.Instances = instances

	return cfg, errors.Join(problems...)
}

func parseServerInstances(raw string, getenv func(string) string) ([]ServerInstance, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	if strings.HasPrefix(raw, "[") {
		var list []ServerInstance
		if err := json.Unmarshal([]byte(raw), &list); err != nil {
			return nil, fmt.Errorf("invalid SERVERS json: %w", err)
		}
		for i := range list {
			list[i].Slug = cleanSlug(list[i].Slug)
			if list[i].Slug == "" {
				return nil, fmt.Errorf("instance slug cannot be empty")
			}
			if list[i].ModuleID == "" {
				list[i].ModuleID = "mochy-navidrome-" + list[i].Slug
			}
			if list[i].ModuleName == "" {
				list[i].ModuleName = defaultNameFromSlug(list[i].Slug)
			}
			list[i].DefaultNavidromeURL = strings.TrimRight(list[i].DefaultNavidromeURL, "/")
		}
		return list, nil
	}

	entries := strings.Split(raw, ",")
	var instances []ServerInstance

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		var slug, name, navURL string
		if idx := strings.Index(entry, "://"); idx > 0 {
			schemeColon := strings.LastIndex(entry[:idx], ":")
			if schemeColon >= 0 {
				navURL = strings.TrimRight(strings.TrimSpace(entry[schemeColon+1:]), "/")
				meta := entry[:schemeColon]
				if s, n, ok := strings.Cut(meta, ":"); ok {
					slug = s
					name = n
				} else {
					slug = meta
				}
			}
		} else {
			if s, n, ok := strings.Cut(entry, ":"); ok {
				slug = s
				name = n
			} else {
				slug = entry
			}
		}

		slug = cleanSlug(slug)
		if slug == "" {
			continue
		}

		upper := strings.ToUpper(strings.ReplaceAll(slug, "-", "_"))
		if navURL == "" {
			navURL = getenv("SERVER_" + upper + "_URL")
			if navURL == "" {
				navURL = getenv("INSTANCE_" + upper + "_URL")
			}
			navURL = strings.TrimRight(strings.TrimSpace(navURL), "/")
		}

		if name == "" {
			name = getenv("SERVER_" + upper + "_NAME")
			if name == "" {
				name = getenv("INSTANCE_" + upper + "_NAME")
			}
			name = strings.TrimSpace(name)
		}

		moduleID := getenv("SERVER_" + upper + "_ID")
		if moduleID == "" {
			moduleID = getenv("INSTANCE_" + upper + "_ID")
		}
		moduleID = strings.TrimSpace(moduleID)
		if moduleID == "" {
			moduleID = "mochy-navidrome-" + slug
		}

		if name == "" {
			name = defaultNameFromSlug(slug)
		}

		instances = append(instances, ServerInstance{
			Slug:                slug,
			ModuleID:            moduleID,
			ModuleName:          name,
			DefaultNavidromeURL: navURL,
		})
	}

	return instances, nil
}

func cleanSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Trim(s, "/")
	return s
}

func defaultNameFromSlug(slug string) string {
	if slug == "" {
		return "Navidrome"
	}
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return "Navidrome (" + strings.Join(parts, " ") + ")"
}
