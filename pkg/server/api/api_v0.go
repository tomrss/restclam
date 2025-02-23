package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	clamd "github.com/tomrss/restclam/pkg/clamdv0"
)

func ClamavV0() http.Handler {
	r := chi.NewRouter()

	h := clamavV0Handler{}

	r.Get("/ping", h.handlePing)
	r.Post("/scan", h.handleScan)
	return r
}

type clamavV0Handler struct {
}

// func HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
// 	handlePing(w, r)
// }

func (h *clamavV0Handler) handlePing(w http.ResponseWriter, r *http.Request) {
	// get session from context
	s, ok := r.Context().Value("session").(*clamd.Session)
	if !ok {
		// this should never happen
		log.Error().Msg("unable to get clamd session from context")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// execute
	pong, err := s.Ping()
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

func (h *clamavV0Handler) handleScan(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("handling file scan")

	// get session from context
	s, ok := r.Context().Value("session").(*clamd.Session)
	if !ok {
		// this should never happen
		log.Error().Msg("unable to get clamd session from context")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		panic(err)
	}

	log.Debug().Str("filename", header.Filename).Msg("scanning file")

	// execute
	scan, err := s.Instream(file)
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
