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
	"syscall"
	"text/template"
)

const label = "com.oriel"

// unitFiles returns the service config paths Oriel might have installed for the
// current platform/user.
func unitFiles() []string {
	switch runtime.GOOS {
	case "darwin":
		if p, err := darwinPlistPath(); err == nil {
			return []string{p}
		}
	case "linux":
		paths := []string{systemUnitPath}
		if p, err := linuxUserUnitPath(); err == nil {
			paths = append(paths, p)
		}
		return paths
	}
	return nil
}

// IsManaged reports whether the running executable is the one an installed Oriel
// service launches — i.e. self-update is safe to replace it and restart. Manual
// `./oriel` runs (a different binary path) return false.
func IsManaged() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	exe, _ = filepath.EvalSymlinks(exe)
	for _, p := range unitFiles() {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		s := string(data)
		// Match the exe as the actual launched program, not just any substring:
		// a ProgramArguments <string> element (launchd plist) or the ExecStart
		// program (systemd unit, always followed by " --no-open …").
		if strings.Contains(s, "<string>"+exe+"</string>") || strings.Contains(s, "ExecStart="+exe+" ") {
			return true
		}
	}
	return false
}

// Restart restarts the installed service so a freshly-replaced binary takes
// effect. On macOS a clean exit is enough (launchd KeepAlive relaunches us); on
// Linux systemd needs an explicit restart (the unit is Restart=on-failure).
func Restart() error {
	switch runtime.GOOS {
	case "darwin":
		// SIGTERM triggers the server's graceful shutdown; launchd then relaunches.
		return syscall.Kill(os.Getpid(), syscall.SIGTERM)
	case "linux":
		// systemctl stops (SIGTERM) then starts a fresh instance. Don't wait — it
		// kills this very process mid-command.
		return exec.Command("systemctl", sctl(useSystem(false), "restart", "oriel.service")...).Start()
	default:
		return fmt.Errorf("self-restart is not supported on %s", runtime.GOOS)
	}
}

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
  status [--system]               show install/run status

  --system   Linux only: install a system unit (starts on boot, runs as the
             service user). Implied when run as root. Otherwise a per-user
             systemd unit is installed (needs an active user session).

To serve behind a reverse proxy after install, configure the running instance:
  oriel config base-path /oriel       set the sub-path (restarts to apply)
  oriel remote allow <hostname>       allow a host to reach /api over the network
  oriel doctor                        check everything is wired up`)
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
