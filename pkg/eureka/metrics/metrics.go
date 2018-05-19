package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	FetchProcessTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "eureka_fetch_nanoseconds",
		Help: "process time per eureka client fetch",
	})
	RequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "eureka_ge_total",
			Help: "Total of the eureka fetch from client.",
		},
		[]string{"path"},
	)
)

func init() {
	prometheus.MustRegister(RequestCount)
	prometheus.MustRegister(FetchProcessTime)
}
