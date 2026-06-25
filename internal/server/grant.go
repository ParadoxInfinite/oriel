package server

import (
	"encoding/json"
	"net/http"
	"time"
)

// grantStatus is the JSON shape for the destructive-grant window, shared by the
// Settings UI and the `oriel ai status` CLI.
type grantStatus struct {
	Active           bool   `json:"active"`
	ExpiresAt        string `json:"expiresAt,omitempty"` // RFC3339, empty when locked
	RemainingSeconds int    `json:"remainingSeconds"`
}

// maxGrantHours caps a window at 30 days, long enough for a trusted automation
// box, short enough that "forever" is a deliberate, repeated choice.
const maxGrantHours = 24 * 30

func (s *Server) writeGrantStatus(w http.ResponseWriter) {
	active, exp := s.grant.Status()
	out := grantStatus{Active: active}
	if active {
		out.ExpiresAt = exp.UTC().Format(time.RFC3339)
		out.RemainingSeconds = int(time.Until(exp).Seconds())
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGrantStatus(w http.ResponseWriter, _ *http.Request) {
	s.writeGrantStatus(w)
}

// handleGrantOpen starts (or extends) the window. Body: {"hours": 6}.
func (s *Server) handleGrantOpen(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Hours float64 `json:"hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody(err))
		return
	}
	if body.Hours <= 0 || body.Hours > maxGrantHours {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "hours must be between 0 and 720 (30 days)"})
		return
	}
	if _, err := s.grant.Open(time.Duration(body.Hours * float64(time.Hour))); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	s.writeGrantStatus(w)
}

func (s *Server) handleGrantLock(w http.ResponseWriter, _ *http.Request) {
	if err := s.grant.Lock(); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	s.writeGrantStatus(w)
}
