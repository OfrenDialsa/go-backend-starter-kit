package metrics

import "github.com/prometheus/client_golang/prometheus"

func Init() {
	prometheus.MustRegister(
		HTTPRequests,
		HTTPDuration,
		AuthRequests,
		AuthDuration,
		EmailJobRequests,
		EmailJobDuration,
	)
}
