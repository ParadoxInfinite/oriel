package docker

import (
	"context"
	"strings"
)

// ContainerDetail is the curated inspect payload the UI shows in a detail panel.
type ContainerDetail struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Image         string            `json:"image"`
	ImageID       string            `json:"imageId"`
	Created       string            `json:"created"`
	State         string            `json:"state"`
	Running       bool              `json:"running"`
	StartedAt     string            `json:"startedAt"`
	ExitCode      int               `json:"exitCode"`
	Health        string            `json:"health"`
	RestartPolicy string            `json:"restartPolicy"`
	Command       string            `json:"command"`
	WorkingDir    string            `json:"workingDir"`
	Env           []string          `json:"env"`
	Labels        map[string]string `json:"labels"`
	Mounts        []MountInfo       `json:"mounts"`
	Networks      []NetInfo         `json:"networks"`
}

type MountInfo struct {
	Type        string `json:"type"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Name        string `json:"name"`
	RW          bool   `json:"rw"`
}

type NetInfo struct {
	Name       string `json:"name"`
	IPAddress  string `json:"ipAddress"`
	Gateway    string `json:"gateway"`
	MacAddress string `json:"macAddress"`
}

func (c *Client) InspectContainer(ctx context.Context, id string) (ContainerDetail, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return ContainerDetail{}, err
	}
	r, err := cli.ContainerInspect(ctx, id)
	if err != nil {
		return ContainerDetail{}, err
	}

	d := ContainerDetail{
		ID:      r.ID,
		Name:    strings.TrimPrefix(r.Name, "/"),
		ImageID: r.Image,
		Command: strings.TrimSpace(r.Path + " " + strings.Join(r.Args, " ")),
	}
	if r.Created != "" {
		d.Created = r.Created
	}
	if r.State != nil {
		d.State = r.State.Status
		d.Running = r.State.Running
		d.StartedAt = r.State.StartedAt
		d.ExitCode = r.State.ExitCode
		if r.State.Health != nil {
			d.Health = r.State.Health.Status
		}
	}
	if r.HostConfig != nil {
		d.RestartPolicy = string(r.HostConfig.RestartPolicy.Name)
	}
	if r.Config != nil {
		d.Image = r.Config.Image
		d.WorkingDir = r.Config.WorkingDir
		d.Env = r.Config.Env
		d.Labels = r.Config.Labels
	}
	for _, m := range r.Mounts {
		d.Mounts = append(d.Mounts, MountInfo{Type: string(m.Type), Source: m.Source, Destination: m.Destination, Name: m.Name, RW: m.RW})
	}
	if r.NetworkSettings != nil {
		for name, ep := range r.NetworkSettings.Networks {
			if ep == nil {
				continue
			}
			d.Networks = append(d.Networks, NetInfo{Name: name, IPAddress: ep.IPAddress, Gateway: ep.Gateway, MacAddress: ep.MacAddress})
		}
	}
	return d, nil
}
