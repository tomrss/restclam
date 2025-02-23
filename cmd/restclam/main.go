package main

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog"
	"github.com/tomrss/restclam/pkg/clamd"
	clamdv0 "github.com/tomrss/restclam/pkg/clamdv0"
	"github.com/tomrss/restclam/pkg/server/api"
	"github.com/tomrss/restclam/pkg/server/api/middleware"
	"github.com/tomrss/restclam/pkg/server/config"
)

//nolint:funlen
func main() {
	// read configuration
	conf, err := config.LoadConfig()
	if err != nil {
		fallbackLogger := httplog.NewLogger("restclam", httplog.Options{JSON: false})
		fallbackLogger.Fatal().Err(err).Msg("unable to read configuration")
	}

	// configure logger
	logger := httplog.NewLogger("restclam", httplog.Options{
		LogLevel: conf.Log.Level,
		Concise:  conf.Log.Concise,
		JSON:     conf.Log.JSON,
		Tags:     map[string]string{"environment": conf.Environment},
	})

	// create router
	r := chi.NewRouter()
	r.Use(middleware.LogRequest(conf.Log, logger))

	// init clamd client v0 and register apiv0
	if conf.FeatureFlags.ApiV0 {
		clamdPool, err := clamdv0.InitSessionPool(clamdv0.SessionPoolOpts{
			PrewarmthSessions: conf.Clam.PrewarmthSessions,
			MaxIdleSessions:   conf.Clam.MaxIdleSessions,
			ConnectMaxRetries: conf.Clam.ConnectMaxRetries,
			NewSession: func() (*clamdv0.Session, error) {
				return clamdv0.OpenSession(clamdv0.SessionOpts{
					Opts: clamdv0.Opts{
						ConnectTimeout:  5 * time.Second,
						ReadTimeout:     120 * time.Second,
						WriteTimeout:    5 * time.Second,
						StreamChunkSize: 2048,
					},
					Network:           conf.Clam.Network,
					Address:           conf.Clam.Address,
					HeartbeatInterval: conf.Clam.HeartbeatInterval,
					ConnectRetries: clamdv0.RetryOpts{
						MaxRetries: 10,
						Backoff: func(retryCount int) time.Duration {
							// linear backoff (TODO conf)
							return time.Duration(retryCount) * 2 * time.Second
						},
					},
					CommandRetries: clamdv0.RetryOpts{},
				})
			},
			Logger: newClamdLogDriver(&logger),
		})
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to setup clamd connection pool")
		}
		defer func() {
			clamdPool.Close()
		}()

		// middleware that injects clamd sessions in http request handling
		sessionMiddleware := middleware.ClamdSession(clamdPool)

		// register the v0 api
		r.With(sessionMiddleware).Mount("/api/v0/clamav", api.ClamavV0())

		logger.Info().Msg("using clamd v0 session pool at /api/v0")
	}

	// init clamd client v1 and register apiv1
	if conf.FeatureFlags.ApiV1 {
		coord := clamd.Coordinator{
			// TODO conf
			MinWorkers: 10,
			MaxWorkers: 10,
			Autoscale:  false,
		}
		if err := coord.InitCoordinator(clamd.SessionOpts{
			Opts: clamd.Opts{
				ConnectTimeout:  5 * time.Second,
				ReadTimeout:     120 * time.Second,
				WriteTimeout:    5 * time.Second,
				StreamChunkSize: 2048,
			},
			Network:           conf.Clam.Network,
			Address:           conf.Clam.Address,
			HeartbeatInterval: conf.Clam.HeartbeatInterval,
			ConnectRetries: clamd.RetryOpts{
				MaxRetries: 10,
				Backoff: func(retryCount int) time.Duration {
					// linear backoff (TODO conf)
					return time.Duration(retryCount) * 2 * time.Second
				},
			},
		}); err != nil {
			logger.Fatal().Err(err).Msg("unable to init clamd session coordinator")
		}

		// register the v1 api
		r.Mount("/api/v1/clamav", api.ClamavV1(&coord))

		logger.Info().Msg("using clamd v1 session coordinator at /api/v1")
	}

	// start server
	httpListenAndServe(context.Background(), r, conf.Server)

	logger.Info().Msg("shutdown completed")
}
