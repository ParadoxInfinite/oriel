package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ParadoxInfinite/oriel/internal/tools"
)

// handleTools lists the registered tools so the command palette knows what
// actions exist.
func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.tools.List())
}

// invokeRequest is the body for POST /api/invoke — the single execution entry
// shared by UI buttons and the palette.
type invokeRequest struct {
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
}

func (s *Server) handleInvoke(w http.ResponseWriter, r *http.Request) {
	var req invokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody(err))
		return
	}

	// /api/invoke is the interactive surface (UI buttons + command palette, both
	// behind their own confirm dialogs), so it carries consent — destructive
	// tools run without needing a grant window. Agent paths (MCP) don't.
	result, err := s.tools.Execute(tools.WithConsent(r.Context()), req.Tool, req.Args)
	if err != nil {
		status := http.StatusUnprocessableEntity
		if errors.Is(err, tools.ErrUnknownTool) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": result})
}
