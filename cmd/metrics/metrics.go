package metrics

import "github.com/prometheus/client_golang/prometheus"

var CacheRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "edge_cache_requests_total",
		Help: "Total de requisições atendidas pelo edge.",
	},
	[]string{"status"},
)

var OriginRequests = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "origin_requests_total",
		Help: "Total de requisições que de fato chegaram ao origin.",
	},
)

func init() {
	prometheus.MustRegister(CacheRequests)
	prometheus.MustRegister(OriginRequests)
}
