package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP requests",
	}, []string{"service", "method", "code"})
	duration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"service", "method"})
)

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func Middleware(service string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, code: 200}
			next.ServeHTTP(sw, r)
			requests.WithLabelValues(service, r.Method, strconv.Itoa(sw.code)).Inc()
			duration.WithLabelValues(service, r.Method).Observe(time.Since(start).Seconds())
		})
	}
}
