package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ParadoxInfinite/oriel/internal/docker"
	"github.com/ParadoxInfinite/oriel/internal/service"
)

// `oriel doctor` is a read-only preflight: it reports the things that actually
// break a reverse-proxy / remote setup (Docker reachability, the running
// instance's base path + allowed hosts, version skew, service status) and prints
// the exact command to fix anything that's wrong. It queries the *running*
// instance over loopback so it reflects the live config, not this process's env.

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
		doctorLine("✓", "Docker", fmt.Sprintf("reachable — %s (%s)", eng.ServerVersion, eng.Host))
	} else {
		doctorLine("✗", "Docker", "unreachable — is the daemon (or colima) running?")
		problems++
	}

	// --- Running instance + its live config ---
	self, err := fetchSelf(*port)
	if err != nil {
		doctorLine("○", "Instance", fmt.Sprintf("none reachable on 127.0.0.1:%d — pass --port if it listens elsewhere", *port))
	} else {
		doctorLine("✓", "Instance", fmt.Sprintf("running on 127.0.0.1:%d, version %s", *port, self.Version))
		if version != "dev" && self.Version != "" && self.Version != version {
			doctorLine("⚠", "Version", fmt.Sprintf("this binary is %s but the running service is %s — restart it to upgrade", version, self.Version))
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
			doctorLine("⚠", "Allowed hosts", "could not read — "+herr.Error())
		case len(hosts) == 0 && !atRoot:
			doctorLine("✗", "Allowed hosts", "none set, but a sub-path proxy is configured — /api will 403 over the proxy")
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
		doctorLine("○", "Service", "not a managed service — install with: oriel service install")
	}

	if problems > 0 {
		return fmt.Errorf("%d problem(s) found", problems)
	}
	return nil
}

func doctorLine(glyph, label, detail string) {
	fmt.Printf("  %s  %-14s %s\n", glyph, label, detail)
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
