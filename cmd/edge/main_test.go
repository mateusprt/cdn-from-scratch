package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mateusprt/cdn-from-scratch/cmd/cache"
	"golang.org/x/sync/singleflight"
)

// fakeOrigin sobe um servidor HTTP real (em memória, via httptest) que
// simula o backend. Conta quantas vezes foi chamado, pra provar que o
// cache/singleflight estão funcionando.
func fakeOrigin(t *testing.T, cacheControl string, delay time.Duration) (*httptest.Server, *int32) {
	t.Helper()
	var hits int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if delay > 0 {
			time.Sleep(delay)
		}
		if cacheControl != "" {
			w.Header().Set("Cache-Control", cacheControl)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"path":"` + r.URL.Path + `"}`))
	}))

	return srv, &hits
}

func newTestHandler(t *testing.T, originURL string) http.Handler {
	t.Helper()
	target, err := url.Parse(originURL)
	if err != nil {
		t.Fatalf("invalid origin URL: %v", err)
	}
	return newHandler(target, cache.New(), &singleflight.Group{})
}

func TestCacheMissThenHit(t *testing.T) {
	origin, hits := fakeOrigin(t, "public, max-age=60", 0)
	defer origin.Close()

	handler := newTestHandler(t, origin.URL)

	// primeira request: deve ser MISS e bater no origin
	req := httptest.NewRequest(http.MethodGet, "/foo.json", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Cache"); got != "MISS" {
		t.Errorf("esperava X-Cache=MISS, veio %q", got)
	}
	if atomic.LoadInt32(hits) != 1 {
		t.Errorf("esperava 1 hit no origin, veio %d", *hits)
	}

	// segunda request: deve ser HIT e NÃO bater no origin de novo
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)

	if got := rec2.Header().Get("X-Cache"); got != "HIT" {
		t.Errorf("esperava X-Cache=HIT, veio %q", got)
	}
	if atomic.LoadInt32(hits) != 1 {
		t.Errorf("esperava continuar em 1 hit no origin, veio %d", *hits)
	}
	if rec2.Body.String() != rec.Body.String() {
		t.Errorf("corpo da resposta em cache deveria ser igual ao original")
	}
}

func TestCacheRespectsNoStore(t *testing.T) {
	origin, hits := fakeOrigin(t, "no-store", 0)
	defer origin.Close()

	handler := newTestHandler(t, origin.URL)
	req := httptest.NewRequest(http.MethodGet, "/foo.json", nil)

	handler.ServeHTTP(httptest.NewRecorder(), req)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if atomic.LoadInt32(hits) != 2 {
		t.Errorf("com no-store, esperava 2 hits no origin, veio %d", *hits)
	}
}

func TestCacheExpiresAfterTTL(t *testing.T) {
	origin, hits := fakeOrigin(t, "public, max-age=1", 0)
	defer origin.Close()

	handler := newTestHandler(t, origin.URL)
	req := httptest.NewRequest(http.MethodGet, "/foo.json", nil)

	handler.ServeHTTP(httptest.NewRecorder(), req) // MISS
	time.Sleep(1200 * time.Millisecond)            // espera expirar
	handler.ServeHTTP(httptest.NewRecorder(), req) // deveria ser MISS de novo

	if atomic.LoadInt32(hits) != 2 {
		t.Errorf("após expirar TTL, esperava 2 hits no origin, veio %d", *hits)
	}
}

func TestSingleflightCoalescesConcurrentMisses(t *testing.T) {
	origin, hits := fakeOrigin(t, "public, max-age=60", 100*time.Millisecond)
	defer origin.Close()

	handler := newTestHandler(t, origin.URL)
	req := httptest.NewRequest(http.MethodGet, "/foo.json", nil)

	const concurrency = 20
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			handler.ServeHTTP(httptest.NewRecorder(), req)
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(hits); got != 1 {
		t.Errorf("singleflight deveria coalescer em 1 request ao origin, mas houve %d", got)
	}
}
