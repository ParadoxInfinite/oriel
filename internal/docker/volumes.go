package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

type Volume struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
	Scope      string `json:"scope"`
	CreatedAt  string `json:"createdAt"`
}

// VolumePreview is one prune candidate: an unused volume and (best-effort) the
// space it would reclaim.
type VolumePreview struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"createdAt"`
}

// PruneableVolumes lists volumes no container references, exactly what a prune
// would remove, with sizes from the disk-usage API (0 when a driver can't
// report it, so the prune still works, the total is just approximate).
func (c *Client) PruneableVolumes(ctx context.Context) ([]VolumePreview, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := cli.VolumeList(ctx, volume.ListOptions{
		Filters: filters.NewArgs(filters.Arg("dangling", "true")),
	})
	if err != nil {
		return nil, err
	}
	sizes := map[string]int64{}
	if du, err := cli.DiskUsage(ctx, types.DiskUsageOptions{
		Types: []types.DiskUsageObject{types.VolumeObject},
	}); err == nil {
		for _, v := range du.Volumes {
			if v != nil && v.UsageData != nil {
				sizes[v.Name] = v.UsageData.Size
			}
		}
	}
	out := make([]VolumePreview, 0, len(resp.Volumes))
	for _, v := range resp.Volumes {
		out = append(out, VolumePreview{Name: v.Name, Size: sizes[v.Name], CreatedAt: v.CreatedAt})
	}
	return out, nil
}

func (c *Client) ListVolumes(ctx context.Context) ([]Volume, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := cli.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]Volume, 0, len(resp.Volumes))
	for _, v := range resp.Volumes {
		out = append(out, Volume{
			Name:       v.Name,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
			Scope:      v.Scope,
			CreatedAt:  v.CreatedAt,
		})
	}
	return out, nil
}

func (c *Client) RemoveVolume(ctx context.Context, name string, force bool) error {
	cli, err := c.api(ctx)
	if err != nil {
		return err
	}
	return cli.VolumeRemove(ctx, name, force)
}

func (c *Client) PruneVolumes(ctx context.Context) (int, int64, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return 0, 0, err
	}
	report, err := cli.VolumesPrune(ctx, filters.NewArgs())
	if err != nil {
		return 0, 0, err
	}
	return len(report.VolumesDeleted), int64(report.SpaceReclaimed), nil
}

func (c *Client) VolumeExists(ctx context.Context, name string) (bool, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return false, err
	}
	_, err = cli.VolumeInspect(ctx, name)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
