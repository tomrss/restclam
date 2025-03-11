package main

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog"
	"github.com/rs/zerolog"
	"github.com/tomrss/restclam/pkg/clamd"
	clamdv0 "github.com/tomrss/restclam/pkg/clamdv0"
	"github.com/tomrss/restclam/pkg/server"
	"github.com/tomrss/restclam/pkg/server/api"
	"github.com/tomrss/restclam/pkg/server/api/middleware"
	"github.com/tomrss/restclam/pkg/server/config"
)

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
	r.Use(middleware.Cors(conf.Cors))

	// init clamd client v0 and register apiv0
	if conf.FeatureFlags.ApiV0 {
		clamdPool, err := initSessionPool(conf.Clam, logger)
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to setup clamd connection pool")
		}
		defer clamdPool.Close()

		// middleware that injects clamd sessions in http request handling
		sessionMiddleware := middleware.ClamdSession(clamdPool)

		// register the v0 api
		r.With(sessionMiddleware).Mount("/api/v0/clamav", api.ClamavV0())

		logger.Info().Msg("using clamd v0 session pool at /api/v0")
	}

	// init clamd client v1 and register apiv1
	if conf.FeatureFlags.ApiV1 {
		coordinator, err := runCoordinator(conf.Clam, logger)
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to init clamd session coordinator")
		}
		defer coordinator.Shutdown()

		// register the v1 api
		r.Mount("/api/v1/clamav", api.ClamavV1(coordinator))

		logger.Info().Msg("using clamd v1 session coordinator at /api/v1")
	}

	// start server
	server.HTTPListenAndServe(r, conf.Server)

	logger.Info().Msg("shutdown completed")
}

func initSessionPool(c config.ClamConfig, logger zerolog.Logger) (*clamdv0.SessionPool, error) {
	clamdPool, err := clamdv0.InitSessionPool(clamdv0.SessionPoolOpts{
		PrewarmthSessions: c.MinWorkers,
		MaxIdleSessions:   c.MaxWorkers,
		ConnectMaxRetries: c.ConnectMaxRetries,
		NewSession: func() (*clamdv0.Session, error) {
			return clamdv0.OpenSession(clamdv0.SessionOpts{
				Opts: clamdv0.Opts{
					ConnectTimeout:  c.ConnectTimeout,
					ReadTimeout:     c.ReadTimeout,
					WriteTimeout:    c.WriteTimeout,
					StreamChunkSize: c.StreamChunkSize,
				},
				Network:           c.Network,
				Address:           c.Address,
				HeartbeatInterval: c.HeartbeatInterval,
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
		Logger: newClamdV0LogDriver(&logger),
	})
	return clamdPool, err
}

func runCoordinator(c config.ClamConfig, _ zerolog.Logger) (*clamd.Coordinator, error) {
	coord := clamd.Coordinator{
		MinWorkers:      c.MinWorkers,
		MaxWorkers:      c.MaxWorkers,
		Autoscale:       false,
		ShutdownTimeout: 10 * time.Second,
	}
	err := coord.InitCoordinator(
		[]clamd.Clamd{{
			Network:         c.Network,
			Address:         c.Address,
			ConnectTimeout:  c.ConnectTimeout,
			ReadTimeout:     c.ReadTimeout,
			WriteTimeout:    c.WriteTimeout,
			StreamChunkSize: c.StreamChunkSize,
		}},
		clamd.SessionOpts{
			HeartbeatInterval: c.HeartbeatInterval,
			ConnectRetries: clamd.RetryOpts{
				MaxRetries: c.ConnectMaxRetries,
				Backoff: func(retryCount int) time.Duration {
					// linear backoff (TODO make algorithm configurable)
					return time.Duration(retryCount) * c.ConnectRetryInterval
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return &coord, nil
}
