package docker

import "context"

// EngineInfo is a generic snapshot of the Docker daemon, used to report status
// when there's no colima VM to describe.
type EngineInfo struct {
	Reachable     bool
	Host          string
	NCPU          int
	MemTotal      int64
	Architecture  string
	ServerVersion string
	Driver        string
	OS            string
}

// EngineInfo pings the daemon via `docker info`. A nil/zero result (Reachable
// false) means the engine is down or unreachable, never an error to the caller.
func (c *Client) EngineInfo(ctx context.Context) EngineInfo {
	cli, err := c.api(ctx)
	if err != nil {
		return EngineInfo{}
	}
	info, err := cli.Info(ctx)
	if err != nil {
		return EngineInfo{Host: cli.DaemonHost()}
	}
	return EngineInfo{
		Reachable:     true,
		Host:          cli.DaemonHost(),
		NCPU:          info.NCPU,
		MemTotal:      info.MemTotal,
		Architecture:  info.Architecture,
		ServerVersion: info.ServerVersion,
		Driver:        info.Driver,
		OS:            info.OperatingSystem,
	}
}
