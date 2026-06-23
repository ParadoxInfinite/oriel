package actions

import (
	"testing"

	"github.com/ParadoxInfinite/oriel/internal/docker"
	"github.com/ParadoxInfinite/oriel/internal/secrets"
)

// TestReadOnlyClassification locks in that pure reads are marked ReadOnly and
// mutations are not — so `oriel mcp --read-only` and the MCP read-only hint are
// accurate (Destructive:false alone would wrongly include start/stop).
func TestReadOnlyClassification(t *testing.T) {
	r := New(docker.New(), func() secrets.Mode { return secrets.MaskAll })

	want := map[string]bool{ // tool name → ReadOnly
		"container.list":   true,
		"container.logs":   true,
		"colima.status":    true,
		"docker.env":       true,
		"stacks.list":      true,
		"container.start":  false,
		"container.remove": false,
		"image.prune":      false,
		"stack.down":       false,
		"stack.alias":      false,
	}
	got := map[string]bool{}
	for _, tl := range r.List() {
		if _, ok := want[tl.Name]; ok {
			got[tl.Name] = tl.ReadOnly
		}
	}
	for name, ro := range want {
		g, ok := got[name]
		if !ok {
			t.Errorf("tool %q not registered", name)
			continue
		}
		if g != ro {
			t.Errorf("tool %q ReadOnly = %v, want %v", name, g, ro)
		}
	}
}
