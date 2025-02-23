package middleware

import (
	"net/http"

	"github.com/go-chi/httplog"
	"github.com/rs/zerolog"
	"github.com/tomrss/restclam/pkg/server/config"
)

// LogRequest is a middleware that logs every request if needed.
func LogRequest(conf config.LogConfig, logger zerolog.Logger) func(next http.Handler) http.Handler {
	if !conf.LogRequests {
		// no logging, just serve the next handler
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}
	}

	// use logging middleware from chi
	return httplog.RequestLogger(logger)
}
