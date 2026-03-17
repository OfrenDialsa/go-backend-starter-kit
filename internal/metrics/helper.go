package metrics

import "time"

func TrackAuth(operation string, status string, duration time.Duration) {
	AuthRequests.WithLabelValues(operation, status).Inc()
	AuthDuration.WithLabelValues(operation).Observe(duration.Seconds())
}
