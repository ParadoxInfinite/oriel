package server

import (
	"net"
	"net/http"

	"github.com/ParadoxInfinite/oriel/internal/docker"
	"github.com/ParadoxInfinite/oriel/internal/secrets"
)

// handleContainers lists all containers (running and stopped).
func (s *Server) handleContainers(w http.ResponseWriter, r *http.Request) {
	list, err := s.docker.ListContainers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// inspectResponse is the inspect payload plus the masking state the UI needs to
// render the env section and decide whether to offer "Reveal values".
type inspectResponse struct {
	docker.ContainerDetail
	EnvMasked bool `json:"envMasked"` // env values are masked in this payload
	CanReveal bool `json:"canReveal"` // this viewer is allowed to request raw values
}

// handleContainerInspect returns the curated inspect payload for a detail panel.
// Env values are masked server-side by default so secrets don't leak from
// screenshots (or, later, to a model over MCP). Raw values require `?reveal=1`
// AND a viewer allowed by the envReveal policy.
func (s *Server) handleContainerInspect(w http.ResponseWriter, r *http.Request) {
	d, err := s.docker.InspectContainer(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorBody(err))
		return
	}
	canReveal := s.canRevealEnv(r)
	mode := secrets.ParseMode(loadSettings().MaskEnv)
	reveal := r.URL.Query().Has("reveal") && canReveal
	masked := !reveal && mode != secrets.MaskOff
	if masked {
		d.Env = secrets.MaskEnv(d.Env, mode)
		d.Command = secrets.MaskCommand(d.Command, mode)
		d.Labels = secrets.MaskLabels(d.Labels, mode)
	}
	writeJSON(w, http.StatusOK, inspectResponse{ContainerDetail: d, EnvMasked: masked, CanReveal: canReveal})
}

// normReveal normalizes the envReveal setting, defaulting to "local".
func normReveal(s string) string {
	switch s {
	case "off", "remote":
		return s
	default:
		return "local"
	}
}

// canRevealEnv decides whether this request may unmask env values, per the
// envReveal setting: "off" never; "local" (default) only from a loopback Host;
// "remote" from any host that already passed the /api host guard.
// TODO(auth): once the optional-auth tier lands, "remote" should additionally
// require an authenticated session.
func (s *Server) canRevealEnv(r *http.Request) bool {
	switch normReveal(loadSettings().EnvReveal) {
	case "off":
		return false
	case "remote":
		return true
	default: // "local"
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		return isLoopbackHost(host)
	}
}
