package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/filters"
)

// SystemUsage summarises what a `docker system prune` would reclaim. Counts and
// sizes mirror what the prune actually removes (stopped containers, dangling
// images, inactive build cache) plus unused volumes shown separately.
type SystemUsage struct {
	StoppedContainers int   `json:"stoppedContainers"`
	ContainersSize    int64 `json:"containersSize"`
	DanglingImages    int   `json:"danglingImages"`
	ImagesSize        int64 `json:"imagesSize"`
	BuildCacheSize    int64 `json:"buildCacheSize"`
	UnusedVolumes     int   `json:"unusedVolumes"`
	VolumesSize       int64 `json:"volumesSize"`
	Reclaimable       int64 `json:"reclaimable"` // excludes volumes (opt-in)
}

func (c *Client) SystemUsage(ctx context.Context) (SystemUsage, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return SystemUsage{}, err
	}
	du, err := cli.DiskUsage(ctx, types.DiskUsageOptions{
		Types: []types.DiskUsageObject{types.ContainerObject, types.ImageObject, types.VolumeObject, types.BuildCacheObject},
	})
	if err != nil {
		return SystemUsage{}, err
	}

	var u SystemUsage
	for _, ct := range du.Containers {
		if ct.State != "running" {
			u.StoppedContainers++
			u.ContainersSize += ct.SizeRw
		}
	}
	for _, img := range du.Images {
		if len(img.RepoTags) == 0 || (len(img.RepoTags) == 1 && img.RepoTags[0] == "<none>:<none>") {
			u.DanglingImages++
			u.ImagesSize += img.Size
		}
	}
	for _, bc := range du.BuildCache {
		if !bc.InUse {
			u.BuildCacheSize += bc.Size
		}
	}
	for _, v := range du.Volumes {
		if v.UsageData != nil && v.UsageData.RefCount == 0 {
			u.UnusedVolumes++
			if v.UsageData.Size > 0 {
				u.VolumesSize += v.UsageData.Size
			}
		}
	}
	u.Reclaimable = u.ContainersSize + u.ImagesSize + u.BuildCacheSize
	return u, nil
}

// PruneResult reports what a system prune removed.
type PruneResult struct {
	Containers int   `json:"containers"`
	Images     int   `json:"images"`
	Networks   int   `json:"networks"`
	Volumes    int   `json:"volumes"`
	Reclaimed  int64 `json:"reclaimed"`
}

// SystemPrune mirrors `docker system prune`: stopped containers, unused networks,
// dangling images, and inactive build cache; unused volumes only when asked.
func (c *Client) SystemPrune(ctx context.Context, includeVolumes bool) (PruneResult, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return PruneResult{}, err
	}
	var res PruneResult

	if rep, err := cli.ContainersPrune(ctx, filters.NewArgs()); err == nil {
		res.Containers = len(rep.ContainersDeleted)
		res.Reclaimed += int64(rep.SpaceReclaimed)
	}
	if rep, err := cli.NetworksPrune(ctx, filters.NewArgs()); err == nil {
		res.Networks = len(rep.NetworksDeleted)
	}
	if rep, err := cli.ImagesPrune(ctx, filters.NewArgs(filters.Arg("dangling", "true"))); err == nil {
		res.Images = len(rep.ImagesDeleted)
		res.Reclaimed += int64(rep.SpaceReclaimed)
	}
	if rep, err := cli.BuildCachePrune(ctx, build.CachePruneOptions{}); err == nil {
		res.Reclaimed += int64(rep.SpaceReclaimed)
	}
	if includeVolumes {
		if rep, err := cli.VolumesPrune(ctx, filters.NewArgs()); err == nil {
			res.Volumes = len(rep.VolumesDeleted)
			res.Reclaimed += int64(rep.SpaceReclaimed)
		}
	}
	return res, nil
}
