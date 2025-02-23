package middleware

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"
	clamd "github.com/tomrss/restclam/pkg/clamdv0"
)

func ClamdSession(p *clamd.SessionPool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// borrow session from pool
			log.Debug().
				Msg("borrowing session from pool....")
			s, err := p.Get()
			if err != nil {
				// TODO json response
				log.Error().
					Err(err).
					Msg("unable to get session from pool")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			log.Debug().
				Int("sessionId", s.ID()).
				Msg("borrowed session from pool")

			defer func() {
				// return session to pool
				p.Put(s)

				log.Debug().
					Int("sessionId", s.ID()).
					Msg("returned session to pool")
			}()

			// TODO context keys handling
			//nolint:revive,staticcheck
			ctx := context.WithValue(r.Context(), "session", s)

			// serve the next handler with context with session inside
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
