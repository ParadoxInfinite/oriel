package main

import (
	"context"
	"fmt"
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

	fmt.Printf("export DOCKER_HOST=%q\n", host)
	fmt.Printf("export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=%q\n", socket)
	fmt.Println(`# Point your shell at colima's docker:  eval "$(oriel env)"`)
	fmt.Println("# (many tools default to /var/run/docker.sock and miss colima's socket)")
	return nil
}
