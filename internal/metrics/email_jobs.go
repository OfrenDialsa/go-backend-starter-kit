package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var EmailJobRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "email_job_total",
		Help: "Total email jobs processed",
	},
	[]string{"type", "status"}, // type: verify_email, forgot_password, etc. status: success, failed, max_retry
)

var EmailJobDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "email_job_duration_seconds",
		Help:    "Duration of email job processing",
		Buckets: []float64{0.5, 1, 2, 5, 10},
	},
	[]string{"type"},
)

func TrackEmailJob(emailType string, status string, duration time.Duration) {
	EmailJobRequests.WithLabelValues(emailType, status).Inc()
	EmailJobDuration.WithLabelValues(emailType).Observe(duration.Seconds())
}
