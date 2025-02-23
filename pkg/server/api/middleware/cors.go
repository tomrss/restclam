package middleware

import (
	"net/http"

	"github.com/go-chi/cors"
	"github.com/tomrss/restclam/pkg/server/config"
)

// Cors is a middleware that handles cors.
func Cors(conf config.CORSConfig) func(next http.Handler) http.Handler {
	if !conf.Enabled {
		// no cors, just serve the next handler
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}
	}

	// use cors middleware from chi
	return cors.Handler(cors.Options{
		AllowedOrigins:   conf.AllowedOrigins,
		AllowedMethods:   conf.AllowedMethods,
		AllowedHeaders:   conf.AllowedHeaders,
		ExposedHeaders:   conf.ExposedHeaders,
		AllowCredentials: conf.AllowCredentials,
		MaxAge:           conf.MaxAge,
	})
}
