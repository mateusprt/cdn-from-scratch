// cmd/scrubber/main.go
package main

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/mateusprt/cdn-from-scratch/cmd/ratelimit"
)

const (
	RequestsPerSecond = 10
	Burst             = 20
)

func main() {
	edgeURL := os.Getenv("EDGE_URL")
	if edgeURL == "" {
		log.Fatal("Edge URL not found")
	}

	target, err := url.Parse(edgeURL)
	if err != nil {
		log.Fatalf("invalid EDGE_URL: %v", err)
	}

	limiter := ratelimit.New(RequestsPerSecond, Burst)
	handler := newHandler(target, limiter)

	log.Println("scrubber listening on :8000, forwarding to", edgeURL)
	log.Fatal(http.ListenAndServe(":8000", handler))
}

func newHandler(target *url.URL, limiter *ratelimit.Limiter) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(target)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := extractIPFromRequest(r)

		if !limiter.Allow(clientIP) {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}

		proxy.ServeHTTP(w, r)
	})
}

func extractIPFromRequest(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
