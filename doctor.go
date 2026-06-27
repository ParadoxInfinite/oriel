package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ParadoxInfinite/oriel/internal/colima"
	"github.com/ParadoxInfinite/oriel/internal/docker"
	"github.com/ParadoxInfinite/oriel/internal/service"
)

// `oriel doctor` is a read-only preflight: it reports the things that actually
// break a reverse-proxy / remote setup (Docker reachability, the running
// instance's base path + allowed hosts, version skew, whether a newer release is
// out, service status) and prints the exact command to fix anything that's wrong.
// It queries the *running* instance over loopback so it reflects the live config,
// not this process's env.

type selfInfo struct {
	Version  string `json:"version"`
	BasePath string `json:"basePath"`
}

func runDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	port := fs.Int("port", 4321, "port the running Oriel instance listens on")
	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Printf("oriel %s\n\n", version)
	problems := 0

	// --- Docker daemon (checked directly, independent of the running server) ---
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if eng := docker.New().EngineInfo(ctx); eng.Reachable {
		doctorLine("✓", "Docker", fmt.Sprintf("reachable, %s (%s)", eng.ServerVersion, eng.Host))
	} else {
		doctorLine("✗", "Docker", "unreachable, is the daemon (or colima) running?")
		problems++
	}

	// --- Docker socket discovery (colima trips up tools that assume the default
	// /var/run/docker.sock: Testcontainers, some SDK clients). Advisory, not a
	// hard failure, `docker context use colima` is a valid alternative. ---
	if socket, err := colima.DockerSocketPath(ctx); err == nil && socket != "" {
		host := "unix://" + socket
		switch env := os.Getenv("DOCKER_HOST"); {
		case env == host:
			doctorLine("✓", "Docker socket", "DOCKER_HOST points at colima")
		case env != "":
			doctorLine("⚠", "Docker socket", fmt.Sprintf("DOCKER_HOST=%s doesn't match colima's %s", env, host))
		case defaultSocketIsColima(socket):
			doctorLine("✓", "Docker socket", "/var/run/docker.sock points at colima")
		default:
			doctorLine("⚠", "Docker socket", "DOCKER_HOST unset and /var/run/docker.sock isn't colima's, Testcontainers / docker SDKs will miss it")
			fmt.Println(`      fix: eval "$(oriel env)"   (or: docker context use colima)`)
		}
	}

	// --- Running instance + its live config ---
	self, err := fetchSelf(*port)
	if err != nil {
		doctorLine("○", "Instance", fmt.Sprintf("not running on 127.0.0.1:%d. Start it with `oriel`, or `oriel service install` to run on login; pass --port if it listens elsewhere", *port))
	} else {
		doctorLine("✓", "Instance", fmt.Sprintf("running on 127.0.0.1:%d, version %s", *port, self.Version))
		if version != "dev" && self.Version != "" && self.Version != version {
			doctorLine("⚠", "Version", fmt.Sprintf("this binary is %s but the running service is %s, restart it to upgrade", version, self.Version))
		}

		// --- Version currency: how the running service compares to the latest
		// release. Reuses the instance's cached /api/update check (shared with the
		// GUI), so repeated doctor runs don't hammer the GitHub rate limit. ---
		if st, uerr := getUpdateStatus(*port); uerr == nil {
			switch {
			case st.Current == "" || st.Current == "dev":
				doctorLine("○", "Updates", "running a local/dev build, version check skipped")
			case st.Error != "":
				doctorLine("○", "Updates", "couldn't reach GitHub to check ("+st.Error+")")
			case st.UpdateAvailable:
				doctorLine("⚠", "Updates", fmt.Sprintf("%s installed, v%s available", st.Current, st.Latest))
				switch {
				case st.PackageManager == "homebrew":
					fmt.Println("      fix: brew upgrade oriel")
				case st.Managed:
					fmt.Println("      fix: oriel update")
				default:
					fmt.Printf("      fix: download v%s from https://github.com/ParadoxInfinite/oriel/releases/latest\n", st.Latest)
				}
			default:
				doctorLine("✓", "Updates", fmt.Sprintf("%s is the latest", st.Current))
			}
		}

		atRoot := self.BasePath == "" || self.BasePath == "/"
		if atRoot {
			doctorLine("✓", "Base path", "served at host root")
		} else {
			doctorLine("✓", "Base path", fmt.Sprintf("%s (reverse-proxy sub-path)", self.BasePath))
		}

		hosts, herr := remoteGet(*port)
		switch {
		case herr != nil:
			doctorLine("⚠", "Allowed hosts", "could not read, "+herr.Error())
		case len(hosts) == 0 && !atRoot:
			doctorLine("✗", "Allowed hosts", "none set, but a sub-path proxy is configured, /api will 403 over the proxy")
			fmt.Println("      fix: oriel remote allow <your-proxy-hostname>")
			problems++
		case len(hosts) == 0:
			doctorLine("✓", "Allowed hosts", "none (loopback only)")
		default:
			doctorLine("✓", "Allowed hosts", strings.Join(hosts, ", "))
		}
	}

	// --- Service install ---
	if service.IsManaged() {
		doctorLine("✓", "Service", "installed and managing this binary (in-app/CLI update available)")
	} else {
		doctorLine("○", "Service", "not a managed service, install with: oriel service install")
	}

	if problems > 0 {
		return fmt.Errorf("%d problem(s) found", problems)
	}
	return nil
}

func doctorLine(glyph, label, detail string) {
	fmt.Printf("  %s  %-14s %s\n", glyph, label, detail)
}

// dockerHostHint returns a one-line warning when the current shell's DOCKER_HOST
// won't reach colima (so docker SDKs / Testcontainers would silently miss it), or
// "" when the environment is fine or colima isn't in use. It reads this process's
// own env, so only a CLI invocation (not the managed service) sees the user's
// interactive shell. Reused by the post-update nudge so a stale env surfaces
// without a separate `oriel doctor` run.
func dockerHostHint(ctx context.Context) string {
	socket, err := colima.DockerSocketPath(ctx)
	if err != nil || socket == "" {
		return "" // no colima socket → nothing colima-specific to advise
	}
	host := "unix://" + socket
	switch env := os.Getenv("DOCKER_HOST"); {
	case env == host || defaultSocketIsColima(socket):
		return ""
	case env != "":
		return fmt.Sprintf("DOCKER_HOST=%s doesn't match colima's %s. Fix: eval \"$(oriel env)\"", env, host)
	default:
		return "DOCKER_HOST is unset and /var/run/docker.sock isn't colima's, so docker SDKs / Testcontainers will miss colima. Fix: eval \"$(oriel env)\""
	}
}

// defaultSocketIsColima reports whether /var/run/docker.sock is a symlink to
// colima's socket (so tools hitting the default path actually reach colima).
func defaultSocketIsColima(colimaSocket string) bool {
	target, err := os.Readlink("/var/run/docker.sock")
	return err == nil && target == colimaSocket
}

func fetchSelf(port int) (selfInfo, error) {
	var s selfInfo
	resp, err := remoteClient.Get(fmt.Sprintf("http://127.0.0.1:%d/api/self", port))
	if err != nil {
		return s, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return s, fmt.Errorf("status %s", resp.Status)
	}
	return s, json.NewDecoder(resp.Body).Decode(&s)
}
