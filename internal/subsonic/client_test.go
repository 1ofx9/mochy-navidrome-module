package subsonic

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/1ofx9/mochy-navidrome-module/internal/fakes"
)

func newTestClient(fake *fakes.Navidrome) *Client {
	return New(fake.URL, fakes.NavidromeUser, fakes.NavidromePassword, nil)
}

func TestPing(t *testing.T) {
	fake := fakes.NewNavidrome(t)
	version, err := newTestClient(fake).Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if version != "1.16.1" {
		t.Fatalf("version = %q, want 1.16.1", version)
	}
}

func TestPingWrongCredentials(t *testing.T) {
	fake := fakes.NewNavidrome(t)
	client := New(fake.URL, fakes.NavidromeUser, "wrong-pass", nil)
	_, err := client.Ping(context.Background())
	if err == nil {
		t.Fatal("expected error with wrong credentials, got nil")
	}
}

func TestSearch(t *testing.T) {
	fake := fakes.NewNavidrome(t)
	client := newTestClient(fake)

	songs, err := client.Search(context.Background(), "Religion", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(songs) != 1 {
		t.Fatalf("got %d songs, want 1", len(songs))
	}
	if songs[0].Title != "New Religion" {
		t.Fatalf("got title %q, want 'New Religion'", songs[0].Title)
	}
	if songs[0].Codec() != "flac" {
		t.Fatalf("got codec %q, want flac", songs[0].Codec())
	}
	if songs[0].QualityLabel() != "HI_RES_LOSSLESS" {
		t.Fatalf("got quality %q, want HI_RES_LOSSLESS", songs[0].QualityLabel())
	}
}

func TestGetSong(t *testing.T) {
	fake := fakes.NewNavidrome(t)
	client := newTestClient(fake)

	song, err := client.GetSong(context.Background(), "s-1")
	if err != nil {
		t.Fatalf("GetSong failed: %v", err)
	}
	if song.ID != "s-1" || song.Title != "New Religion" {
		t.Fatalf("unexpected song: %+v", song)
	}

	_, err = client.GetSong(context.Background(), "s-999")
	if err == nil {
		t.Fatal("expected error for non-existent song, got nil")
	}
}

func TestGetLyricsSynced(t *testing.T) {
	fake := fakes.NewNavidrome(t)
	client := newTestClient(fake)

	synced, plain, err := client.GetLyrics(context.Background(), "s-1", "All Time Low", "New Religion")
	if err != nil {
		t.Fatalf("GetLyrics failed: %v", err)
	}
	if plain != "" {
		t.Fatalf("expected empty plain lyrics, got %q", plain)
	}
	if !strings.Contains(synced, "[00:12.34] I've been walking down this road") {
		t.Fatalf("unexpected synced lyrics format: %q", synced)
	}
}

func TestGetLyricsPlainFallback(t *testing.T) {
	fake := fakes.NewNavidrome(t)
	client := newTestClient(fake)

	synced, plain, err := client.GetLyrics(context.Background(), "s-2", "Limp Bizkit", "Endless Slaughter")
	if err != nil {
		t.Fatalf("GetLyrics failed: %v", err)
	}
	if synced != "" {
		t.Fatalf("expected empty synced lyrics, got %q", synced)
	}
	if !strings.Contains(plain, "Line one of slaughter") {
		t.Fatalf("unexpected plain lyrics: %q", plain)
	}
}

func TestOpenStreamRange(t *testing.T) {
	fake := fakes.NewNavidrome(t)
	client := newTestClient(fake)

	header := http.Header{}
	header.Set("Range", "bytes=0-9")
	res, err := client.OpenStream(context.Background(), "s-1", 0, header)
	if err != nil {
		t.Fatalf("OpenStream failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusPartialContent {
		t.Fatalf("status = %d, want 206", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if string(body) != "FAKEAUDIOB" {
		t.Fatalf("got %q, want 'FAKEAUDIOB'", string(body))
	}
}

func TestFetchArt(t *testing.T) {
	fake := fakes.NewNavidrome(t)
	client := newTestClient(fake)

	body, ct, etag, _, err := client.FetchArt(context.Background(), "al-1", 600)
	if err != nil {
		t.Fatalf("FetchArt failed: %v", err)
	}
	if ct != "image/jpeg" {
		t.Fatalf("contentType = %q, want image/jpeg", ct)
	}
	if etag == "" {
		t.Fatal("expected non-empty ETag")
	}
	if string(body) != "FAKEIMAGEJPEGDATA" {
		t.Fatalf("unexpected body: %q", string(body))
	}
}
