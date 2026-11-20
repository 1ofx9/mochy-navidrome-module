package fakes

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

const (
	NavidromeUser     = "lyle"
	NavidromePassword = "correct horse"
)

type Navidrome struct {
	URL    string
	server *httptest.Server
}

func NewNavidrome(t testing.TB) *Navidrome {
	f := &Navidrome{}
	f.server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.server.Close)
	f.URL = f.server.URL
	return f
}

func (f *Navidrome) serve(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if !f.authorized(q) {
		if strings.HasSuffix(r.URL.Path, ".view") && !strings.Contains(r.URL.Path, "stream") && !strings.Contains(r.URL.Path, "getCoverArt") {
			writeSub(w, map[string]any{
				"status": "failed",
				"error":  map[string]any{"code": 40, "message": "Wrong username or password."},
			})
			return
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.URL.Path {
	case "/rest/stream.view":
		f.serveStream(w, r)
		return
	case "/rest/getCoverArt.view":
		id := q.Get("id")
		if id == "notfound" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("ETag", `"art-etag-123"`)
		w.Header().Set("Last-Modified", "Thu, 20 Nov 2026 12:00:00 GMT")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("FAKEIMAGEJPEGDATA"))
		return
	}

	if !strings.HasPrefix(r.URL.Path, "/rest/") {
		http.NotFound(w, r)
		return
	}

	method := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/rest/"), ".view")
	switch method {
	case "ping":
		writeSub(w, map[string]any{"status": "ok", "version": "1.16.1"})
	case "search3":
		query := strings.ToLower(q.Get("query"))
		var matches []map[string]any
		for _, s := range fakeSongs() {
			title := strings.ToLower(s["title"].(string))
			artist := strings.ToLower(s["artist"].(string))
			album := strings.ToLower(s["album"].(string))
			if query == "" || strings.Contains(title, query) || strings.Contains(artist, query) || strings.Contains(album, query) {
				matches = append(matches, s)
			}
		}
		writeSub(w, map[string]any{
			"status": "ok",
			"searchResult3": map[string]any{
				"song": matches,
			},
		})
	case "getSong":
		id := q.Get("id")
		for _, s := range fakeSongs() {
			if s["id"] == id {
				writeSub(w, map[string]any{"status": "ok", "song": s})
				return
			}
		}
		writeSub(w, map[string]any{
			"status": "failed",
			"error":  map[string]any{"code": 70, "message": "not found"},
		})
	case "getLyricsBySongId":
		id := q.Get("id")
		if id == "s-1" {
			writeSub(w, map[string]any{
				"status": "ok",
				"lyricsList": map[string]any{
					"structuredLyrics": []map[string]any{
						{
							"lang":   "eng",
							"synced": true,
							"line": []map[string]any{
								{"start": 12340, "value": "I've been walking down this road"},
								{"start": 15800, "value": "With the radio on"},
							},
						},
					},
				},
			})
			return
		}
		writeSub(w, map[string]any{
			"status": "failed",
			"error":  map[string]any{"code": 70, "message": "lyrics not found"},
		})
	case "getLyrics":
		title := strings.ToLower(q.Get("title"))
		if strings.Contains(title, "endless") {
			writeSub(w, map[string]any{
				"status": "ok",
				"lyrics": map[string]any{
					"artist":  "Limp Bizkit",
					"title":   "Endless Slaughter",
					"content": "Line one of slaughter\nLine two of slaughter",
				},
			})
			return
		}
		writeSub(w, map[string]any{
			"status": "failed",
			"error":  map[string]any{"code": 70, "message": "no lyrics found"},
		})
	default:
		writeSub(w, map[string]any{
			"status": "failed",
			"error":  map[string]any{"code": 0, "message": "unknown method: " + method},
		})
	}
}

func (f *Navidrome) serveStream(w http.ResponseWriter, r *http.Request) {
	data := []byte("FAKEAUDIOBYTESTREAMING0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	totalLen := len(data)

	w.Header().Set("Content-Type", "audio/flac")
	w.Header().Set("Accept-Ranges", "bytes")

	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		w.Header().Set("Content-Length", strconv.Itoa(totalLen))
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	if !strings.HasPrefix(rangeHeader, "bytes=") {
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(rangeSpec, "-")
	start, err := strconv.Atoi(parts[0])
	if err != nil || start >= totalLen {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", totalLen))
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	end := totalLen - 1
	if len(parts) > 1 && parts[1] != "" {
		if parsedEnd, err := strconv.Atoi(parts[1]); err == nil && parsedEnd < totalLen {
			end = parsedEnd
		}
	}

	length := end - start + 1
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, totalLen))
	w.Header().Set("Content-Length", strconv.Itoa(length))
	w.WriteHeader(http.StatusPartialContent)
	w.Write(data[start : end+1])
}

func (f *Navidrome) authorized(q map[string][]string) bool {
	get := func(k string) string {
		if v, ok := q[k]; ok && len(v) > 0 {
			return v[0]
		}
		return ""
	}
	if get("u") != NavidromeUser {
		return false
	}
	salt := get("s")
	want := get("t")
	sum := md5.Sum([]byte(NavidromePassword + salt))
	return hex.EncodeToString(sum[:]) == want
}

func writeSub(w http.ResponseWriter, inner map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"subsonic-response": inner})
}

func fakeSongs() []map[string]any {
	return []map[string]any{
		{
			"id":           "s-1",
			"title":        "New Religion",
			"album":        "Tell Me I'm Alive",
			"artist":       "All Time Low",
			"duration":     184,
			"suffix":       "flac",
			"bitRate":      1875,
			"samplingRate": 48000,
			"bitDepth":     24,
			"coverArt":     "al-1",
			"year":         2023,
			"genre":        "Alternative Rock",
		},
		{
			"id":           "s-2",
			"title":        "Endless Slaughter",
			"album":        "Endless Slaughter",
			"artist":       "Limp Bizkit",
			"duration":     336,
			"suffix":       "mp3",
			"bitRate":      320,
			"samplingRate": 44100,
			"bitDepth":     16,
			"coverArt":     "al-2",
			"year":         2014,
			"genre":        "Nu Metal",
		},
		{
			"id":           "s-3",
			"title":        "Small Town Girl",
			"album":        "Small Town Girl",
			"artist":       "Never Shout Never",
			"duration":     218,
			"suffix":       "m4a",
			"contentType":  "audio/mp4",
			"bitRate":      256,
			"samplingRate": 44100,
			"bitDepth":     16,
			"year":         2015,
			"genre":        "Acoustic",
		},
	}
}
