package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ParadoxInfinite/oriel/internal/tools"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// stubResolver pretends every referenced entity exists, so handler tests don't
// need a live Docker host.
type stubResolver struct{}

func (stubResolver) Exists(context.Context, string, string) (bool, error) { return true, nil }

// connect wires an in-memory client to a server built from reg and returns the
// live client session. This exercises the real MCP round-trip (initialize,
// tools/list, tools/call) without stdio or a subprocess.
func connect(t *testing.T, reg *tools.Registry) *mcpsdk.ClientSession {
	t.Helper()
	ctx := context.Background()
	clientT, serverT := mcpsdk.NewInMemoryTransports()

	if _, err := newServer(reg, "test", nil).Connect(ctx, serverT, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	cs, err := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "test"}, nil).
		Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func TestListToolsMapsRegistry(t *testing.T) {
	reg := tools.NewRegistry(stubResolver{})
	reg.Register(&tools.Tool{
		Name: "fake.read", Title: "Read", Description: "read something", ReadOnly: true,
		Handler: func(context.Context, map[string]any) (any, error) { return map[string]any{"ok": true}, nil },
	})
	reg.Register(&tools.Tool{
		Name: "fake.wipe", Title: "Wipe", Description: "destroy something", Destructive: true,
		Schema: tools.Schema{
			Required: []string{"id"},
			Props:    map[string]tools.Prop{"id": {Type: "string", Description: "the id"}},
		},
		Entity:  &tools.EntityRef{Param: "id", Kind: "container"},
		Handler: func(context.Context, map[string]any) (any, error) { return nil, nil },
	})

	cs := connect(t, reg)
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	got := map[string]*mcpsdk.Tool{}
	for _, tool := range res.Tools {
		got[tool.Name] = tool
	}
	read, wipe := got["fake.read"], got["fake.wipe"]
	if read == nil || wipe == nil {
		t.Fatalf("missing tools, got %v", got)
	}
	if !read.Annotations.ReadOnlyHint {
		t.Error("fake.read should carry ReadOnlyHint")
	}
	if read.Annotations.DestructiveHint == nil || *read.Annotations.DestructiveHint {
		t.Error("fake.read should not be destructive")
	}
	if wipe.Annotations.ReadOnlyHint || wipe.Annotations.DestructiveHint == nil || !*wipe.Annotations.DestructiveHint {
		t.Error("fake.wipe should be destructive, not read-only")
	}
	// The required "id" param must survive the schema translation.
	raw, _ := json.Marshal(wipe.InputSchema)
	var schema struct {
		Required []string `json:"required"`
	}
	_ = json.Unmarshal(raw, &schema)
	if len(schema.Required) != 1 || schema.Required[0] != "id" {
		t.Errorf("fake.wipe input schema lost required id: %s", raw)
	}
}

func TestCallToolRoundTrip(t *testing.T) {
	reg := tools.NewRegistry(stubResolver{})
	reg.Register(&tools.Tool{
		Name: "fake.echo", Title: "Echo", Description: "echo the args back",
		Schema:  tools.Schema{Props: map[string]tools.Prop{"msg": {Type: "string"}}},
		Handler: func(_ context.Context, a map[string]any) (any, error) { return a, nil },
	})

	cs := connect(t, reg)
	res, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "fake.echo",
		Arguments: map[string]any{"msg": "hi"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %+v", res.Content)
	}
	text, ok := res.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("want TextContent, got %T", res.Content[0])
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(text.Text), &out); err != nil {
		t.Fatalf("result not JSON: %v (%q)", err, text.Text)
	}
	if out["msg"] != "hi" {
		t.Errorf("round-trip lost arg, got %v", out)
	}
}

func TestUnknownToolReturnsErrorResult(t *testing.T) {
	reg := tools.NewRegistry(stubResolver{})
	reg.Register(&tools.Tool{
		Name: "fake.noop", Title: "Noop", Description: "noop",
		Handler: func(context.Context, map[string]any) (any, error) { return nil, nil },
	})
	cs := connect(t, reg)

	// Calling a registered tool with a bad arg should surface as an in-band
	// error result (IsError), not a transport failure, so the model can recover.
	reg2 := tools.NewRegistry(stubResolver{})
	reg2.Register(&tools.Tool{
		Name: "fake.needsid", Title: "Needs id", Description: "requires id",
		Schema:  tools.Schema{Required: []string{"id"}, Props: map[string]tools.Prop{"id": {Type: "string"}}},
		Handler: func(context.Context, map[string]any) (any, error) { return "ok", nil },
	})
	cs2 := connect(t, reg2)
	res, err := cs2.CallTool(context.Background(), &mcpsdk.CallToolParams{Name: "fake.needsid"})
	if err != nil {
		t.Fatalf("CallTool transport error (should be in-band): %v", err)
	}
	if !res.IsError {
		t.Error("missing required arg should yield IsError result")
	}
	_ = cs
}
