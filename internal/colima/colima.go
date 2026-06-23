// Package colima wraps the `colima` CLI, which has no API. It exposes the VM
// status (resources, runtime, docker socket) and lifecycle controls.
package colima

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ParadoxInfinite/oriel/internal/execstream"
)

// Profile is the colima profile this GUI manages. v1 targets the default.
const Profile = "default"

// Status is the merged view from `colima list --json` (always available) and
// `colima status --json` (only when the VM is running, adds the socket).
type Status struct {
	Engine       string `json:"engine"` // "colima" | "docker"
	Running      bool   `json:"running"`
	Profile      string `json:"profile"`
	Runtime      string `json:"runtime"`
	Arch         string `json:"arch"`
	CPU          int    `json:"cpu"`
	Memory       int64  `json:"memory"`
	Disk         int64  `json:"disk"`
	Kubernetes   bool   `json:"kubernetes"`
	DockerSocket string `json:"dockerSocket"`
	MountType    string `json:"mountType"`
	Driver       string `json:"driver"`
	Version      string `json:"version,omitempty"`
}

// Installed reports whether the colima CLI is on PATH. When it isn't, the GUI
// drives a generic Docker engine instead of managing a colima VM.
func Installed() bool {
	_, err := exec.LookPath("colima")
	return err == nil
}

// listEntry mirrors one JSON-lines record from `colima list --json`.
type listEntry struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Arch    string `json:"arch"`
	CPUs    int    `json:"cpus"`
	Memory  int64  `json:"memory"`
	Disk    int64  `json:"disk"`
	Runtime string `json:"runtime"`
}

// statusEntry mirrors `colima status --json` (emitted only when running).
type statusEntry struct {
	DisplayName  string `json:"display_name"`
	Driver       string `json:"driver"`
	Arch         string `json:"arch"`
	Runtime      string `json:"runtime"`
	MountType    string `json:"mount_type"`
	DockerSocket string `json:"docker_socket"`
	Kubernetes   bool   `json:"kubernetes"`
	CPU          int    `json:"cpu"`
	Memory       int64  `json:"memory"`
	Disk         int64  `json:"disk"`
}

// GetStatus returns the current status of the managed profile. A stopped VM is
// a normal result (Running=false), not an error.
func GetStatus(ctx context.Context) (Status, error) {
	st := Status{Profile: Profile}

	entry, err := listProfile(ctx, Profile)
	if err != nil {
		return st, err
	}
	if entry == nil {
		// Profile has never been created — treat as stopped/empty.
		return st, nil
	}

	st.Runtime = entry.Runtime
	st.Arch = entry.Arch
	st.CPU = entry.CPUs
	st.Memory = entry.Memory
	st.Disk = entry.Disk
	st.Running = strings.EqualFold(entry.Status, "Running")

	if st.Running {
		// Enrich with socket/driver details, available only while running.
		if se, err := statusDetail(ctx); err == nil {
			st.DockerSocket = se.DockerSocket
			st.MountType = se.MountType
			st.Driver = se.Driver
			st.Kubernetes = se.Kubernetes
			if se.Runtime != "" {
				st.Runtime = se.Runtime
			}
		}
	}
	return st, nil
}

// DockerSocketPath returns the filesystem path of the docker socket (without
// the unix:// scheme), or an error if the VM is not running.
func DockerSocketPath(ctx context.Context) (string, error) {
	st, err := GetStatus(ctx)
	if err != nil {
		return "", err
	}
	if !st.Running {
		return "", fmt.Errorf("colima is not running")
	}
	if st.DockerSocket == "" {
		return "", fmt.Errorf("colima reported no docker socket")
	}
	return strings.TrimPrefix(st.DockerSocket, "unix://"), nil
}

// listProfile runs `colima list --json` and returns the entry for name, or nil
// if not present. The command emits JSON Lines (one object per profile).
func listProfile(ctx context.Context, name string) (*listEntry, error) {
	out, err := exec.CommandContext(ctx, "colima", "list", "--json").Output()
	if err != nil {
		return nil, fmt.Errorf("colima list: %w", err)
	}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e listEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		if e.Name == name {
			return &e, nil
		}
	}
	return nil, nil
}

// statusDetail runs `colima status --json` for the managed profile.
func statusDetail(ctx context.Context) (*statusEntry, error) {
	out, err := exec.CommandContext(ctx, "colima", "status", "--json", "-p", Profile).Output()
	if err != nil {
		return nil, fmt.Errorf("colima status: %w", err)
	}
	var e statusEntry
	if err := json.Unmarshal(out, &e); err != nil {
		return nil, fmt.Errorf("parse colima status: %w", err)
	}
	return &e, nil
}

// validActions guards the lifecycle endpoint against arbitrary subcommands.
var validActions = map[string]bool{"start": true, "stop": true, "restart": true}

// Stream runs a lifecycle action (start|stop|restart) and streams its output.
// colima logs progress to stderr, which execstream merges in.
func Stream(ctx context.Context, action string) (<-chan string, <-chan error, error) {
	if !validActions[action] {
		return nil, nil, fmt.Errorf("invalid action %q", action)
	}
	return execstream.Run(ctx, "colima", action, "-p", Profile)
}

// Run executes a lifecycle action and blocks until it finishes, returning the
// collected output. It's the synchronous form for the tool registry (MCP); the
// UI uses Stream for live progress.
func Run(ctx context.Context, action string) ([]string, error) {
	lines, errc, err := Stream(ctx, action)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for l := range lines {
		out = append(out, l)
	}
	if err := <-errc; err != nil {
		return out, err
	}
	return out, nil
}
