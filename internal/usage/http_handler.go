package usage

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/focusd-so/focusd/internal/settings"
)

type PauseRequets struct {
	DurationSeconds int    `json:"duration_seconds"`
	Reason          string `json:"reason"`
}

type UnpauseRequest struct {
	Reason string `json:"reason"`
}

type WhitelistRequest struct {
	ExecutablePath  string `json:"executable_path"`
	Hostname        string `json:"hostname"`
	DurationSeconds int    `json:"duration_seconds"`
}

type UnwhitelistRequest struct {
	ID int64 `json:"id"`
}

// apiKeyAuth is a middleware that validates the Authorization header against the stored API key.
func apiKeyAuth(settingsService *settings.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey, err := settingsService.GetAPIKey()
			if err != nil || apiKey == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || authHeader != "Bearer "+apiKey {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (s *Service) RegisterHTTPHandlers(r *chi.Mux, settingsService *settings.Service) {
	r.Group(func(r chi.Router) {
		r.Use(apiKeyAuth(settingsService))

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

		r.Post("/whitelist", func(w http.ResponseWriter, r *http.Request) {
			var req WhitelistRequest

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if req.DurationSeconds <= 0 {
				req.DurationSeconds = 60 * 60 // 1 hour
			}

			if err := s.Whitelist(req.ExecutablePath, req.Hostname, time.Duration(req.DurationSeconds)*time.Second); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
		})

		r.Post("/unwhitelist", func(w http.ResponseWriter, r *http.Request) {
			var req UnwhitelistRequest

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if err := s.RemoveWhitelist(req.ID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
		})
	})
}
