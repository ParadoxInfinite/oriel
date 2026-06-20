package docker

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/docker/docker/api/types/container"

	"github.com/ParadoxInfinite/oriel/internal/execstream"
)

// Compose label keys used to reconstruct the `docker compose` invocation.
const (
	composeConfigFilesLabel = "com.docker.compose.project.config_files"
	composeWorkingDirLabel  = "com.docker.compose.project.working_dir"
)

// Stack is a discovered compose project (grouped by the project label).
type Stack struct {
	Name        string      `json:"name"`
	Running     int         `json:"running"`
	Total       int         `json:"total"`
	ConfigFiles string      `json:"configFiles"`
	WorkingDir  string      `json:"workingDir"`
	Containers  []Container `json:"containers"`
}

// ListStacks groups all containers by their compose project label.
func (c *Client) ListStacks(ctx context.Context) ([]Stack, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return nil, err
	}
	raw, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	byProject := map[string]*Stack{}
	for _, r := range raw {
		proj := r.Labels[ComposeProjectLabel]
		if proj == "" {
			continue
		}
		s := byProject[proj]
		if s == nil {
			s = &Stack{
				Name:        proj,
				ConfigFiles: r.Labels[composeConfigFilesLabel],
				WorkingDir:  r.Labels[composeWorkingDirLabel],
			}
			byProject[proj] = s
		}
		s.Total++
		if string(r.State) == "running" {
			s.Running++
		}
		s.Containers = append(s.Containers, toContainer(r))
	}

	out := make([]Stack, 0, len(byProject))
	for _, s := range byProject {
		sort.Slice(s.Containers, func(i, j int) bool { return s.Containers[i].Name < s.Containers[j].Name })
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (c *Client) findStack(ctx context.Context, name string) (*Stack, error) {
	stacks, err := c.ListStacks(ctx)
	if err != nil {
		return nil, err
	}
	for i := range stacks {
		if stacks[i].Name == name {
			return &stacks[i], nil
		}
	}
	return nil, fmt.Errorf("stack %q not found", name)
}

var composeActions = map[string][]string{
	"up":      {"up", "-d"},
	"down":    {"down"},
	"start":   {"start"},
	"stop":    {"stop"},
	"restart": {"restart"},
}

// StreamCompose runs a compose action against a discovered stack, streaming
// output. The stack's config-files and working-dir labels are replayed so
// compose targets exactly the original project definition.
func (c *Client) StreamCompose(ctx context.Context, project, action string) (<-chan string, <-chan error, error) {
	sub, ok := composeActions[action]
	if !ok {
		return nil, nil, fmt.Errorf("invalid compose action %q", action)
	}
	stack, err := c.findStack(ctx, project)
	if err != nil {
		return nil, nil, err
	}

	args := []string{"compose", "-p", stack.Name}
	if stack.WorkingDir != "" {
		args = append(args, "--project-directory", stack.WorkingDir)
	}
	for _, f := range strings.Split(stack.ConfigFiles, ",") {
		if f = strings.TrimSpace(f); f != "" {
			args = append(args, "-f", f)
		}
	}
	args = append(args, sub...)

	return execstream.Run(ctx, "docker", args...)
}

// ComposeUpFile deploys a discovered compose project by file path — it has no
// containers yet, so it can't be found by label. --project-directory is set so
// an adjacent .env and relative paths resolve; -p is passed only when the file
// doesn't declare its own name, so the resulting labels match what we discovered.
func (c *Client) ComposeUpFile(ctx context.Context, dir, file, name string, ownName bool) (<-chan string, <-chan error, error) {
	args := []string{"compose", "--project-directory", dir, "-f", file}
	if !ownName && name != "" {
		args = append(args, "-p", name)
	}
	args = append(args, "up", "-d")
	return execstream.Run(ctx, "docker", args...)
}
