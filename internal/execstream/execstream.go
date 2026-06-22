// Package execstream runs a command and streams its combined output line by
// line, used for long-running CLI operations (colima lifecycle, docker compose)
// that the UI displays as live progress.
package execstream

import (
	"bufio"
	"context"
	"io"
	"os/exec"
)

// maxLine bounds a single output line; bufio.Scanner's default 64 KiB cap
// silently drops the rest of a longer line (and the rest of that stream).
const maxLine = 1 << 20 // 1 MiB

// Run starts name with args and returns a channel of output lines (closed when
// the command exits) plus an error channel that yields the single terminal
// error (or nil) after the lines channel closes.
func Run(ctx context.Context, name string, args ...string) (<-chan string, <-chan error, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	lines := make(chan string, 32)
	errc := make(chan error, 1)

	newScanner := func(rc io.Reader) *bufio.Scanner {
		sc := bufio.NewScanner(rc)
		sc.Buffer(make([]byte, 0, 64*1024), maxLine)
		return sc
	}

	go func() {
		defer close(lines)
		scanners := []*bufio.Scanner{newScanner(stdout), newScanner(stderr)}
		done := make(chan error, len(scanners))
		for _, sc := range scanners {
			go func(sc *bufio.Scanner) {
				for sc.Scan() {
					select {
					case lines <- sc.Text():
					case <-ctx.Done():
						done <- nil
						return
					}
				}
				done <- sc.Err() // nil on clean EOF; non-nil e.g. on a too-long line
			}(sc)
		}
		var scanErr error
		for range scanners {
			if e := <-done; e != nil && scanErr == nil {
				scanErr = e
			}
		}
		// The process's own exit error is the more meaningful one; fall back to a
		// scanner error (truncation) when the command otherwise exited cleanly.
		if werr := cmd.Wait(); werr != nil {
			errc <- werr
		} else {
			errc <- scanErr
		}
		close(errc)
	}()

	return lines, errc, nil
}
