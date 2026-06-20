// Package execstream runs a command and streams its combined output line by
// line, used for long-running CLI operations (colima lifecycle, docker compose)
// that the UI displays as live progress.
package execstream

import (
	"bufio"
	"context"
	"os/exec"
)

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

	go func() {
		defer close(lines)
		scanners := []*bufio.Scanner{bufio.NewScanner(stdout), bufio.NewScanner(stderr)}
		done := make(chan struct{}, len(scanners))
		for _, sc := range scanners {
			go func(sc *bufio.Scanner) {
				for sc.Scan() {
					select {
					case lines <- sc.Text():
					case <-ctx.Done():
						done <- struct{}{}
						return
					}
				}
				done <- struct{}{}
			}(sc)
		}
		for range scanners {
			<-done
		}
		errc <- cmd.Wait()
		close(errc)
	}()

	return lines, errc, nil
}
