package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

func requestLogger(name string, logger *logrus.Logger) func(next http.Handler) http.Handler {
	latencyField := fmt.Sprintf("measure#%s.latency", name)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			responseWriter := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(responseWriter, r)

			latency := time.Since(start)
			fields := logrus.Fields{
				"status":     responseWriter.Status(),
				"took":       latency,
				latencyField: latency.Nanoseconds(),
				"remote":     r.RemoteAddr,
				"request":    r.RequestURI,
				"method":     r.Method,
			}
			if requestID := middleware.GetReqID(r.Context()); requestID != "" {
				fields["request-id"] = requestID
			}

			logger.WithFields(fields).Info("request completed")
		})
	}
}
