package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ParadoxInfinite/oriel/internal/colima"
)

// `oriel env` prints the Docker connection environment for this machine's actual
// socket. Colima puts its socket at ~/.colima/<profile>/docker.sock, but tools
// that assume /var/run/docker.sock (Testcontainers, some SDK clients) miss it.
// `eval "$(oriel env)"` points the current shell at the right one.
func runEnv(_ []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	socket, err := colima.DockerSocketPath(ctx)
	if err != nil {
		return fmt.Errorf("could not find a colima docker socket (is colima running? if you use Docker Desktop you don't need this): %w", err)
	}
	if socket == "" {
		return fmt.Errorf("colima reported no docker socket (is it running?)")
	}
	host := "unix://" + socket

	// This output is meant to be run as `eval "$(oriel env)"`, so quote with POSIX
	// single quotes, not Go's %q. %q produces a double-quoted string in which a
	// `$`, backtick, or `"` in the path would still be expanded by the shell;
	// single-quoting (with the '\'' escape for an embedded quote) is literal.
	fmt.Printf("export DOCKER_HOST=%s\n", shellQuote(host))
	fmt.Printf("export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=%s\n", shellQuote(socket))
	fmt.Println(`# Point your shell at colima's docker:  eval "$(oriel env)"`)
	fmt.Println("# (many tools default to /var/run/docker.sock and miss colima's socket)")
	return nil
}

// shellQuote wraps s in POSIX single quotes so it survives `eval` literally, with
// any embedded single quote rendered as the standard '\” sequence.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
