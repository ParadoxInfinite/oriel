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

// StreamLogs reads a container's logs, invoking emit per line. With follow=true it
// tails live until ctx is cancelled; with follow=false it returns one historical
// batch and stops. `until` (RFC3339Nano) bounds the batch to lines before that
// moment — the cursor for lazy-loading older lines. Timestamps are always on so
// each line carries the cursor; the ts is split out of the displayed text.
func (c *Client) StreamLogs(ctx context.Context, id string, tail int, follow bool, until string, emit func(stream, ts, line string)) error {
	cli, err := c.api(ctx)
	if err != nil {
		return err
	}
	insp, err := cli.ContainerInspect(ctx, id)
	if err != nil {
		return err
	}

	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     follow,
		Tail:       strconv.Itoa(tail),
		Until:      until,
	}
	rc, err := cli.ContainerLogs(ctx, id, opts)
	if err != nil {
		return err
	}
	defer rc.Close()

	onLine := func(stream, raw string) {
		ts, line := splitTimestamp(raw)
		emit(stream, ts, line)
	}

	if insp.Config != nil && insp.Config.Tty {
		lw := &lineWriter{stream: "stdout", emit: onLine}
		_, err = io.Copy(lw, rc)
		lw.flush()
		return err
	}
	out := &lineWriter{stream: "stdout", emit: onLine}
	errw := &lineWriter{stream: "stderr", emit: onLine}
	_, err = stdcopy.StdCopy(out, errw, rc)
	out.flush()
	errw.flush()
	return err
}

// splitTimestamp peels the RFC3339Nano prefix that Timestamps:true prepends to
// each line ("2024-01-02T15:04:05.123Z message"). Returns ("", raw) if absent.
func splitTimestamp(raw string) (ts, line string) {
	if i := strings.IndexByte(raw, ' '); i > 0 {
		return raw[:i], raw[i+1:]
	}
	return "", raw
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

// flush emits any trailing line the stream ended without a newline — otherwise
// the last line of a one-shot/crashed container is lost.
func (w *lineWriter) flush() {
	if len(w.buf) == 0 {
		return
	}
	line := strings.TrimRight(string(w.buf), "\r")
	w.buf = nil
	w.emit(w.stream, line)
}
