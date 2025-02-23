package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/tomrss/restclam/pkg/clamd"
)

func ClamavV1(c *clamd.Coordinator) http.Handler {
	r := chi.NewRouter()

	h := clamavV1handler{c}

	r.Get("/ping", h.handlePing)
	r.Post("/scan", h.handleScan)
	return r
}

type clamavV1handler struct {
	c *clamd.Coordinator
}

// func (h *ClamavV1Handler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
// 	handlePing(w, r)
// }

func (h *clamavV1handler) handlePing(w http.ResponseWriter, _ *http.Request) {
	// execute
	pong, err := h.c.Ping()
	if err != nil {
		// todo json response
		log.Error().Err(err).Msg("error pinging clamd")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Debug().Str("ping", pong).Msg("ping success")

	// marshal response
	w.Header().Set("Content-Type", "application/json")
	resp := struct {
		Message string `json:"message"`
	}{pong}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error().Err(err).Msg("Error marshalling response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *clamavV1handler) handleScan(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("handling file scan")

	file, header, err := r.FormFile("file")
	if err != nil {
		panic(err)
	}

	log.Debug().Str("filename", header.Filename).Msg("scanning file")

	// execute
	scan, err := h.c.Instream(file)
	if err != nil {
		// todo json response
		log.Error().
			Str("filename", header.Filename).
			Err(err).
			Msg("error scanning file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Debug().
		Str("filename", header.Filename).
		Str("virus", scan.Virus).
		Str("error", scan.Error).
		Str("status", string(scan.Status)).
		Msg("file scan complete")

	// marshal response
	w.Header().Set("Content-Type", "application/json")
	// TODO non-anonymous struct please!
	resp := struct {
		Status   string `json:"status"`
		Virus    string `json:"virus"`
		Error    string `json:"error"`
		Filename string `json:"filename"`
	}{string(scan.Status), scan.Virus, scan.Error, header.Filename}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error().Err(err).Msg("Error marshalling response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
