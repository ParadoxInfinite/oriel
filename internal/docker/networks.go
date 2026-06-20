package docker

import (
	"context"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type Network struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Driver   string `json:"driver"`
	Scope    string `json:"scope"`
	Internal bool   `json:"internal"`
	Created  int64  `json:"created"`
}

func (c *Client) ListNetworks(ctx context.Context) ([]Network, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return nil, err
	}
	raw, err := cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]Network, 0, len(raw))
	for _, n := range raw {
		out = append(out, Network{
			ID:       n.ID,
			Name:     n.Name,
			Driver:   n.Driver,
			Scope:    n.Scope,
			Internal: n.Internal,
			Created:  n.Created.Unix(),
		})
	}
	return out, nil
}

func (c *Client) RemoveNetwork(ctx context.Context, id string) error {
	cli, err := c.api(ctx)
	if err != nil {
		return err
	}
	return cli.NetworkRemove(ctx, id)
}

func (c *Client) NetworkExists(ctx context.Context, idOrName string) (bool, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return false, err
	}
	_, err = cli.NetworkInspect(ctx, idOrName, network.InspectOptions{})
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
