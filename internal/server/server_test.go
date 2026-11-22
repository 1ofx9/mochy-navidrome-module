package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/1ofx9/mochy-navidrome-module/internal/fakes"
)

type testRig struct {
	fake   *fakes.Navidrome
	server *Server
}

func newTestRig(t testing.TB) *testRig {
	fake := fakes.NewNavidrome(t)
	srv := New(Options{
		PublicURL: "http://localhost:8090",
		Version:   "1.0.0",
	})
	return &testRig{
		fake:   fake,
		server: srv,
	}
}

func (r *testRig) credQuery() url.Values {
	return url.Values{
		"navidrome_url":      {r.fake.URL},
		"navidrome_user":     {fakes.NavidromeUser},
		"navidrome_password": {fakes.NavidromePassword},
	}
}

func TestManifest(t *testing.T) {
	rig := newTestRig(t)
	handler := rig.server.Handler()

	for _, path := range []string{"/manifest.json", "/manifest"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, rec.Code)
		}

		var m manifestResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
			t.Fatalf("%s failed to parse manifest json: %v", path, err)
		}

		if m.ID != "mochy-navidrome" || m.Name != "Navidrome" || m.Type != "web" {
			t.Fatalf("unexpected manifest fields: %+v", m)
		}
		if m.Icon != "http://localhost:8090/icon.png" || m.Logo != "http://localhost:8090/icon.png" {
			t.Fatalf("unexpected icon/logo fields: icon=%q, logo=%q", m.Icon, m.Logo)
		}
		if len(m.Resources) != 4 {
			t.Fatalf("resources length = %d, want 4", len(m.Resources))
		}
		if len(m.Settings) != 4 {
			t.Fatalf("settings length = %d, want 4", len(m.Settings))
		}
		for _, s := range m.Settings {
			if s.Key == "navidrome_password" {
				if s.Type != "password" || s.InputType != "password" || s.Format != "password" || !s.Secure || !s.Secret || !s.Password || !s.Masked {
					t.Fatalf("expected password setting with masking attributes, got: %+v", s)
				}
			}
		}
	}
}

func TestManifestWithDefaultNavidromeURLHidesSetting(t *testing.T) {
	srv := New(Options{
		PublicURL:           "http://localhost:8090",
		DefaultNavidromeURL: "http://navidrome:4533",
		Version:             "1.1.0",
	})
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/manifest.json", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var m manifestResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}
	if len(m.Settings) != 3 {
		t.Fatalf("settings length = %d, want 3 (navidrome_url must be hidden)", len(m.Settings))
	}
	for _, s := range m.Settings {
		if s.Key == "navidrome_url" {
			t.Fatalf("navidrome_url setting should be hidden when DefaultNavidromeURL is configured")
		}
	}
}

func TestSearchWithDefaultNavidromeURL(t *testing.T) {
	fake := fakes.NewNavidrome(t)
	srv := New(Options{
		DefaultNavidromeURL: fake.URL,
	})
	handler := srv.Handler()

	q := url.Values{
		"navidrome_user":     {fakes.NavidromeUser},
		"navidrome_password": {fakes.NavidromePassword},
		"q":                  {"Religion"},
	}
	req := httptest.NewRequest(http.MethodGet, "/search?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestRootHandler(t *testing.T) {
	rig := newTestRig(t)
	handler := rig.server.Handler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("root request status = %d, want 200", rec.Code)
	}
	var m manifestResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse manifest json from root: %v", err)
	}
	if m.ID != "mochy-navidrome" {
		t.Fatalf("unexpected manifest ID: %s", m.ID)
	}
}

func TestSearch(t *testing.T) {
	rig := newTestRig(t)
	handler := rig.server.Handler()

	req := httptest.NewRequest(http.MethodGet, "/search?q=test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}

	q := rig.credQuery()
	q.Set("q", "Religion")
	req = httptest.NewRequest(http.MethodGet, "/search?"+q.Encode(), nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp searchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(resp.Tracks) != 1 || resp.Tracks[0].Title != "New Religion" {
		t.Fatalf("unexpected search tracks: %+v", resp.Tracks)
	}
	track := resp.Tracks[0]
	if !strings.Contains(track.CoverURL, "/art/al-1") ||
		track.CoverURL != track.ArtworkURL ||
		track.CoverURL != track.Artwork ||
		track.CoverURL != track.AlbumCover ||
		track.CoverURL != track.Thumbnail {
		t.Fatalf("unexpected search artwork fields: %+v", track)
	}
}

func TestStream(t *testing.T) {
	rig := newTestRig(t)
	handler := rig.server.Handler()

	q := rig.credQuery()
	q.Set("quality", "lossless")
	req := httptest.NewRequest(http.MethodGet, "/stream/s-1?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp streamResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if resp.Format != "flac" || resp.Codec != "flac" || resp.Quality != "Hi-Res Lossless" {
		t.Fatalf("unexpected stream specs: %+v", resp)
	}
	if !strings.Contains(resp.URL, "/audio/s-1") {
		t.Fatalf("stream URL does not point to audio proxy: %s", resp.URL)
	}

	qLow := rig.credQuery()
	qLow.Set("quality", "low")
	reqLow := httptest.NewRequest(http.MethodGet, "/stream/s-1?"+qLow.Encode(), nil)
	recLow := httptest.NewRecorder()
	handler.ServeHTTP(recLow, reqLow)
	var respLow streamResponse
	_ = json.Unmarshal(recLow.Body.Bytes(), &respLow)
	if respLow.Quality != "Low" || respLow.Codec != "mp3" {
		t.Fatalf("expected LOW mp3, got %+v", respLow)
	}
}

func TestAudioProxy(t *testing.T) {
	rig := newTestRig(t)
	handler := rig.server.Handler()
	q := rig.credQuery()

	req := httptest.NewRequest(http.MethodGet, "/audio/s-1?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Header().Get("Accept-Ranges") != "bytes" {
		t.Fatalf("Accept-Ranges = %q, want bytes", rec.Header().Get("Accept-Ranges"))
	}
	if rec.Body.Len() == 0 {
		t.Fatal("empty audio body")
	}

	req = httptest.NewRequest(http.MethodGet, "/audio/s-1?"+q.Encode(), nil)
	req.Header.Set("Range", "bytes=0-9")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusPartialContent {
		t.Fatalf("range status = %d, want 206", rec.Code)
	}
	if rec.Header().Get("Content-Range") == "" {
		t.Fatal("missing Content-Range header")
	}
	if rec.Body.Len() != 10 {
		t.Fatalf("range body len = %d, want 10", rec.Body.Len())
	}

	req = httptest.NewRequest(http.MethodHead, "/audio/s-1?"+q.Encode(), nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("HEAD status = %d, want 200", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD response must be empty, got %d bytes", rec.Body.Len())
	}
}

func TestArtProxyAndCaching(t *testing.T) {
	rig := newTestRig(t)
	handler := rig.server.Handler()
	q := rig.credQuery()

	req := httptest.NewRequest(http.MethodGet, "/art/al-1?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("missing ETag")
	}

	req = httptest.NewRequest(http.MethodGet, "/art/al-1?"+q.Encode(), nil)
	req.Header.Set("If-None-Match", etag)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want 304 Not Modified", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("304 must have empty body, got %d bytes", rec.Body.Len())
	}
}

func TestLyrics(t *testing.T) {
	rig := newTestRig(t)
	handler := rig.server.Handler()
	q := rig.credQuery()

	q.Set("id", "s-1")
	req := httptest.NewRequest(http.MethodGet, "/lyrics?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp lyricsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !strings.Contains(resp.SyncedLyrics, "[00:12.34]") {
		t.Fatalf("unexpected synced lyrics: %s", resp.SyncedLyrics)
	}

	q2 := rig.credQuery()
	q2.Set("artist", "Limp Bizkit")
	q2.Set("title", "Endless Slaughter")
	req = httptest.NewRequest(http.MethodGet, "/lyrics?"+q2.Encode(), nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp2 lyricsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !strings.Contains(resp2.PlainLyrics, "Line one of slaughter") {
		t.Fatalf("unexpected plain lyrics: %s", resp2.PlainLyrics)
	}
}

func TestMetadata(t *testing.T) {
	rig := newTestRig(t)
	handler := rig.server.Handler()
	q := rig.credQuery()
	q.Set("id", "s-1")

	req := httptest.NewRequest(http.MethodGet, "/metadata?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp metadataResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Album != "Tell Me I'm Alive" || resp.ReleaseDate != "2023" {
		t.Fatalf("unexpected metadata: %+v", resp)
	}
	if len(resp.Genres) != 1 || resp.Genres[0] != "Alternative Rock" {
		t.Fatalf("unexpected genres: %+v", resp.Genres)
	}
	if !strings.Contains(resp.CoverURL, "/art/al-1") {
		t.Fatalf("unexpected coverUrl: %s", resp.CoverURL)
	}
}

func TestHealth(t *testing.T) {
	rig := newTestRig(t)
	handler := rig.server.Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp healthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Status != "ok" || resp.Version != "1.0.0" {
		t.Fatalf("unexpected health response: %+v", resp)
	}

	q := rig.credQuery()
	req = httptest.NewRequest(http.MethodGet, "/health?"+q.Encode(), nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("credential health status = %d, want 200", rec.Code)
	}

	badQ := url.Values{
		"navidrome_url":      {rig.fake.URL},
		"navidrome_user":     {"wrong"},
		"navidrome_password": {"wrong"},
	}
	req = httptest.NewRequest(http.MethodGet, "/health?"+badQ.Encode(), nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong credential health status = %d, want 401", rec.Code)
	}
}

func TestCORS(t *testing.T) {
	rig := newTestRig(t)
	handler := rig.server.Handler()

	req := httptest.NewRequest(http.MethodOptions, "/search", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("CORS allow origin = %q, want *", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestSecurityHeaders(t *testing.T) {
	rig := newTestRig(t)
	handler := rig.server.Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", rec.Header().Get("X-Content-Type-Options"))
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("X-Frame-Options = %q, want DENY", rec.Header().Get("X-Frame-Options"))
	}
	if rec.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Errorf("Referrer-Policy = %q, want no-referrer", rec.Header().Get("Referrer-Policy"))
	}
}

func TestInvalidNavidromeURLRejected(t *testing.T) {
	rig := newTestRig(t)
	handler := rig.server.Handler()

	invalidURLs := []string{
		"ftp://malicious.com",
		"file:///etc/passwd",
		"javascript:alert(1)",
		"not-a-url",
	}

	for _, badURL := range invalidURLs {
		q := url.Values{
			"navidrome_url":      {badURL},
			"navidrome_user":     {"user"},
			"navidrome_password": {"pass"},
			"q":                  {"test"},
		}
		req := httptest.NewRequest(http.MethodGet, "/search?"+q.Encode(), nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad URL %q, got %d", badURL, rec.Code)
		}
	}
}

func TestIcon(t *testing.T) {
	rig := newTestRig(t)
	handler := rig.server.Handler()

	for _, endpoint := range []string{"/icon.png", "/logo.png"} {
		req := httptest.NewRequest(http.MethodGet, endpoint, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", endpoint, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
			t.Fatalf("%s content-type = %s, want image/png", endpoint, ct)
		}
		if rec.Body.Len() == 0 {
			t.Fatalf("%s empty icon body", endpoint)
		}
	}
}

func TestManifestCustomModuleIDAndName(t *testing.T) {
	srv := New(Options{
		ModuleID:   "navidrome-family",
		ModuleName: "Navidrome (Family)",
		Version:    "1.5.0",
	})
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/manifest.json", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var m manifestResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse manifest json: %v", err)
	}
	if m.ID != "navidrome-family" {
		t.Errorf("ID = %q, want navidrome-family", m.ID)
	}
	if m.Name != "Navidrome (Family)" {
		t.Errorf("Name = %q, want 'Navidrome (Family)'", m.Name)
	}
	if !strings.Contains(m.Description, "Navidrome (Family)") {
		t.Errorf("Description = %q, want it to contain 'Navidrome (Family)'", m.Description)
	}
}

func TestMultiServerHandler(t *testing.T) {
	fakeMusic := fakes.NewNavidrome(t)
	fakeDolby := fakes.NewNavidrome(t)

	srv := New(Options{
		PublicURL: "https://mochy.subs-server.site",
		Version:   "1.5.0",
		Instances: []InstanceConfig{
			{
				Slug:                "music",
				ModuleID:            "mochy-navidrome-music",
				ModuleName:          "Navidrome (Music)",
				DefaultNavidromeURL: fakeMusic.URL,
			},
			{
				Slug:                "dolby",
				ModuleID:            "mochy-navidrome-dolby",
				ModuleName:          "Navidrome (Dolby)",
				DefaultNavidromeURL: fakeDolby.URL,
			},
		},
	})
	handler := srv.Handler()

	// 1. Healthcheck
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("/health status = %d, want 200", rec.Code)
	}
	var health map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &health)
	if health["mode"] != "multi-instance" {
		t.Fatalf("expected mode=multi-instance, got: %+v", health)
	}

	// 2. Root Index
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("/ status = %d, want 200", rec.Code)
	}
	var rootResp multiRootResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &rootResp); err != nil {
		t.Fatalf("failed to parse root response: %v", err)
	}
	if len(rootResp.Instances) != 2 {
		t.Fatalf("root instances count = %d, want 2", len(rootResp.Instances))
	}
	if rootResp.Instances[0].ManifestURL != "https://mochy.subs-server.site/music/manifest.json" {
		t.Errorf("unexpected music manifest URL: %s", rootResp.Instances[0].ManifestURL)
	}
	if rootResp.Instances[1].ManifestURL != "https://mochy.subs-server.site/dolby/manifest.json" {
		t.Errorf("unexpected dolby manifest URL: %s", rootResp.Instances[1].ManifestURL)
	}

	// 3. Root manifest redirect to first instance
	req = httptest.NewRequest(http.MethodGet, "/manifest.json", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("/manifest.json redirect code = %d, want 307", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/music/manifest.json" {
		t.Fatalf("redirect location = %q, want /music/manifest.json", loc)
	}

	// 4. Music Instance Manifest
	req = httptest.NewRequest(http.MethodGet, "/music/manifest.json", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("/music/manifest.json status = %d, want 200", rec.Code)
	}
	var mMusic manifestResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &mMusic)
	if mMusic.ID != "mochy-navidrome-music" || mMusic.Name != "Navidrome (Music)" {
		t.Fatalf("unexpected music manifest: %+v", mMusic)
	}
	if mMusic.Icon != "https://mochy.subs-server.site/music/icon.png" {
		t.Fatalf("unexpected music icon: %s", mMusic.Icon)
	}

	// 5. Dolby Instance Manifest
	req = httptest.NewRequest(http.MethodGet, "/dolby/manifest.json", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("/dolby/manifest.json status = %d, want 200", rec.Code)
	}
	var mDolby manifestResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &mDolby)
	if mDolby.ID != "mochy-navidrome-dolby" || mDolby.Name != "Navidrome (Dolby)" {
		t.Fatalf("unexpected dolby manifest: %+v", mDolby)
	}
	if mDolby.Icon != "https://mochy.subs-server.site/dolby/icon.png" {
		t.Fatalf("unexpected dolby icon: %s", mDolby.Icon)
	}

	// 6. Music Search
	q := url.Values{
		"navidrome_user":     {fakes.NavidromeUser},
		"navidrome_password": {fakes.NavidromePassword},
		"q":                  {"Religion"},
	}
	req = httptest.NewRequest(http.MethodGet, "/music/search?"+q.Encode(), nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("/music/search status = %d, want 200", rec.Code)
	}
	var searchResp searchResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &searchResp)
	if len(searchResp.Tracks) != 1 || searchResp.Tracks[0].Title != "New Religion" {
		t.Fatalf("unexpected search response: %+v", searchResp)
	}
	if !strings.HasPrefix(searchResp.Tracks[0].CoverURL, "https://mochy.subs-server.site/music/art/") {
		t.Fatalf("unexpected cover URL path prefix: %s", searchResp.Tracks[0].CoverURL)
	}

	// 7. Instance Icon
	req = httptest.NewRequest(http.MethodGet, "/music/icon.png", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("/music/icon.png failed: status = %d", rec.Code)
	}
}
