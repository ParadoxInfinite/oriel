package docker

import (
	"context"
	"sort"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// ComposeProjectLabel groups containers into compose stacks.
const ComposeProjectLabel = "com.docker.compose.project"

// Container is the frontend-facing DTO; SDK types are not leaked past this
// package so the API contract stays stable across SDK upgrades.
type Container struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	ImageID string `json:"imageId"` // digest, for matching a container to its image
	State   string `json:"state"`
	Status  string `json:"status"`
	Created int64  `json:"created"`
	Ports   []Port `json:"ports"`
	Project string `json:"project"`
}

type Port struct {
	Private uint16 `json:"private"`
	Public  uint16 `json:"public"`
	Type    string `json:"type"`
}

// ListContainers returns all containers (running and stopped).
func (c *Client) ListContainers(ctx context.Context) ([]Container, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return nil, err
	}
	raw, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}
	out := make([]Container, 0, len(raw))
	for _, r := range raw {
		out = append(out, toContainer(r))
	}
	return out, nil
}

func toContainer(r container.Summary) Container {
	name := ""
	if len(r.Names) > 0 {
		name = strings.TrimPrefix(r.Names[0], "/")
	}
	// Dedupe (Docker reports IPv4 + IPv6 mappings separately) and sort, so the
	// list is stable and never reshuffles unless a port actually changes.
	seen := map[Port]bool{}
	ports := make([]Port, 0, len(r.Ports))
	for _, p := range r.Ports {
		pt := Port{Private: p.PrivatePort, Public: p.PublicPort, Type: p.Type}
		if seen[pt] {
			continue
		}
		seen[pt] = true
		ports = append(ports, pt)
	}
	sort.Slice(ports, func(i, j int) bool {
		if ports[i].Private != ports[j].Private {
			return ports[i].Private < ports[j].Private
		}
		if ports[i].Public != ports[j].Public {
			return ports[i].Public < ports[j].Public
		}
		return ports[i].Type < ports[j].Type
	})
	return Container{
		ID:      r.ID,
		Name:    name,
		Image:   r.Image,
		ImageID: r.ImageID,
		State:   string(r.State),
		Status:  r.Status,
		Created: r.Created,
		Ports:   ports,
		Project: r.Labels[ComposeProjectLabel],
	}
}

func (c *Client) StartContainer(ctx context.Context, id string) error {
	cli, err := c.api(ctx)
	if err != nil {
		return err
	}
	return cli.ContainerStart(ctx, id, container.StartOptions{})
}

func (c *Client) StopContainer(ctx context.Context, id string) error {
	cli, err := c.api(ctx)
	if err != nil {
		return err
	}
	return cli.ContainerStop(ctx, id, container.StopOptions{})
}

func (c *Client) RestartContainer(ctx context.Context, id string) error {
	cli, err := c.api(ctx)
	if err != nil {
		return err
	}
	return cli.ContainerRestart(ctx, id, container.StopOptions{})
}

func (c *Client) RemoveContainer(ctx context.Context, id string, force bool) error {
	cli, err := c.api(ctx)
	if err != nil {
		return err
	}
	return cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: force})
}

// ContainerExists reports whether a container with the given id or name exists.
func (c *Client) ContainerExists(ctx context.Context, idOrName string) (bool, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return false, err
	}
	_, err = cli.ContainerInspect(ctx, idOrName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
