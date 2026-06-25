// Package docker talks to the Docker Engine API exposed by Colima's unix
// socket. The client is created lazily so the app starts even when the VM is
// down. The connection is cached once established and survives colima restarts
// as long as the socket path is stable; a profile or socket-path change needs an
// Oriel restart.
package docker

import (
	"context"
	"sync"

	"github.com/docker/docker/client"

	"github.com/ParadoxInfinite/oriel/internal/colima"
)

// Client wraps the Docker SDK with lazy, socket-aware connection management.
type Client struct {
	mu  sync.Mutex
	cli *client.Client
}

// New returns an unconnected client. The connection is established on first use.
func New() *Client { return &Client{} }

// api returns a connected Docker SDK client. The connection is cached once
// established: the SDK dials the unix socket per request, so a cached client
// survives colima restarts, and we avoid spawning the `colima` CLI (socket
// discovery) on every call, critical now that the recorder samples each second.
func (c *Client) api(ctx context.Context) (*client.Client, error) {
	c.mu.Lock()
	if c.cli != nil {
		cli := c.cli
		c.mu.Unlock()
		return cli, nil
	}
	c.mu.Unlock()

	// Discover the socket and dial WITHOUT holding the lock: colima's CLI probe
	// can be slow or hang, and holding the mutex would freeze every docker call
	// (the recorder samples each second) behind it.
	opts := []client.Opt{client.WithAPIVersionNegotiation()}
	var sock string
	switch s, err := colima.DockerSocketPath(ctx); {
	case err == nil:
		// colima is running, talk to its socket.
		sock = s
		opts = append(opts, client.WithHost("unix://"+sock))
	case colima.Installed():
		// colima is present but not ready; don't cache a fallback, retry next
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

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cli != nil {
		// Another caller dialed while we were probing, keep theirs, drop ours.
		_ = cli.Close()
		return c.cli, nil
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
