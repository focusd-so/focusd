package usage

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type PauseRequets struct {
	DurationSeconds int    `json:"duration_seconds"`
	Reason          string `json:"reason"`
}

type UnpauseRequest struct {
	Reason string `json:"reason"`
}

func (s *Service) RegisterHTTPHandlers(r *chi.Mux) {
	r.Post("/pause", func(w http.ResponseWriter, r *http.Request) {
		var pauseRequest PauseRequets

		if err := json.NewDecoder(r.Body).Decode(&pauseRequest); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if pauseRequest.DurationSeconds <= 0 {
			pauseRequest.DurationSeconds = 60 * 60 // 1 hour
		}

		protectionPause, err := s.PauseProtection(pauseRequest.DurationSeconds, pauseRequest.Reason)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(protectionPause)
		w.WriteHeader(http.StatusOK)
	})

	r.Post("/unpause", func(w http.ResponseWriter, r *http.Request) {
		var unpauseRequest UnpauseRequest

		if err := json.NewDecoder(r.Body).Decode(&unpauseRequest); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		protectionPause, err := s.ResumeProtection(unpauseRequest.Reason)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(protectionPause)
		w.WriteHeader(http.StatusOK)
	})
}
