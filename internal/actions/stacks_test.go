package actions

import (
	"testing"

	"github.com/ParadoxInfinite/oriel/internal/docker"
	"github.com/ParadoxInfinite/oriel/internal/secrets"
)

// TestStackToolsRegistered locks in that the four compose-stack lifecycle tools
// are present in the registry (so MCP exposes them) with the right destructive
// flags: only `down` tears things down.
func TestStackToolsRegistered(t *testing.T) {
	r := New(docker.New(), func() secrets.Mode { return secrets.MaskAll }, func() secrets.Mode { return secrets.MaskSensitive })

	want := map[string]bool{ // tool name → destructive
		"stack.start":   false,
		"stack.stop":    false,
		"stack.restart": false,
		"stack.down":    true,
		"stack.alias":   false,
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
