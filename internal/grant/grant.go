// Package grant implements the time-boxed "destructive actions" window that
// unlocks Destructive tools for non-interactive callers (the MCP server, a
// future in-app assistant). Interactive UI calls don't need it — they're
// already human-gated — but an agent can only remove/prune inside a window the
// user opened on purpose, and the window auto-relocks when it lapses.
//
// State is a single JSON file shared by every Oriel process (server, `oriel
// mcp`, the CLI), so opening a window from one is seen by all. Active/Status
// read the file fresh; they're only hit on destructive calls, so the cost is
// irrelevant.
package grant

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/ParadoxInfinite/oriel/internal/userdata"
)

// Store is the on-disk grant window. The zero value is unusable; call New.
type Store struct {
	path string
	now  func() time.Time // injectable for tests
}

type state struct {
	ExpiresAt time.Time `json:"expiresAt"`
}

// New returns the default store backed by <userdata>/grant.json.
func New() *Store { return NewAt(userdata.Path("grant.json")) }

// NewAt returns a store backed by an explicit file. Used by tests and any
// caller that needs to target a non-default location.
func NewAt(path string) *Store { return &Store{path: path, now: time.Now} }

// at builds a store over an explicit path with a fixed clock, for tests.
func at(path string, now func() time.Time) *Store { return &Store{path: path, now: now} }

func (s *Store) read() state {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return state{}
	}
	var st state
	if json.Unmarshal(b, &st) != nil {
		return state{}
	}
	return st
}

// Active reports whether a destructive window is currently open.
func (s *Store) Active() bool {
	active, _ := s.Status()
	return active
}

// Status returns whether a window is open and, if so, when it expires.
func (s *Store) Status() (bool, time.Time) {
	exp := s.read().ExpiresAt
	if exp.IsZero() || !s.now().Before(exp) {
		return false, time.Time{}
	}
	return true, exp
}

// Open starts (or extends) the window to now+d and returns the new expiry.
func (s *Store) Open(d time.Duration) (time.Time, error) {
	exp := s.now().Add(d)
	if err := s.write(state{ExpiresAt: exp}); err != nil {
		return time.Time{}, err
	}
	return exp, nil
}

// Lock closes the window immediately.
func (s *Store) Lock() error { return s.write(state{}) }

func (s *Store) write(st state) error {
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
