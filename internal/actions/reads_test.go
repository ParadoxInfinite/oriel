package actions

import (
	"testing"

	"github.com/ParadoxInfinite/oriel/internal/docker"
	"github.com/ParadoxInfinite/oriel/internal/secrets"
)

// The read tools are infrastructure for the assistant/MCP surface, so guard
// their contract: they must be registered, never Destructive, and the two that
// take an id must require it. Handlers aren't executed here (that needs a live
// Docker host); we only assert the registration shape.
func TestReadToolsRegistered(t *testing.T) {
	r := New(docker.New(), func() secrets.Mode { return secrets.MaskAll }, func() secrets.Mode { return secrets.MaskSensitive })

	byName := map[string]bool{}
	dangerous := map[string]bool{}
	for _, tool := range r.List() {
		byName[tool.Name] = true
		dangerous[tool.Name] = tool.Destructive
	}

	reads := []string{
		"container.list", "container.inspect", "container.logs",
		"image.list", "volume.list", "network.list", "network.inspect",
		"stacks.list", "system.df", "colima.status",
	}
	for _, name := range reads {
		if !byName[name] {
			t.Errorf("read tool %q not registered", name)
		}
		if dangerous[name] {
			t.Errorf("read tool %q must not be Destructive", name)
		}
	}

	// Mutations from the existing registrations must still be present.
	for _, name := range []string{"container.stop", "image.remove", "volume.remove", "network.remove", "network.create", "network.connect", "network.disconnect"} {
		if !byName[name] {
			t.Errorf("mutation tool %q missing after adding reads", name)
		}
	}
}

func TestReadToolSchemas(t *testing.T) {
	r := New(docker.New(), func() secrets.Mode { return secrets.MaskAll }, func() secrets.Mode { return secrets.MaskSensitive })
	// tool name → {required param, entity kind it validates}
	want := map[string]struct{ req, kind string }{
		"container.inspect": {"id", "container"},
		"container.logs":    {"id", "container"},
		"network.inspect":   {"id", "network"},
	}
	for _, tool := range r.List() {
		w, ok := want[tool.Name]
		if !ok {
			continue
		}
		if len(tool.Schema.Required) != 1 || tool.Schema.Required[0] != w.req {
			t.Errorf("%s: want required [%s], got %v", tool.Name, w.req, tool.Schema.Required)
		}
		if tool.Entity == nil || tool.Entity.Kind != w.kind {
			t.Errorf("%s: want %s entity ref, got %+v", tool.Name, w.kind, tool.Entity)
		}
	}
}
