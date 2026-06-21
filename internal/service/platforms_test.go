package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The unit no longer carries any config (that lives in settings.json now); this
// just guards the basic shape of the rendered systemd unit.
func TestLinuxUnitShape(t *testing.T) {
	p := filepath.Join(t.TempDir(), "oriel.service")
	if err := render(p, linuxUnit, map[string]any{"Bin": "/usr/local/bin/oriel", "Port": 4321, "System": true}); err != nil {
		t.Fatalf("render: %v", err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "ExecStart=/usr/local/bin/oriel --no-open --port 4321\n") {
		t.Errorf("unexpected ExecStart, got:\n%s", s)
	}
	if strings.Contains(s, "Environment=") {
		t.Errorf("unit should carry no Environment= lines, got:\n%s", s)
	}
	if !strings.Contains(s, "WantedBy=multi-user.target") {
		t.Errorf("expected system WantedBy, got:\n%s", s)
	}
}
