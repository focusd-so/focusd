package usage

import (
	"encoding/json"
	"net/http"
	"time"

	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
	"github.com/focusd-so/focusd/internal/identity"

	"github.com/go-chi/chi/v5"
)

type PauseRequets struct {
	DurationSeconds int    `json:"duration_seconds"`
	Reason          string `json:"reason"`
}

type UnpauseRequest struct {
	Reason string `json:"reason"`
}

type WhitelistRequest struct {
	AppName         string `json:"app_name"`
	Hostname        string `json:"hostname"`
	DurationSeconds int    `json:"duration_seconds"`
}

type UnwhitelistRequest struct {
	ID int64 `json:"id"`
}

func (s *Service) RegisterHTTPHandlers(r *chi.Mux) {
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if identity.GetAccountTier() == apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE {
					http.Error(w, `{"error": "This local API feature requires a Focusd Plus or Pro plan."}`, http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
			})
		})

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

			if err := s.Whitelist(req.AppName, req.Hostname, time.Duration(req.DurationSeconds)*time.Second); err != nil {
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
