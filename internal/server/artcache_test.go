package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestArtCacheLRUEviction(t *testing.T) {
	cache := NewArtCache(2, 1000)

	cache.Put("item1", []byte("data1"), "image/jpeg", `"etag1"`, "")
	cache.Put("item2", []byte("data2"), "image/jpeg", `"etag2"`, "")

	if items, bytes := cache.Stats(); items != 2 || bytes != 10 {
		t.Fatalf("Stats = %d items, %d bytes; want 2 items, 10 bytes", items, bytes)
	}

	if _, ok := cache.Get("item1"); !ok {
		t.Fatal("item1 should be in cache")
	}

	cache.Put("item3", []byte("data3"), "image/jpeg", `"etag3"`, "")

	if _, ok := cache.Get("item2"); ok {
		t.Fatal("item2 should have been evicted")
	}
	if _, ok := cache.Get("item1"); !ok {
		t.Fatal("item1 should still be present")
	}
	if _, ok := cache.Get("item3"); !ok {
		t.Fatal("item3 should be present")
	}

	cache.Clear()
	if items, bytes := cache.Stats(); items != 0 || bytes != 0 {
		t.Fatalf("Stats after Clear = %d items, %d bytes; want 0, 0", items, bytes)
	}
}

func TestArtCacheByteLimitEviction(t *testing.T) {
	cache := NewArtCache(10, 20)

	cache.Put("a", []byte("12345678"), "image/jpeg", "", "")
	cache.Put("b", []byte("12345678"), "image/jpeg", "", "")
	cache.Put("c", []byte("12345678"), "image/jpeg", "", "")

	if _, ok := cache.Get("a"); ok {
		t.Fatal("'a' should have been evicted due to byte limit")
	}
	if _, ok := cache.Get("b"); !ok {
		t.Fatal("'b' should still be present")
	}
	if _, ok := cache.Get("c"); !ok {
		t.Fatal("'c' should be present")
	}
}

func TestArtCacheLargeItemNotCached(t *testing.T) {
	cache := NewArtCache(10, 100)
	large := make([]byte, 150)
	item := cache.Put("large", large, "image/jpeg", "", "")
	if item != nil {
		t.Fatal("large item should return nil")
	}
	if _, ok := cache.Get("large"); ok {
		t.Fatal("large item should not be in cache")
	}
}

func TestArtCacheConcurrentAccess(t *testing.T) {
	cache := NewArtCache(50, 10000)
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", n%10)
			cache.Put(key, []byte("data"), "image/jpeg", "", "")
			cache.Get(key)
			cache.Stats()
		}(i)
	}
	wg.Wait()
}

func TestServeCachedArtETagRevalidation(t *testing.T) {
	item := &artItem{
		key:          "test-art",
		body:         []byte("ARTDATA"),
		contentType:  "image/jpeg",
		etag:         `"art-123"`,
		lastModified: "Thu, 20 Nov 2026 12:00:00 GMT",
		size:         7,
	}

	req1 := httptest.NewRequest(http.MethodGet, "/art/test", nil)
	rec1 := httptest.NewRecorder()
	serveCachedArt(rec1, req1, item)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec1.Code)
	}
	if rec1.Header().Get("ETag") != item.etag {
		t.Fatalf("expected ETag %s, got %s", item.etag, rec1.Header().Get("ETag"))
	}
	if rec1.Body.String() != "ARTDATA" {
		t.Fatalf("unexpected body: %s", rec1.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/art/test", nil)
	req2.Header.Set("If-None-Match", item.etag)
	rec2 := httptest.NewRecorder()
	serveCachedArt(rec2, req2, item)

	if rec2.Code != http.StatusNotModified {
		t.Fatalf("expected 304 Not Modified, got %d", rec2.Code)
	}
	if rec2.Body.Len() != 0 {
		t.Fatalf("expected empty body for 304, got %d bytes", rec2.Body.Len())
	}
}
