package docker

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

// StreamLogs follows a container's logs, invoking emit per line until ctx is
// cancelled. Non-TTY logs are multiplexed and demuxed via stdcopy; TTY logs are
// a raw stream.
func (c *Client) StreamLogs(ctx context.Context, id string, tail int, emit func(stream, line string)) error {
	cli, err := c.api(ctx)
	if err != nil {
		return err
	}
	insp, err := cli.ContainerInspect(ctx, id)
	if err != nil {
		return err
	}

	rc, err := cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       strconv.Itoa(tail),
	})
	if err != nil {
		return err
	}
	defer rc.Close()

	if insp.Config != nil && insp.Config.Tty {
		_, err = io.Copy(&lineWriter{stream: "stdout", emit: emit}, rc)
		return err
	}
	_, err = stdcopy.StdCopy(
		&lineWriter{stream: "stdout", emit: emit},
		&lineWriter{stream: "stderr", emit: emit},
		rc,
	)
	return err
}

// lineWriter buffers writes and emits complete newline-delimited lines.
type lineWriter struct {
	stream string
	buf    []byte
	emit   func(stream, line string)
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimRight(string(w.buf[:i]), "\r")
		w.buf = w.buf[i+1:]
		w.emit(w.stream, line)
	}
	return len(p), nil
}
