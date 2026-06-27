package tools

import (
	"context"
	"errors"
	"testing"
)

// TestRegisterDestructiveNameTripwire locks in the startup guard: a tool whose
// name reads as an irreversible action must carry Destructive, or Register
// panics. This stops a new `*.remove`/`*.prune`/… tool from shipping unflagged
// and inheriting the ungated default for MCP/agent callers.
func TestRegisterDestructiveNameTripwire(t *testing.T) {
	cases := []struct {
		name        string
		destructive bool
		wantPanic   bool
	}{
		{"volume.remove", false, true},   // destructive verb, unflagged → panic
		{"container.prune", false, true}, // ditto
		{"image.delete", false, true},
		{"volume.remove", true, false},   // flagged → fine
		{"container.stop", false, false}, // reversible verb → fine
		{"network.create", false, false}, // reversible verb → fine
		{"container.list", false, false}, // read → fine
	}
	for _, c := range cases {
		func() {
			defer func() {
				if got := recover() != nil; got != c.wantPanic {
					t.Errorf("Register(%q, destructive=%v): panic=%v, want %v", c.name, c.destructive, got, c.wantPanic)
				}
			}()
			NewRegistry(nil).Register(&Tool{
				Name:        c.name,
				Destructive: c.destructive,
				Handler:     func(context.Context, map[string]any) (any, error) { return nil, nil },
			})
		}()
	}
}

// TestAuditRecordsAgentCallsOnly verifies the audit hook fires for agent (no
// consent) calls and not for the operator's own consented UI/palette calls.
func TestAuditRecordsAgentCallsOnly(t *testing.T) {
	r, _ := newTestRegistry()
	var logged []string
	r.SetAuditLog(func(name string, _ map[string]any, _ error) { logged = append(logged, name) })

	if _, err := r.Execute(context.Background(), "container.stop", map[string]any{"id": "web"}); err != nil {
		t.Fatalf("agent call: %v", err)
	}
	if _, err := r.Execute(WithConsent(context.Background()), "container.stop", map[string]any{"id": "web"}); err != nil {
		t.Fatalf("consented call: %v", err)
	}
	if len(logged) != 1 || logged[0] != "container.stop" {
		t.Fatalf("audit should record only the agent call, got %v", logged)
	}
}

// fakeResolver reports existence from a fixed set of "kind/id" keys.
type fakeResolver struct{ known map[string]bool }

func (f fakeResolver) Exists(_ context.Context, kind, id string) (bool, error) {
	return f.known[kind+"/"+id], nil
}

func newTestRegistry() (*Registry, *int) {
	calls := 0
	r := NewRegistry(fakeResolver{known: map[string]bool{"container/web": true}})
	r.Register(&Tool{
		Name:   "container.stop",
		Schema: Schema{Required: []string{"id"}, Props: map[string]Prop{"id": {Type: "string"}}},
		Entity: &EntityRef{Param: "id", Kind: "container"},
		Handler: func(context.Context, map[string]any) (any, error) {
			calls++
			return "stopped", nil
		},
	})
	return r, &calls
}

func TestExecute_UnknownTool(t *testing.T) {
	r, _ := newTestRegistry()
	_, err := r.Execute(context.Background(), "does.not.exist", nil)
	if !errors.Is(err, ErrUnknownTool) {
		t.Fatalf("want ErrUnknownTool, got %v", err)
	}
}

func TestExecute_UnknownEntityRejected(t *testing.T) {
	r, calls := newTestRegistry()
	_, err := r.Execute(context.Background(), "container.stop", map[string]any{"id": "ghost"})
	if err == nil {
		t.Fatal("expected rejection for non-existent container")
	}
	if *calls != 0 {
		t.Fatalf("handler must not run for unknown entity; ran %d times", *calls)
	}
}

func TestExecute_ValidEntityRuns(t *testing.T) {
	r, calls := newTestRegistry()
	got, err := r.Execute(context.Background(), "container.stop", map[string]any{"id": "web"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "stopped" || *calls != 1 {
		t.Fatalf("handler did not run as expected: got=%v calls=%d", got, *calls)
	}
}

func TestSchema_Validate(t *testing.T) {
	s := Schema{
		Required: []string{"id"},
		Props: map[string]Prop{
			"id":    {Type: "string"},
			"force": {Type: "boolean"},
			"mode":  {Type: "string", Enum: []string{"a", "b"}},
		},
	}
	cases := []struct {
		name    string
		args    map[string]any
		wantErr bool
	}{
		{"ok", map[string]any{"id": "x"}, false},
		{"ok with bool", map[string]any{"id": "x", "force": true}, false},
		{"missing required", map[string]any{"force": true}, true},
		{"wrong type", map[string]any{"id": 5.0}, true},
		{"unknown arg", map[string]any{"id": "x", "extra": "y"}, true},
		{"bad enum", map[string]any{"id": "x", "mode": "c"}, true},
		{"good enum", map[string]any{"id": "x", "mode": "a"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := s.Validate(tc.args)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate(%v) err=%v, wantErr=%v", tc.args, err, tc.wantErr)
			}
		})
	}
}
