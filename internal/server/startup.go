package server

import (
	"context"
	"log"
	"strings"
	"time"
)

// LogStartup writes a one-line-per-fact config summary to the log on boot, plus a
// warning when the config looks like it will 403 behind a reverse proxy. Headless
// operators see this in journalctl and don't have to guess what's configured.
func (s *Server) LogStartup(url string) {
	cfg := loadSettings()
	log.Printf("oriel %s listening on %s", s.version, url)

	base := "(root)"
	if s.base != "/" {
		base = s.base
	}
	hosts := "(none, loopback only)"
	if len(cfg.AllowedHosts) > 0 {
		hosts = strings.Join(cfg.AllowedHosts, ", ")
	}
	log.Printf("  base-path=%s  allowed-hosts=%s", base, hosts)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if eng := s.docker.EngineInfo(ctx); eng.Reachable {
		log.Printf("  docker=reachable (%s)", eng.ServerVersion)
	} else {
		log.Printf("  docker=unreachable, start the daemon (or colima)")
	}

	if s.base != "/" && len(cfg.AllowedHosts) == 0 {
		log.Print("  ⚠ base-path is set but no allowed hosts, /api will 403 over a reverse proxy; fix: oriel remote allow <host>")
	}
}
