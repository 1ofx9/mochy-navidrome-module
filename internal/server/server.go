package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Options struct {
	PublicURL           string
	DefaultNavidromeURL string
	Version             string
	Log                 *slog.Logger
	ArtCache            *ArtCache
}

type Server struct {
	Options
	artCache *ArtCache
	log      *slog.Logger
}

func New(o Options) *Server {
	log := o.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	cache := o.ArtCache
	if cache == nil {
		cache = NewArtCache(defaultArtMaxItems, defaultArtMaxBytes)
	}
	if o.Version == "" {
		o.Version = "1.2.1"
	}
	return &Server{
		Options:  o,
		artCache: cache,
		log:      log,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /manifest.json", s.serveManifest)
	mux.HandleFunc("GET /manifest", s.serveManifest)
	mux.HandleFunc("GET /search", s.serveSearch)
	mux.HandleFunc("GET /stream/{id}", s.serveStream)
	mux.HandleFunc("GET /audio/{id}", s.serveAudio)
	mux.HandleFunc("HEAD /audio/{id}", s.serveAudio)
	mux.HandleFunc("GET /art/{id}", s.serveArt)
	mux.HandleFunc("HEAD /art/{id}", s.serveArt)
	mux.HandleFunc("GET /lyrics", s.serveLyrics)
	mux.HandleFunc("GET /metadata", s.serveMetadata)
	mux.HandleFunc("GET /health", s.serveHealth)
	mux.HandleFunc("GET /{$}", s.serveRoot)

	handler := s.withSecurityHeaders(mux)
	handler = s.withCORS(handler)
	handler = s.withLogging(handler)
	handler = s.withRecovery(handler)

	return handler
}

func (s *Server) baseURL(r *http.Request) string {
	if s.PublicURL != "" {
		return strings.TrimRight(s.PublicURL, "/")
	}

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := r.Host
	if xfh := r.Header.Get("X-Forwarded-Host"); xfh != "" {
		host = xfh
	}
	return scheme + "://" + host
}

func (s *Server) withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cross-Origin-Resource-Policy", "cross-origin")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Range, If-Range, If-None-Match")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Accept-Ranges, Content-Length, ETag, Last-Modified")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	bytes      int64
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += int64(n)
	return n, err
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)

		if r.URL.Path == "/health" {
			return
		}

		status := rec.statusCode
		if status == 0 {
			status = http.StatusOK
		}

		var extra string
		if q := r.URL.Query().Get("q"); q != "" && r.URL.Path == "/search" {
			extra = fmt.Sprintf("q=%q", q)
		} else if strings.HasPrefix(r.URL.Path, "/stream/") {
			quality := r.URL.Query().Get("quality")
			if quality == "" {
				quality = "lossless"
			}
			extra = fmt.Sprintf("[resolve stream] (%s)", quality)
		} else if strings.HasPrefix(r.URL.Path, "/audio/") {
			quality := r.URL.Query().Get("quality")
			if quality == "" {
				quality = "lossless"
			}
			if status == http.StatusPartialContent {
				extra = fmt.Sprintf("[seek range] (%s)", quality)
			} else {
				extra = fmt.Sprintf("[audio stream] (%s)", quality)
			}
		}

		s.log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", strconv.Itoa(status),
			"bytes", formatBytes(rec.bytes),
			"duration", formatDuration(time.Since(start)),
			"extra", extra,
		)
	})
}

func formatBytes(b int64) string {
	if b <= 0 {
		return "0B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000.0)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func (s *Server) withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("panic recovered", "panic", rec)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
