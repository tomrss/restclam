package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/tomrss/restclam/pkg/server/config"
)

// HTTPListenAndServe starts an HTTP server in a goroutine with a given router.
func HTTPListenAndServe(router *chi.Mux, cfg config.ServerConfig) {
	// setup server
	addr := net.JoinHostPort(cfg.Host, fmt.Sprint(cfg.Port))
	server := &http.Server{
		Addr:         addr,
		WriteTimeout: cfg.WriteTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		IdleTimeout:  cfg.IdleTimeout,
		Handler:      router,
	}

	// start server async
	go func() {
		if err := server.ListenAndServe(); errors.Is(err, http.ErrServerClosed) {
			log.Info().Msg("Server closed")
		} else if err != nil {
			log.Fatal().
				Err(err).
				Msg("Error starting server")
		}
	}()

	log.Info().Str("address", addr).Msg("Server started")

	// handle interruption signals
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	signal := <-done
	log.Info().Str("signal", signal.String()).Msg("Received interruption signal")

	// graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer func() {
		cancel()
	}()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().
			Err(err).
			Msg("Server shutdown failed")
	}
}
