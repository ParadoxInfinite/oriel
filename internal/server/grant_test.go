package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ParadoxInfinite/oriel/internal/grant"
)

// grantServer builds a Server with only the grant store wired, backed by a temp
// file so the test never touches the user's real grant.json.
func grantServer(t *testing.T) *Server {
	t.Helper()
	return &Server{grant: grant.NewAt(filepath.Join(t.TempDir(), "grant.json"))}
}

func decodeStatus(t *testing.T, body *httptest.ResponseRecorder) grantStatus {
	t.Helper()
	var st grantStatus
	if err := json.Unmarshal(body.Body.Bytes(), &st); err != nil {
		t.Fatalf("decode: %v (%s)", err, body.Body.String())
	}
	return st
}

func TestGrantOpenStatusLock(t *testing.T) {
	s := grantServer(t)

	// Initially locked.
	rec := httptest.NewRecorder()
	s.handleGrantStatus(rec, httptest.NewRequest("GET", "/api/grant", nil))
	if decodeStatus(t, rec).Active {
		t.Fatal("expected locked initially")
	}

	// Open a 6h window.
	rec = httptest.NewRecorder()
	s.handleGrantOpen(rec, httptest.NewRequest("POST", "/api/grant", strings.NewReader(`{"hours":6}`)))
	st := decodeStatus(t, rec)
	if !st.Active || st.RemainingSeconds <= 0 || st.ExpiresAt == "" {
		t.Fatalf("expected active window, got %+v", st)
	}

	// Lock it.
	rec = httptest.NewRecorder()
	s.handleGrantLock(rec, httptest.NewRequest("DELETE", "/api/grant", nil))
	if decodeStatus(t, rec).Active {
		t.Fatal("expected locked after DELETE")
	}
}

func TestGrantRejectsBadHours(t *testing.T) {
	s := grantServer(t)
	for _, body := range []string{`{"hours":0}`, `{"hours":-3}`, `{"hours":100000}`} {
		rec := httptest.NewRecorder()
		s.handleGrantOpen(rec, httptest.NewRequest("POST", "/api/grant", strings.NewReader(body)))
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("hours %s: want 422, got %d", body, rec.Code)
		}
	}
}
