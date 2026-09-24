package main

import (
	"log"
	"net/http"

	"github.com/mateusprt/cdn-from-scratch/cmd/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func handler(w http.ResponseWriter, r *http.Request) {
	log.Println("Request received", r.URL.Path)
	metrics.OriginRequests.Inc()
}

func main() {
	http.HandleFunc("/", handler)
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Server running...")
	log.Fatal(http.ListenAndServe(":4000", nil))
}
