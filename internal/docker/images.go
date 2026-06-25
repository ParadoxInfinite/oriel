package docker

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
)

type Image struct {
	ID         string   `json:"id"`
	Tags       []string `json:"tags"`
	Size       int64    `json:"size"`
	Created    int64    `json:"created"`
	Containers int64    `json:"containers"`
}

func (c *Client) ListImages(ctx context.Context) ([]Image, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return nil, err
	}
	raw, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]Image, 0, len(raw))
	for _, r := range raw {
		// Untagged images pulled by digest (e.g. compose `image: repo:tag@sha256:…`)
		// have no RepoTags but a RepoDigest "repo@sha256:…", show that, not <none>,
		// so digest-pinned images (immich, etc.) are identifiable. Only a true
		// dangling layer (no tags AND no digests) falls back to <none>.
		tags := r.RepoTags
		if len(tags) == 0 {
			if len(r.RepoDigests) > 0 {
				tags = r.RepoDigests
			} else {
				tags = []string{"<none>"}
			}
		}
		out = append(out, Image{
			ID:         r.ID,
			Tags:       tags,
			Size:       r.Size,
			Created:    r.Created,
			Containers: r.Containers,
		})
	}
	return out, nil
}

// TagImage adds a repository:tag reference to an existing image, so a digest-pinned
// (untagged) image gains a readable name.
func (c *Client) TagImage(ctx context.Context, id, ref string) error {
	cli, err := c.api(ctx)
	if err != nil {
		return err
	}
	return cli.ImageTag(ctx, id, ref)
}

func (c *Client) RemoveImage(ctx context.Context, id string, force bool) error {
	cli, err := c.api(ctx)
	if err != nil {
		return err
	}
	_, err = cli.ImageRemove(ctx, id, image.RemoveOptions{Force: force, PruneChildren: true})
	return err
}

// PruneImages removes dangling images and reports what was reclaimed.
func (c *Client) PruneImages(ctx context.Context) (int, int64, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return 0, 0, err
	}
	report, err := cli.ImagesPrune(ctx, filters.NewArgs())
	if err != nil {
		return 0, 0, err
	}
	return len(report.ImagesDeleted), int64(report.SpaceReclaimed), nil
}

// PullImage pulls ref and invokes emit for each progress object in the stream.
func (c *Client) PullImage(ctx context.Context, ref string, emit func(map[string]any)) error {
	cli, err := c.api(ctx)
	if err != nil {
		return err
	}
	rc, err := cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return err
	}
	defer rc.Close()

	dec := json.NewDecoder(rc)
	for {
		var msg map[string]any
		if err := dec.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		emit(msg)
	}
}

// SearchResult is one Docker Hub search hit.
type SearchResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Stars       int    `json:"stars"`
	Official    bool   `json:"official"`
}

// SearchImages queries the registry (Docker Hub) through the daemon, so it uses
// the same network path as a pull and needs no extra HTTP client or CORS dance.
func (c *Client) SearchImages(ctx context.Context, term string, limit int) ([]SearchResult, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return nil, err
	}
	raw, err := cli.ImageSearch(ctx, term, registry.SearchOptions{Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(raw))
	for _, r := range raw {
		out = append(out, SearchResult{Name: r.Name, Description: r.Description, Stars: r.StarCount, Official: r.IsOfficial})
	}
	return out, nil
}

func (c *Client) ImageExists(ctx context.Context, idOrRef string) (bool, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return false, err
	}
	_, err = cli.ImageInspect(ctx, idOrRef)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
