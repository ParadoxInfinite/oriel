package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ParadoxInfinite/oriel/internal/provider"
)

// handleProviderStatus reports whether NL mode is available and the current URL.
func (s *Server) handleProviderStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled": s.provider.Enabled(),
		"url":     s.provider.URL(),
	})
}

// handleSetProvider swaps the resolver endpoint at runtime and persists it. An
// empty url returns the seam to dormant.
func (s *Server) handleSetProvider(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	s.provider.SetURL(body.URL)
	url := s.provider.URL()
	if err := updateSettings(func(st *settings) { st.ProviderURL = url }); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled": s.provider.Enabled(),
		"url":     s.provider.URL(),
	})
}

// handleResolve maps free text to a tool call via the configured provider and
// executes it through the same validated path as every other action. The
// provider only proposes; the registry validates and runs.
func (s *Server) handleResolve(w http.ResponseWriter, r *http.Request) {
	if !s.provider.Enabled() {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no provider configured"})
		return
	}
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Text == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing text"})
		return
	}

	call, err := s.provider.Resolve(r.Context(), provider.Request{
		Text:     body.Text,
		Tools:    s.tools.List(),
		Entities: s.gatherEntities(r.Context()),
	})
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorBody(err))
		return
	}

	result, err := s.tools.Execute(r.Context(), call.Tool, call.Args)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"call": call, "result": result})
}

// gatherEntities collects live entity names to give the provider grounding
// context. Best-effort: a failing kind is simply omitted.
func (s *Server) gatherEntities(c context.Context) map[string][]string {
	out := map[string][]string{}

	if list, err := s.docker.ListContainers(c); err == nil {
		for _, x := range list {
			out["container"] = append(out["container"], x.Name)
		}
	}
	if list, err := s.docker.ListImages(c); err == nil {
		for _, x := range list {
			out["image"] = append(out["image"], x.Tags...)
		}
	}
	if list, err := s.docker.ListVolumes(c); err == nil {
		for _, x := range list {
			out["volume"] = append(out["volume"], x.Name)
		}
	}
	if list, err := s.docker.ListNetworks(c); err == nil {
		for _, x := range list {
			out["network"] = append(out["network"], x.Name)
		}
	}
	if list, err := s.docker.ListStacks(c); err == nil {
		for _, x := range list {
			out["stack"] = append(out["stack"], x.Name)
		}
	}
	return out
}
