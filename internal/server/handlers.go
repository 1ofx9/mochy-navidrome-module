package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/1ofx9/mochy-navidrome-module/internal/subsonic"
)

var (
	forwardedAudioHeaders = []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "ETag", "Last-Modified"}
)

type manifestResponse struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Version     string              `json:"version"`
	Description string              `json:"description"`
	Author      string              `json:"author"`
	Icon        string              `json:"icon,omitempty"`
	Type        string              `json:"type"`
	Resources   []string            `json:"resources"`
	Settings    []settingDefinition `json:"settings"`
}

type settingDefinition struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Hint        string   `json:"hint,omitempty"`
	Description string   `json:"description,omitempty"`
	Default     string   `json:"default,omitempty"`
	Options     []option `json:"options,omitempty"`
}

type option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type trackItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Duration int    `json:"duration"`
	ISRC     string `json:"isrc,omitempty"`
}

type searchResponse struct {
	Tracks []trackItem `json:"tracks"`
}

type streamResponse struct {
	URL        string `json:"url"`
	Format     string `json:"format"`
	Codec      string `json:"codec"`
	Quality    string `json:"quality"`
	SampleRate int    `json:"sampleRate,omitempty"`
	BitDepth   int    `json:"bitDepth,omitempty"`
}

type lyricsResponse struct {
	SyncedLyrics string `json:"syncedLyrics,omitempty"`
	PlainLyrics  string `json:"plainLyrics,omitempty"`
}

type metadataResponse struct {
	Album       string   `json:"album,omitempty"`
	ReleaseDate string   `json:"releaseDate,omitempty"`
	Genres      []string `json:"genres,omitempty"`
	CoverURL    string   `json:"coverUrl,omitempty"`
}

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func (s *Server) serveRoot(w http.ResponseWriter, r *http.Request) {
	s.serveManifest(w, r)
}

func (s *Server) serveManifest(w http.ResponseWriter, r *http.Request) {
	var settings []settingDefinition
	if s.DefaultNavidromeURL == "" {
		settings = append(settings, settingDefinition{
			Key:         "navidrome_url",
			Label:       "Server URL",
			Type:        "text",
			Hint:        "http://192.168.1.100:4533",
			Description: "Navidrome server address",
			Default:     "",
		})
	}
	settings = append(settings,
		settingDefinition{
			Key:         "navidrome_user",
			Label:       "Username",
			Type:        "text",
			Hint:        "admin",
			Description: "Navidrome username",
			Default:     "",
		},
		settingDefinition{
			Key:         "navidrome_password",
			Label:       "Password",
			Type:        "text",
			Hint:        "password",
			Description: "Navidrome password",
			Default:     "",
		},
		settingDefinition{
			Key:         "quality",
			Label:       "Audio Quality",
			Type:        "select",
			Default:     "lossless",
			Description: "Preferred audio quality",
			Options: []option{
				{Value: "lossless", Label: "Lossless (FLAC/ALAC)"},
				{Value: "hires", Label: "Hi-Res"},
				{Value: "high", Label: "High (320 kbps)"},
				{Value: "low", Label: "Low (128 kbps)"},
			},
		},
	)

	manifest := manifestResponse{
		ID:          "mochy-navidrome",
		Name:        "Navidrome",
		Version:     s.Version,
		Description: "Stream music directly from your Navidrome server",
		Author:      "1ofx9",
		Icon:        "https://www.navidrome.org/favicon.png",
		Type:        "web",
		Resources:   []string{"search", "stream", "lyrics", "metadata"},
		Settings:    settings,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(manifest)
}

func (s *Server) getSubsonicClient(r *http.Request) (*subsonic.Client, error) {
	q := r.URL.Query()
	baseURL := strings.TrimSpace(q.Get("navidrome_url"))
	if baseURL == "" {
		baseURL = s.DefaultNavidromeURL
	}
	user := strings.TrimSpace(q.Get("navidrome_user"))
	pass := q.Get("navidrome_password")

	if baseURL == "" || user == "" {
		if u, p, ok := r.BasicAuth(); ok {
			if user == "" {
				user = u
			}
			if pass == "" {
				pass = p
			}
		}
	}

	if baseURL == "" || user == "" {
		return nil, errors.New("navidrome_url and navidrome_user are required")
	}

	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, errors.New("navidrome_url must be a valid http or https URL")
	}

	return subsonic.New(baseURL, user, pass, nil), nil
}

func (s *Server) serveSearch(w http.ResponseWriter, r *http.Request) {
	client, err := s.getSubsonicClient(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeJSON(w, http.StatusOK, searchResponse{Tracks: []trackItem{}})
		return
	}

	songs, err := client.Search(r.Context(), query, 10)
	if err != nil {
		s.log.Error("search failed", "query", query, "error", err)
		writeJSON(w, http.StatusOK, searchResponse{Tracks: []trackItem{}})
		return
	}

	tracks := make([]trackItem, 0, len(songs))
	for _, song := range songs {
		tracks = append(tracks, trackItem{
			ID:       song.ID,
			Title:    song.Title,
			Artist:   song.EffectiveArtist(),
			Album:    song.Album,
			Duration: song.Duration,
		})
	}

	writeJSON(w, http.StatusOK, searchResponse{Tracks: tracks})
}

func (s *Server) serveStream(w http.ResponseWriter, r *http.Request) {
	client, err := s.getSubsonicClient(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	id := pathID(r, "/stream/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "song id required"})
		return
	}

	song, err := client.GetSong(r.Context(), id)
	if err != nil {
		if errors.Is(err, subsonic.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "song not found"})
			return
		}
		s.log.Error("getSong failed", "id", id, "error", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "upstream error"})
		return
	}

	q := r.URL.Query()
	qualityPref := q.Get("quality")
	maxBitRate := qualityToMaxBitRate(qualityPref)

	audioParams := url.Values{
		"navidrome_url":      {q.Get("navidrome_url")},
		"navidrome_user":     {q.Get("navidrome_user")},
		"navidrome_password": {q.Get("navidrome_password")},
	}
	if qualityPref != "" {
		audioParams.Set("quality", qualityPref)
	}

	audioURL := s.baseURL(r) + "/audio/" + url.PathEscape(id) + "?" + audioParams.Encode()

	resp := streamResponse{
		URL:        audioURL,
		Format:     strings.ToLower(song.Suffix),
		Codec:      song.Codec(),
		Quality:    song.QualityLabel(),
		SampleRate: song.SamplingRate,
		BitDepth:   song.BitDepth,
	}

	if maxBitRate == 128 {
		resp.Quality = "Low"
		resp.Format = "mp3"
		resp.Codec = "mp3"
	} else if maxBitRate == 320 {
		resp.Quality = "High"
		resp.Format = "mp3"
		resp.Codec = "mp3"
	}

	writeJSON(w, http.StatusOK, resp)
}

func qualityToMaxBitRate(q string) int {
	switch strings.ToLower(strings.TrimSpace(q)) {
	case "low", "128":
		return 128
	case "high", "320":
		return 320
	default:
		return 0
	}
}

func (s *Server) serveAudio(w http.ResponseWriter, r *http.Request) {
	client, err := s.getSubsonicClient(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := pathID(r, "/audio/")
	if id == "" {
		http.Error(w, "missing song id", http.StatusBadRequest)
		return
	}

	maxBitRate := qualityToMaxBitRate(r.URL.Query().Get("quality"))
	upstream, err := client.OpenStream(r.Context(), id, maxBitRate, r.Header)
	if err != nil {
		if errors.Is(err, subsonic.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		s.log.Error("upstream stream open failed", "id", id, "error", err)
		http.Error(w, "upstream stream error", http.StatusBadGateway)
		return
	}
	defer upstream.Body.Close()

	if upstream.StatusCode != http.StatusOK && upstream.StatusCode != http.StatusPartialContent {
		w.WriteHeader(upstream.StatusCode)
		return
	}

	for _, name := range forwardedAudioHeaders {
		if v := upstream.Header.Get(name); v != "" {
			w.Header().Set(name, v)
		}
	}

	w.WriteHeader(upstream.StatusCode)
	if r.Method != http.MethodHead {
		_, _ = io.Copy(w, upstream.Body)
	}
}

func (s *Server) serveArt(w http.ResponseWriter, r *http.Request) {
	client, err := s.getSubsonicClient(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := pathID(r, "/art/")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	baseURL := strings.TrimSpace(r.URL.Query().Get("navidrome_url"))
	cacheKey := baseURL + ":" + id

	if s.artCache != nil {
		if item, ok := s.artCache.Get(cacheKey); ok {
			serveCachedArt(w, r, item)
			return
		}
	}

	body, ct, etag, lastMod, err := client.FetchArt(r.Context(), id, 600)
	if err != nil {
		if errors.Is(err, subsonic.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		s.log.Error("fetch art failed", "id", id, "error", err)
		http.Error(w, "failed fetching art", http.StatusBadGateway)
		return
	}

	if s.artCache != nil && int64(len(body)) <= maxCachedItemSize {
		item := s.artCache.Put(cacheKey, body, ct, etag, lastMod)
		if item != nil {
			serveCachedArt(w, r, item)
			return
		}
	}

	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	if etag != "" {
		w.Header().Set("ETag", etag)
	}
	if lastMod != "" {
		w.Header().Set("Last-Modified", lastMod)
	}
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(body)
	}
}

func (s *Server) serveLyrics(w http.ResponseWriter, r *http.Request) {
	client, err := s.getSubsonicClient(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	q := r.URL.Query()
	id := q.Get("id")
	artist := q.Get("artist")
	title := q.Get("title")

	synced, plain, err := client.GetLyrics(r.Context(), id, artist, title)
	if err != nil || (synced == "" && plain == "") {
		writeJSON(w, http.StatusOK, lyricsResponse{})
		return
	}

	writeJSON(w, http.StatusOK, lyricsResponse{
		SyncedLyrics: synced,
		PlainLyrics:  plain,
	})
}

func (s *Server) serveMetadata(w http.ResponseWriter, r *http.Request) {
	client, err := s.getSubsonicClient(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		writeJSON(w, http.StatusOK, metadataResponse{})
		return
	}

	song, err := client.GetSong(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusOK, metadataResponse{})
		return
	}

	resp := metadataResponse{
		Album: song.Album,
	}
	if song.Year > 0 {
		resp.ReleaseDate = strconv.Itoa(song.Year)
	}
	if song.Genre != "" {
		resp.Genres = []string{song.Genre}
	}
	if song.CoverArt != "" {
		artParams := url.Values{
			"navidrome_url":      {q.Get("navidrome_url")},
			"navidrome_user":     {q.Get("navidrome_user")},
			"navidrome_password": {q.Get("navidrome_password")},
		}
		resp.CoverURL = s.baseURL(r) + "/art/" + url.PathEscape(song.CoverArt) + "?" + artParams.Encode()
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) serveHealth(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("navidrome_user") != "" || r.URL.Query().Get("navidrome_url") != "" {
		client, err := s.getSubsonicClient(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"status":  "error",
				"version": s.Version,
				"error":   err.Error(),
			})
			return
		}
		if _, err := client.Ping(r.Context()); err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"status":  "error",
				"version": s.Version,
				"error":   "invalid navidrome credentials or unreachable server",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"status":    "ok",
			"version":   s.Version,
			"navidrome": "connected",
		})
		return
	}

	writeJSON(w, http.StatusOK, healthResponse{
		Status:  "ok",
		Version: s.Version,
	})
}

func pathID(r *http.Request, prefix string) string {
	id := r.PathValue("id")
	if id == "" {
		id = strings.TrimPrefix(r.URL.Path, prefix)
	}
	id = strings.TrimSpace(id)
	id = strings.ReplaceAll(id, "\r", "")
	id = strings.ReplaceAll(id, "\n", "")
	return id
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
