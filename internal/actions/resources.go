package actions

import (
	"context"

	"github.com/ParadoxInfinite/oriel/internal/docker"
	"github.com/ParadoxInfinite/oriel/internal/tools"
)

// okResult is the shared success payload returned by mutating tools. It is
// read-only: handlers return it for JSON encoding and must never mutate it.
var okResult = map[string]any{"ok": true}

func registerImages(r *tools.Registry, dc *docker.Client) {
	r.Register(&tools.Tool{
		Name: "image.remove", Title: "Remove image", Description: "Remove an image",
		Destructive: true,
		Schema: tools.Schema{
			Required: []string{"id"},
			Props: map[string]tools.Prop{
				"id":    {Type: "string", Description: "image id or reference"},
				"force": {Type: "boolean", Description: "force removal even if in use"},
			},
		},
		Entity: &tools.EntityRef{Param: "id", Kind: "image"},
		Handler: func(ctx context.Context, a map[string]any) (any, error) {
			force, _ := a["force"].(bool)
			return okResult, dc.RemoveImage(ctx, a["id"].(string), force)
		},
	})
	r.Register(&tools.Tool{
		Name: "image.tag", Title: "Tag image", Description: "Add a repository:tag to an image",
		Schema: tools.Schema{
			Required: []string{"id", "ref"},
			Props: map[string]tools.Prop{
				"id":  {Type: "string", Description: "image id"},
				"ref": {Type: "string", Description: "new repository:tag reference"},
			},
		},
		Entity: &tools.EntityRef{Param: "id", Kind: "image"},
		Handler: func(ctx context.Context, a map[string]any) (any, error) {
			return okResult, dc.TagImage(ctx, a["id"].(string), a["ref"].(string))
		},
	})
	r.Register(&tools.Tool{
		Name: "image.prune", Title: "Prune images", Description: "Remove dangling images",
		Destructive: true,
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			n, reclaimed, err := dc.PruneImages(ctx)
			return map[string]any{"removed": n, "reclaimed": reclaimed}, err
		},
	})
}

func registerVolumes(r *tools.Registry, dc *docker.Client) {
	r.Register(&tools.Tool{
		Name: "volume.remove", Title: "Remove volume", Description: "Remove a volume",
		Destructive: true,
		Schema: tools.Schema{
			Required: []string{"name"},
			Props: map[string]tools.Prop{
				"name":  {Type: "string", Description: "volume name"},
				"force": {Type: "boolean", Description: "force removal"},
			},
		},
		Entity: &tools.EntityRef{Param: "name", Kind: "volume"},
		Handler: func(ctx context.Context, a map[string]any) (any, error) {
			force, _ := a["force"].(bool)
			return okResult, dc.RemoveVolume(ctx, a["name"].(string), force)
		},
	})
	r.Register(&tools.Tool{
		Name: "volume.prune", Title: "Prune volumes", Description: "Remove unused volumes",
		Destructive: true,
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			n, reclaimed, err := dc.PruneVolumes(ctx)
			return map[string]any{"removed": n, "reclaimed": reclaimed}, err
		},
	})
}

func registerNetworks(r *tools.Registry, dc *docker.Client) {
	r.Register(&tools.Tool{
		Name: "network.remove", Title: "Remove network", Description: "Remove a network",
		Destructive: true,
		Schema: tools.Schema{
			Required: []string{"id"},
			Props:    map[string]tools.Prop{"id": {Type: "string", Description: "network id or name"}},
		},
		Entity: &tools.EntityRef{Param: "id", Kind: "network"},
		Handler: func(ctx context.Context, a map[string]any) (any, error) {
			return okResult, dc.RemoveNetwork(ctx, a["id"].(string))
		},
	})
}
