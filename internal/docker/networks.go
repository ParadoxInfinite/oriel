package docker

import (
	"context"
	"sort"
	"time"

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

// NetworkDetail is the curated inspect payload the UI's detail view shows: the
// addressing (IPAM subnet/gateway) and which containers are attached, plus the
// driver options and labels.
type NetworkDetail struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Scope      string            `json:"scope"`
	Internal   bool              `json:"internal"`
	Created    string            `json:"created"`
	IPAM       []IPAMEntry       `json:"ipam"`
	Containers []NetworkEndpoint `json:"containers"`
	Options    map[string]string `json:"options,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

type IPAMEntry struct {
	Subnet  string `json:"subnet,omitempty"`
	Gateway string `json:"gateway,omitempty"`
}

type NetworkEndpoint struct {
	ContainerID string `json:"containerId"`
	Name        string `json:"name"`
	IPv4        string `json:"ipv4,omitempty"`
	IPv6        string `json:"ipv6,omitempty"`
	MAC         string `json:"mac,omitempty"`
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

// CreateNetwork creates a user-defined network. Driver defaults to bridge.
// Engine-level addressing (custom subnets, gateways) is intentionally left to the
// engine, this is the everyday "give me a network to attach containers to" path.
func (c *Client) CreateNetwork(ctx context.Context, name, driver string, internal bool) (string, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return "", err
	}
	if driver == "" {
		driver = "bridge"
	}
	resp, err := cli.NetworkCreate(ctx, name, network.CreateOptions{Driver: driver, Internal: internal})
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (c *Client) InspectNetwork(ctx context.Context, id string) (*NetworkDetail, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return nil, err
	}
	n, err := cli.NetworkInspect(ctx, id, network.InspectOptions{})
	if err != nil {
		return nil, err
	}
	d := &NetworkDetail{
		ID:       n.ID,
		Name:     n.Name,
		Driver:   n.Driver,
		Scope:    n.Scope,
		Internal: n.Internal,
		Created:  n.Created.Format(time.RFC3339),
		Options:  n.Options,
		Labels:   n.Labels,
	}
	for _, cfg := range n.IPAM.Config {
		d.IPAM = append(d.IPAM, IPAMEntry{Subnet: cfg.Subnet, Gateway: cfg.Gateway})
	}
	for cid, ep := range n.Containers {
		d.Containers = append(d.Containers, NetworkEndpoint{
			ContainerID: cid,
			Name:        ep.Name,
			IPv4:        ep.IPv4Address,
			IPv6:        ep.IPv6Address,
			MAC:         ep.MacAddress,
		})
	}
	// Map iteration order is random; sort by name so the detail view is stable.
	sort.Slice(d.Containers, func(i, j int) bool { return d.Containers[i].Name < d.Containers[j].Name })
	return d, nil
}

// ConnectContainer attaches a container to a network. DisconnectContainer detaches
// it (force detaches even a running container).
func (c *Client) ConnectContainer(ctx context.Context, networkID, containerID string) error {
	cli, err := c.api(ctx)
	if err != nil {
		return err
	}
	return cli.NetworkConnect(ctx, networkID, containerID, nil)
}

func (c *Client) DisconnectContainer(ctx context.Context, networkID, containerID string, force bool) error {
	cli, err := c.api(ctx)
	if err != nil {
		return err
	}
	return cli.NetworkDisconnect(ctx, networkID, containerID, force)
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
