package actions

import (
	"testing"

	"github.com/ParadoxInfinite/oriel/internal/docker"
	"github.com/ParadoxInfinite/oriel/internal/secrets"
)

// reversible is the explicit allowlist of mutating-but-not-destructive tools
// that run ungated (a UI button or an MCP/agent caller can invoke them without a
// grant) because they're trivially reversible. A tool that is neither ReadOnly
// nor Destructive and NOT listed here is unclassified, and the test below fails.
//
// This is the backstop for the grant gate: the gate keys on Destructive, so a
// new destructive tool that ships without the flag would inherit the ungated
// default. Adding any new mutating tool forces a deliberate choice here (with a
// reviewer watching the diff): list it as reversible, or flag it Destructive.
// The Register() verb tripwire catches the obvious `*.remove`/`*.prune`/… names;
// this catches everything else.
var reversible = map[string]bool{
	"container.start":    true,
	"container.stop":     true,
	"container.restart":  true,
	"stack.start":        true,
	"stack.stop":         true,
	"stack.restart":      true,
	"stack.alias":        true, // display-only label write, not a Docker action
	"image.tag":          true,
	"network.create":     true,
	"network.connect":    true,
	"network.disconnect": true,
	"colima.start":       true,
}

// TestNoUnclassifiedTools fails if any registered tool is neither a pure read
// (ReadOnly) nor Destructive nor an explicitly-allowlisted reversible mutation.
// It exists so a future destructive tool can't silently inherit the ungated
// default and run for an agent with no grant window open.
func TestNoUnclassifiedTools(t *testing.T) {
	r := New(docker.New(),
		func() secrets.Mode { return secrets.MaskAll },
		func() secrets.Mode { return secrets.MaskSensitive })

	for _, tl := range r.List() {
		if tl.ReadOnly || tl.Destructive {
			continue
		}
		if !reversible[tl.Name] {
			t.Errorf("tool %q is neither ReadOnly nor Destructive and not in the reversible allowlist; "+
				"classify it: if it changes state irreversibly set Destructive:true, "+
				"otherwise add it to reversible (and confirm an agent may run it with no grant)", tl.Name)
		}
	}

	// Catch a stale allowlist entry (renamed/removed tool) so the list can't rot.
	registered := map[string]bool{}
	for _, tl := range r.List() {
		registered[tl.Name] = true
	}
	for name := range reversible {
		if !registered[name] {
			t.Errorf("reversible allowlist names %q, which is no longer registered; remove it", name)
		}
	}
}
