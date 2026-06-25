package actions

import (
	"testing"

	"github.com/ParadoxInfinite/oriel/internal/docker"
	"github.com/ParadoxInfinite/oriel/internal/secrets"
)

// TestColimaToolsRegistered locks in the VM lifecycle tools and their gating:
// stop/restart take the whole VM down, so they're destructive; start is free.
func TestColimaToolsRegistered(t *testing.T) {
	r := New(docker.New(), func() secrets.Mode { return secrets.MaskAll }, func() secrets.Mode { return secrets.MaskSensitive })

	want := map[string]bool{ // tool name → destructive
		"colima.start":   false,
		"colima.stop":    true,
		"colima.restart": true,
	}
	got := map[string]bool{}
	for _, tl := range r.List() {
		if _, ok := want[tl.Name]; ok {
			got[tl.Name] = tl.Destructive
		}
	}
	for name, destructive := range want {
		d, ok := got[name]
		if !ok {
			t.Errorf("tool %q not registered", name)
			continue
		}
		if d != destructive {
			t.Errorf("tool %q destructive = %v, want %v", name, d, destructive)
		}
	}
}
