package docker

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/filters"
)

// humanBytes formats a byte count for progress lines (e.g. "1.2 GiB").
func humanBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	const units = "KMGT"
	v, i := float64(n), -1
	for v >= 1024 && i < 3 {
		v /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %ciB", v, units[i])
}

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
		// Dangling = no usable tag AND no repo digest. A digest-pinned image
		// (untagged but with a RepoDigest, e.g. compose images) is named and in
		// use, not reclaimable — matching `docker image prune`'s dangling filter.
		named := len(img.RepoDigests) > 0
		for _, t := range img.RepoTags {
			if t != "<none>:<none>" {
				named = true
			}
		}
		if !named {
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

// PruneOptions selects which categories a system prune removes. Each maps to one
// `docker … prune` step; nothing runs unless explicitly selected. BuildCacheAll
// extends the build-cache step from dangling-only to every unused entry.
type PruneOptions struct {
	Containers    bool
	Images        bool
	Networks      bool
	BuildCache    bool
	BuildCacheAll bool
	Volumes       bool
}

// SystemPrune runs the selected prune steps (stopped containers, unused networks,
// dangling images, build cache, unused volumes). Build cache prunes dangling layers
// by default; BuildCacheAll (All:true) also removes cache for existing images — the
// larger amount SystemUsage previews. emit (nil-safe) gets a line per step.
func (c *Client) SystemPrune(ctx context.Context, opts PruneOptions, emit func(string)) (PruneResult, error) {
	if emit == nil {
		emit = func(string) {}
	}
	cli, err := c.api(ctx)
	if err != nil {
		return PruneResult{}, err
	}
	var res PruneResult
	var errs []error
	// fail records a step error and surfaces it on the progress stream. Steps are
	// best-effort across categories, so a later step still runs — unless the
	// context was cancelled (the per-step ctx.Err() guards below stop the run).
	fail := func(what string, err error) {
		errs = append(errs, fmt.Errorf("%s: %w", what, err))
		emit(fmt.Sprintf("  failed: %v", err))
	}

	if opts.Containers && ctx.Err() == nil {
		emit("Removing stopped containers…")
		if rep, err := cli.ContainersPrune(ctx, filters.NewArgs()); err != nil {
			fail("containers", err)
		} else {
			res.Containers = len(rep.ContainersDeleted)
			res.Reclaimed += int64(rep.SpaceReclaimed)
			emit(fmt.Sprintf("  %d removed (%s)", res.Containers, humanBytes(int64(rep.SpaceReclaimed))))
		}
	}
	if opts.Networks && ctx.Err() == nil {
		emit("Removing unused networks…")
		if rep, err := cli.NetworksPrune(ctx, filters.NewArgs()); err != nil {
			fail("networks", err)
		} else {
			res.Networks = len(rep.NetworksDeleted)
			emit(fmt.Sprintf("  %d removed", res.Networks))
		}
	}
	if opts.Images && ctx.Err() == nil {
		emit("Removing dangling images…")
		if rep, err := cli.ImagesPrune(ctx, filters.NewArgs(filters.Arg("dangling", "true"))); err != nil {
			fail("images", err)
		} else {
			res.Images = len(rep.ImagesDeleted)
			res.Reclaimed += int64(rep.SpaceReclaimed)
			emit(fmt.Sprintf("  %d removed (%s)", res.Images, humanBytes(int64(rep.SpaceReclaimed))))
		}
	}
	if opts.BuildCache && ctx.Err() == nil {
		if opts.BuildCacheAll {
			emit("Pruning all build cache…")
		} else {
			emit("Pruning dangling build cache…")
		}
		if rep, err := cli.BuildCachePrune(ctx, build.CachePruneOptions{All: opts.BuildCacheAll}); err != nil {
			fail("build cache", err)
		} else {
			res.Reclaimed += int64(rep.SpaceReclaimed)
			emit(fmt.Sprintf("  reclaimed %s", humanBytes(int64(rep.SpaceReclaimed))))
		}
	}
	if opts.Volumes && ctx.Err() == nil {
		emit("Removing unused volumes…")
		if rep, err := cli.VolumesPrune(ctx, filters.NewArgs()); err != nil {
			fail("volumes", err)
		} else {
			res.Volumes = len(rep.VolumesDeleted)
			res.Reclaimed += int64(rep.SpaceReclaimed)
			emit(fmt.Sprintf("  %d removed (%s)", res.Volumes, humanBytes(int64(rep.SpaceReclaimed))))
		}
	}

	if err := errors.Join(errs...); err != nil {
		emit("Done with errors — some steps failed")
		return res, err
	}
	emit(fmt.Sprintf("Done — reclaimed %s total", humanBytes(res.Reclaimed)))
	return res, nil
}
