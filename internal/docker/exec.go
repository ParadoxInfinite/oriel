package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

// ExecSession is a live interactive exec (a PTY running a command) attached to a
// container. Read and write Conn for the terminal stream; call Close when done.
type ExecSession struct {
	ID   string
	Conn types.HijackedResponse
	c    *Client
}

// Exec starts an interactive shell-style exec in a container and attaches to it
// with a TTY, so Conn carries the raw terminal stream both ways. The caller pumps
// bytes between Conn and the browser and must Close the session.
func (c *Client) Exec(ctx context.Context, id string, cmd []string, rows, cols uint) (*ExecSession, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return nil, err
	}
	created, err := cli.ContainerExecCreate(ctx, id, container.ExecOptions{
		Cmd:          cmd,
		Tty:          true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return nil, err
	}
	att, err := cli.ContainerExecAttach(ctx, created.ID, container.ExecAttachOptions{Tty: true})
	if err != nil {
		return nil, err
	}
	if rows > 0 && cols > 0 {
		_ = cli.ContainerExecResize(ctx, created.ID, container.ResizeOptions{Height: rows, Width: cols})
	}
	return &ExecSession{ID: created.ID, Conn: att, c: c}, nil
}

// Resize updates the exec's PTY dimensions (rows × cols).
func (s *ExecSession) Resize(ctx context.Context, rows, cols uint) error {
	cli, err := s.c.api(ctx)
	if err != nil {
		return err
	}
	return cli.ContainerExecResize(ctx, s.ID, container.ResizeOptions{Height: rows, Width: cols})
}

// Close releases the attached connection.
func (s *ExecSession) Close() { s.Conn.Close() }
