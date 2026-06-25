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
// container.inspect masks env, plus sensitive command/label values, through
// envMask(), the same settings knob the HTTP inspect handler honours. A
// non-consented caller (an MCP client / automated agent) gets a hard floor:
// even when the human set masking to "off" for the local UI, secrets are never
// returned raw on that path. The dedicated `oriel mcp` process masks all (it
// passes MaskAll); only interactive UI calls, which carry consent, honour "off".
func registerReads(r *tools.Registry, dc *docker.Client, envMask func() secrets.Mode) {
	// Every tool here is a pure read, mark it ReadOnly so `oriel mcp --read-only`
	// and the MCP read-only hint are accurate (Destructive:false isn't enough,
	// since start/stop mutate without being destructive).
	ro := func(t *tools.Tool) { t.ReadOnly = true; r.Register(t) }
	ro(&tools.Tool{
		Name: "container.list", Title: "List containers", Description: "List all containers with state and ports",
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			return dc.ListContainers(ctx)
		},
	})
	ro(&tools.Tool{
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
			mode := envMask()
			// Floor: a non-consented caller (MCP / agent) never gets raw
			// secrets, even if the human set masking to "off" for the local UI.
			if mode == secrets.MaskOff && !tools.HasConsent(ctx) {
				mode = secrets.MaskSensitive
			}
			d.Env = secrets.MaskEnv(d.Env, mode)
			d.Command = secrets.MaskCommand(d.Command, mode)
			d.Labels = secrets.MaskLabels(d.Labels, mode)
			return d, nil
		},
	})
	ro(&tools.Tool{
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
				if tail > 10000 { // bound the in-memory buffer for an ungated read
					tail = 10000
				}
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

	ro(&tools.Tool{
		Name: "image.list", Title: "List images", Description: "List images with repo tags, size and age",
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			return dc.ListImages(ctx)
		},
	})
	ro(&tools.Tool{
		Name: "volume.list", Title: "List volumes", Description: "List volumes with driver and mountpoint",
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			return dc.ListVolumes(ctx)
		},
	})
	ro(&tools.Tool{
		Name: "network.list", Title: "List networks", Description: "List networks with driver and scope",
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			return dc.ListNetworks(ctx)
		},
	})
	ro(&tools.Tool{
		Name: "stacks.list", Title: "List compose stacks", Description: "List Docker Compose projects with their running/total counts",
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			list, err := dc.ListStacks(ctx)
			if err != nil {
				return nil, err
			}
			// Lean projection for agents: identity + counts + dir, without the full
			// per-container detail (an agent that needs it uses container.list). Keeps
			// the MCP payload small; the UI uses /api/stacks for the full shape.
			type stack struct {
				Name       string `json:"name"`
				Running    int    `json:"running"`
				Total      int    `json:"total"`
				WorkingDir string `json:"workingDir"`
			}
			out := make([]stack, len(list))
			for i, s := range list {
				out[i] = stack{s.Name, s.Running, s.Total, s.WorkingDir}
			}
			return out, nil
		},
	})
	ro(&tools.Tool{
		Name: "system.df", Title: "Disk usage", Description: "Docker disk usage across images, containers and volumes",
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			return dc.SystemUsage(ctx)
		},
	})
	ro(&tools.Tool{
		Name: "colima.status", Title: "Colima status", Description: "Colima VM status: runtime, resources and docker socket",
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			return colima.GetStatus(ctx)
		},
	})
	ro(&tools.Tool{
		Name: "docker.env", Title: "Docker connection env", Description: "DOCKER_HOST + Testcontainers socket override for this machine's docker socket, fixes tools that default to /var/run/docker.sock and miss colima",
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			socket, err := colima.DockerSocketPath(ctx)
			if err != nil {
				return nil, err
			}
			host := "unix://" + socket
			return map[string]any{
				"dockerHost": host,
				"socket":     socket,
				"env": map[string]string{
					"DOCKER_HOST":                           host,
					"TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE": socket,
				},
			}, nil
		},
	})
}
