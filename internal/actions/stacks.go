package actions

import (
	"context"

	"github.com/ParadoxInfinite/oriel/internal/docker"
	"github.com/ParadoxInfinite/oriel/internal/settings"
	"github.com/ParadoxInfinite/oriel/internal/tools"
)

// registerStacks wires compose-project lifecycle as tools. Each blocks until the
// compose action finishes and returns the collected output — the synchronous
// path the registry (and so MCP) needs. The UI drives the same actions over a
// stream (op tray) for live progress. stack.down removes containers + networks,
// so it's Destructive; start/stop/restart are reversible. Deploying a not-yet-
// running project stays out of the registry — that's the discovery/file-path
// flow, not an action on a known stack name.
func registerStacks(r *tools.Registry, dc *docker.Client) {
	nameArg := tools.Schema{
		Required: []string{"name"},
		Props: map[string]tools.Prop{
			"name": {Type: "string", Description: "compose project (stack) name"},
		},
	}
	ref := func() *tools.EntityRef { return &tools.EntityRef{Param: "name", Kind: "stack"} }
	run := func(action string) func(context.Context, map[string]any) (any, error) {
		return func(ctx context.Context, a map[string]any) (any, error) {
			out, err := dc.RunCompose(ctx, a["name"].(string), action)
			if err != nil {
				return nil, err
			}
			return map[string]any{"ok": true, "output": out}, nil
		}
	}

	r.Register(&tools.Tool{
		Name: "stack.start", Title: "Start stack", Description: "Start a stopped compose project",
		Schema: nameArg, Entity: ref(),
		Handler: run("start"),
	})
	r.Register(&tools.Tool{
		Name: "stack.stop", Title: "Stop stack", Description: "Stop a running compose project",
		Schema: nameArg, Entity: ref(),
		Handler: run("stop"),
	})
	r.Register(&tools.Tool{
		Name: "stack.restart", Title: "Restart stack", Description: "Restart a compose project",
		Schema: nameArg, Entity: ref(),
		Handler: run("restart"),
	})
	r.Register(&tools.Tool{
		Name: "stack.down", Title: "Take stack down", Description: "Stop and remove a compose project's containers and networks",
		Destructive: true,
		Schema:      nameArg, Entity: ref(),
		Handler: run("down"),
	})

	// stack.alias is a display-only label write, not a Docker action: it sets the
	// Oriel name shown for a project; the real compose name is unchanged. No Entity
	// check — an alias can target a discovered (not-yet-running) project too.
	r.Register(&tools.Tool{
		Name: "stack.alias", Title: "Rename stack in Oriel", Description: "Set or clear the Oriel display alias for a compose project (display only; the real project name is unchanged)",
		Schema: tools.Schema{
			Required: []string{"name"},
			Props: map[string]tools.Prop{
				"name":  {Type: "string", Description: "real compose project name"},
				"alias": {Type: "string", Description: "display alias; empty clears it"},
			},
		},
		Handler: func(_ context.Context, a map[string]any) (any, error) {
			alias, _ := a["alias"].(string)
			if err := settings.SetAlias(a["name"].(string), alias); err != nil {
				return nil, err
			}
			return okResult, nil
		},
	})
}
