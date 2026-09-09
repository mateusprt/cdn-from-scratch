// cmd/edge/main.go
package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/mateusprt/cdn-from-scratch/cmd/cache"
	"golang.org/x/sync/singleflight"
)

func newHandler(target *url.URL, c *cache.Cache, bagOfRequests *singleflight.Group) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(target)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path

		if entry, ok := c.Get(key); ok {
			writeResponse(w, entry, "HIT")
			return
		}

		result, err, _ := bagOfRequests.Do(key, func() (interface{}, error) {
			return fetchFromOrigin(proxy, r)
		})

		if err != nil {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}

		entry := result.(cache.Entry)
		c.Set(key, entry)
		writeResponse(w, entry, "MISS")
	})
}

func fetchFromOrigin(proxy *httputil.ReverseProxy, r *http.Request) (cache.Entry, error) {
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, r)

	entry := cache.Entry{
		Body:       rec.Body.Bytes(),
		Headers:    rec.Header().Clone(),
		StatusCode: rec.Code,
		ExpiresAt:  time.Now().Add(cache.TTLFromHeaders(rec.Header())),
	}
	return entry, nil
}

func writeResponse(w http.ResponseWriter, entry cache.Entry, status string) {
	for k, v := range entry.Headers {
		w.Header()[k] = v
	}
	w.Header().Set("X-Cache", status)
	w.WriteHeader(entry.StatusCode)
	io.Copy(w, bytes.NewReader(entry.Body))
}

func main() {
	originURL := os.Getenv("ORIGIN_URL")
	if originURL == "" {
		originURL = "http://origin:8001"
	}

	target, err := url.Parse(originURL)
	if err != nil {
		log.Fatalf("invalid ORIGIN_URL: %v", err)
	}

	handler := newHandler(target, cache.New(), &singleflight.Group{})

	log.Println("edge listening on :8001, forwarding to", originURL)
	log.Fatal(http.ListenAndServe(":8001", handler))
}
