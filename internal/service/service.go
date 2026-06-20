// Package service installs Oriel as a background service so it starts
// automatically and stays running: a launchd LaunchAgent on macOS, and a systemd
// service on Linux (a per-user unit, or a system unit with --system / when run as
// root). It binds to 127.0.0.1 like the foreground server.
package service

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

const label = "com.oriel"

// Run handles the `service` subcommand: install | uninstall | status.
func Run(args []string) error {
	if len(args) == 0 {
		return usage()
	}
	sub := args[0]
	fs := flag.NewFlagSet("service", flag.ContinueOnError)
	port := fs.Int("port", 4321, "port the service listens on (127.0.0.1)")
	system := fs.Bool("system", false, "install a system-wide service (Linux; implied when run as root)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	switch sub {
	case "install":
		return install(*port, *system)
	case "uninstall", "remove":
		return uninstall(*system)
	case "status":
		return status(*system)
	default:
		return usage()
	}
}

func usage() error {
	fmt.Println(`Usage: oriel service <command>

Commands:
  install [--port N] [--system]   install and start the background service
  uninstall [--system]            stop and remove the service
  status [--system]               show whether the service is installed and running

  --system   Linux only: install a system unit (starts on boot, runs as the
             service user). Implied when run as root. Otherwise a per-user
             systemd unit is installed (needs an active user session).`)
	return nil
}

func install(port int, system bool) error {
	bin, err := os.Executable()
	if err != nil {
		return err
	}
	bin, _ = filepath.EvalSymlinks(bin)

	switch runtime.GOOS {
	case "darwin":
		return installDarwin(bin, port)
	case "linux":
		return installLinux(bin, port, system)
	default:
		return fmt.Errorf("service install is not supported on %s", runtime.GOOS)
	}
}

func uninstall(system bool) error {
	switch runtime.GOOS {
	case "darwin":
		return uninstallDarwin()
	case "linux":
		return uninstallLinux(system)
	default:
		return fmt.Errorf("service is not supported on %s", runtime.GOOS)
	}
}

func status(system bool) error {
	switch runtime.GOOS {
	case "darwin":
		return statusDarwin()
	case "linux":
		return statusLinux(system)
	default:
		return fmt.Errorf("service is not supported on %s", runtime.GOOS)
	}
}

// render writes tmpl with data to path (0644), creating parent dirs.
func render(path, tmpl string, data any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	t, err := template.New("unit").Parse(tmpl)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, data)
}

// run executes a command and returns combined output on error for context.
func run(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return fmt.Errorf("%s: %s", err, msg)
	}
	return nil
}
