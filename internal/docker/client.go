// Package docker talks to the Docker Engine API exposed by Colima's unix
// socket. The client is created lazily so the app starts even when the VM is
// down, and re-dials if the socket path changes.
package docker

import (
	"context"
	"sync"

	"github.com/docker/docker/client"

	"github.com/ParadoxInfinite/oriel/internal/colima"
)

// Client wraps the Docker SDK with lazy, socket-aware connection management.
type Client struct {
	mu     sync.Mutex
	cli    *client.Client
	socket string
}

// New returns an unconnected client. The connection is established on first use.
func New() *Client { return &Client{} }

// api returns a connected Docker SDK client. The connection is cached once
// established: the SDK dials the unix socket per request, so a cached client
// survives colima restarts, and we avoid spawning the `colima` CLI (socket
// discovery) on every call — critical now that the recorder samples each second.
func (c *Client) api(ctx context.Context) (*client.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cli != nil {
		return c.cli, nil
	}
	opts := []client.Opt{client.WithAPIVersionNegotiation()}
	switch sock, err := colima.DockerSocketPath(ctx); {
	case err == nil:
		// colima is running — talk to its socket.
		opts = append(opts, client.WithHost("unix://"+sock))
		c.socket = sock
	case colima.Installed():
		// colima is present but not ready; don't cache a fallback — retry next
		// call once the VM is up, rather than latch onto the wrong socket.
		return nil, err
	default:
		// No colima on this host: drive the standard Docker engine via the
		// environment (DOCKER_HOST, else the platform's default socket).
		opts = append(opts, client.FromEnv)
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, err
	}
	c.cli = cli
	return cli, nil
}

// Close releases the underlying connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cli != nil {
		err := c.cli.Close()
		c.cli = nil
		return err
	}
	return nil
}
