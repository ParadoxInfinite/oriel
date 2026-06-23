// Package actions wires concrete Docker/Colima operations into the generic tool
// Registry and supplies the entity resolver. Keeping this separate lets the
// tools package stay dependency-free and unit-testable.
package actions

import (
	"context"
	"fmt"

	"github.com/ParadoxInfinite/oriel/internal/docker"
	"github.com/ParadoxInfinite/oriel/internal/secrets"
	"github.com/ParadoxInfinite/oriel/internal/tools"
)

// New builds the registry with all builtin tools registered. envMask supplies
// the current env-masking mode (from settings) so container.inspect can mask
// secrets in its output; it is read per call, not captured once.
func New(dc *docker.Client, envMask func() secrets.Mode) *tools.Registry {
	r := tools.NewRegistry(resolver{dc})
	registerContainers(r, dc)
	registerImages(r, dc)
	registerVolumes(r, dc)
	registerNetworks(r, dc)
	registerStacks(r, dc)
	registerColima(r)
	registerReads(r, dc, envMask)
	return r
}

// resolver checks live existence of referenced entities, backed by Docker.
type resolver struct{ dc *docker.Client }

func (r resolver) Exists(ctx context.Context, kind, idOrName string) (bool, error) {
	switch kind {
	case "container":
		return r.dc.ContainerExists(ctx, idOrName)
	case "image":
		return r.dc.ImageExists(ctx, idOrName)
	case "volume":
		return r.dc.VolumeExists(ctx, idOrName)
	case "network":
		return r.dc.NetworkExists(ctx, idOrName)
	case "stack":
		return r.dc.StackExists(ctx, idOrName)
	default:
		return false, fmt.Errorf("unknown entity kind %q", kind)
	}
}

func registerContainers(r *tools.Registry, dc *docker.Client) {
	idArg := tools.Schema{
		Required: []string{"id"},
		Props: map[string]tools.Prop{
			"id": {Type: "string", Description: "container id or name"},
		},
	}
	ref := func() *tools.EntityRef { return &tools.EntityRef{Param: "id", Kind: "container"} }

	r.Register(&tools.Tool{
		Name: "container.start", Title: "Start container", Description: "Start a stopped container",
		Schema: idArg, Entity: ref(),
		Handler: func(ctx context.Context, a map[string]any) (any, error) {
			return okResult, dc.StartContainer(ctx, a["id"].(string))
		},
	})
	r.Register(&tools.Tool{
		Name: "container.stop", Title: "Stop container", Description: "Stop a running container",
		Schema: idArg, Entity: ref(),
		Handler: func(ctx context.Context, a map[string]any) (any, error) {
			return okResult, dc.StopContainer(ctx, a["id"].(string))
		},
	})
	r.Register(&tools.Tool{
		Name: "container.restart", Title: "Restart container", Description: "Restart a container",
		Schema: idArg, Entity: ref(),
		Handler: func(ctx context.Context, a map[string]any) (any, error) {
			return okResult, dc.RestartContainer(ctx, a["id"].(string))
		},
	})
	r.Register(&tools.Tool{
		Name: "container.remove", Title: "Remove container", Description: "Remove a container",
		Destructive: true,
		Schema: tools.Schema{
			Required: []string{"id"},
			Props: map[string]tools.Prop{
				"id":    {Type: "string", Description: "container id or name"},
				"force": {Type: "boolean", Description: "force removal of a running container"},
			},
		},
		Entity: ref(),
		Handler: func(ctx context.Context, a map[string]any) (any, error) {
			force, _ := a["force"].(bool)
			return okResult, dc.RemoveContainer(ctx, a["id"].(string), force)
		},
	})
}
