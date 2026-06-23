package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/ParadoxInfinite/oriel/internal/tools"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func readTool(name string, result any) *tools.Tool {
	return &tools.Tool{
		Name: name, ReadOnly: true,
		Schema:  tools.Schema{Required: []string{"id"}, Props: map[string]tools.Prop{"id": {Type: "string"}}},
		Entity:  &tools.EntityRef{Param: "id", Kind: "container"},
		Handler: func(context.Context, map[string]any) (any, error) { return result, nil },
	}
}

func TestResourcesAndPrompts(t *testing.T) {
	reg := tools.NewRegistry(stubResolver{})
	reg.Register(readTool("container.logs", map[string]any{"lines": []string{"boom"}}))
	reg.Register(readTool("container.inspect", map[string]any{"name": "web"}))

	cs := connect(t, reg)
	ctx := context.Background()

	// Prompts: all three canned diagnostics are advertised.
	pr, err := cs.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	got := map[string]bool{}
	for _, p := range pr.Prompts {
		got[p.Name] = true
	}
	for _, want := range []string{"diagnose-container", "fix-docker-connection", "reclaim-disk"} {
		if !got[want] {
			t.Errorf("prompt %q not advertised", want)
		}
	}

	gp, err := cs.GetPrompt(ctx, &mcpsdk.GetPromptParams{Name: "fix-docker-connection"})
	if err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	if len(gp.Messages) == 0 {
		t.Fatal("fix-docker-connection returned no messages")
	}

	// Resource templates: backed by the in-scope read tools.
	rt, err := cs.ListResourceTemplates(ctx, nil)
	if err != nil {
		t.Fatalf("ListResourceTemplates: %v", err)
	}
	tmpls := map[string]bool{}
	for _, x := range rt.ResourceTemplates {
		tmpls[x.Name] = true
	}
	if !tmpls["container-logs"] || !tmpls["container-inspect"] {
		t.Errorf("missing resource templates, got %v", tmpls)
	}

	// Reading a resource runs its backing tool through Execute (same validated path).
	rr, err := cs.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: "oriel://container/web/logs"})
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	if len(rr.Contents) == 0 || !strings.Contains(rr.Contents[0].Text, "boom") {
		t.Errorf("resource did not return tool output, got %+v", rr.Contents)
	}
}
