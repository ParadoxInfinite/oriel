package actions

import (
	"context"

	"github.com/ParadoxInfinite/oriel/internal/colima"
	"github.com/ParadoxInfinite/oriel/internal/docker"
	"github.com/ParadoxInfinite/oriel/internal/secrets"
	"github.com/ParadoxInfinite/oriel/internal/tools"
)

// registerReads wires the read-only query tools: list/inspect/logs and the
// system-level status calls. None mutate state, so all are Destructive:false and
// safe to expose to an assistant or MCP client without a grant.
//
// container.inspect masks env values through envMask() — the same settings knob
// the HTTP inspect handler honours — so secrets never reach an LLM/MCP consumer
// even though the human UI can reveal them locally.
func registerReads(r *tools.Registry, dc *docker.Client, envMask func() secrets.Mode) {
	r.Register(&tools.Tool{
		Name: "container.list", Title: "List containers", Description: "List all containers with state and ports",
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			return dc.ListContainers(ctx)
		},
	})
	r.Register(&tools.Tool{
		Name: "container.inspect", Title: "Inspect container", Description: "Full container detail; env values are masked",
		Schema: tools.Schema{
			Required: []string{"id"},
			Props:    map[string]tools.Prop{"id": {Type: "string", Description: "container id or name"}},
		},
		Entity: &tools.EntityRef{Param: "id", Kind: "container"},
		Handler: func(ctx context.Context, a map[string]any) (any, error) {
			d, err := dc.InspectContainer(ctx, a["id"].(string))
			if err != nil {
				return nil, err
			}
			d.Env = secrets.MaskEnv(d.Env, envMask())
			return d, nil
		},
	})
	r.Register(&tools.Tool{
		Name: "container.logs", Title: "Read container logs", Description: "Return the most recent log lines (no follow)",
		Schema: tools.Schema{
			Required: []string{"id"},
			Props: map[string]tools.Prop{
				"id":   {Type: "string", Description: "container id or name"},
				"tail": {Type: "number", Description: "number of trailing lines to return (default 100)"},
			},
		},
		Entity: &tools.EntityRef{Param: "id", Kind: "container"},
		Handler: func(ctx context.Context, a map[string]any) (any, error) {
			tail := 100
			if n, ok := a["tail"].(float64); ok && n > 0 {
				tail = int(n)
			}
			type line struct {
				Stream string `json:"stream"`
				TS     string `json:"ts"`
				Line   string `json:"line"`
			}
			lines := []line{}
			err := dc.StreamLogs(ctx, a["id"].(string), tail, false, "", func(stream, ts, l string) {
				lines = append(lines, line{stream, ts, l})
			})
			if err != nil {
				return nil, err
			}
			return map[string]any{"lines": lines}, nil
		},
	})

	r.Register(&tools.Tool{
		Name: "image.list", Title: "List images", Description: "List images with repo tags, size and age",
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			return dc.ListImages(ctx)
		},
	})
	r.Register(&tools.Tool{
		Name: "volume.list", Title: "List volumes", Description: "List volumes with driver and mountpoint",
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			return dc.ListVolumes(ctx)
		},
	})
	r.Register(&tools.Tool{
		Name: "network.list", Title: "List networks", Description: "List networks with driver and scope",
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			return dc.ListNetworks(ctx)
		},
	})
	r.Register(&tools.Tool{
		Name: "stacks.list", Title: "List compose stacks", Description: "List Docker Compose projects and their containers",
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			return dc.ListStacks(ctx)
		},
	})
	r.Register(&tools.Tool{
		Name: "system.df", Title: "Disk usage", Description: "Docker disk usage across images, containers and volumes",
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			return dc.SystemUsage(ctx)
		},
	})
	r.Register(&tools.Tool{
		Name: "colima.status", Title: "Colima status", Description: "Colima VM status: runtime, resources and docker socket",
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			return colima.GetStatus(ctx)
		},
	})
}
