package grant

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWindowLifecycle(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	s := at(filepath.Join(t.TempDir(), "grant.json"), clock)

	// Closed by default (no file).
	if s.Active() {
		t.Fatal("window should be closed before any Open")
	}

	if _, err := s.Open(6 * time.Hour); err != nil {
		t.Fatalf("Open: %v", err)
	}
	active, exp := s.Status()
	if !active {
		t.Fatal("window should be active right after Open")
	}
	if !exp.Equal(now.Add(6 * time.Hour)) {
		t.Errorf("expiry = %v, want %v", exp, now.Add(6*time.Hour))
	}

	// Just before expiry: still open. At/after expiry: closed.
	now = now.Add(6*time.Hour - time.Second)
	if !s.Active() {
		t.Error("window should still be open one second before expiry")
	}
	now = now.Add(2 * time.Second)
	if s.Active() {
		t.Error("window should auto-relock after expiry")
	}
}

func TestLockClosesEarly(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	s := at(filepath.Join(t.TempDir(), "grant.json"), func() time.Time { return now })

	s.Open(6 * time.Hour)
	if !s.Active() {
		t.Fatal("precondition: window open")
	}
	if err := s.Lock(); err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if s.Active() {
		t.Error("window should be closed after Lock")
	}
}

func TestCorruptFileIsClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "grant.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := at(path, time.Now)
	if s.Active() {
		t.Error("a corrupt grant file must read as closed, not panic or open")
	}
}
