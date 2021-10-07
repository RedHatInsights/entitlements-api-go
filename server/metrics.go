package server

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var responseStatus = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "entitlements_api_response_status",
		Help: "Status of Entitlements api response.",
	},
	[]string{"code", "path"},
)

var requestDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "entitlements_api_duration_seconds",
		Help:    "Duration of entitlements api requests.",
		Buckets: prometheus.LinearBuckets(0.25, 0.25, 20),
	},
	[]string{"path"})

// prometheusMiddleware implements mux.MiddlewareFunc.
func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		timer := prometheus.NewTimer(requestDuration.WithLabelValues(path))
		rw := NewResponseWriter(w)
		next.ServeHTTP(rw, r)

		statusCode := strconv.Itoa(rw.statusCode)
		responseStatus.WithLabelValues(statusCode, path).Inc()
		timer.ObserveDuration()
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func init() {
	prometheus.Register(responseStatus)
}
