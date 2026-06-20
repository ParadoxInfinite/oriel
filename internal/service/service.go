// Package service installs colima-gui as a per-user background service so it
// starts on login and stays running: a launchd LaunchAgent on macOS, a systemd
// user service on Linux. It binds to 127.0.0.1 like the foreground server.
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
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	switch sub {
	case "install":
		return install(*port)
	case "uninstall", "remove":
		return uninstall()
	case "status":
		return status()
	default:
		return usage()
	}
}

func usage() error {
	fmt.Println(`Usage: oriel service <command>

Commands:
  install [--port N]   install and start the background service (default port 4321)
  uninstall            stop and remove the service
  status               show whether the service is installed and running`)
	return nil
}

func install(port int) error {
	bin, err := os.Executable()
	if err != nil {
		return err
	}
	bin, _ = filepath.EvalSymlinks(bin)

	switch runtime.GOOS {
	case "darwin":
		return installDarwin(bin, port)
	case "linux":
		return installLinux(bin, port)
	default:
		return fmt.Errorf("service install is not supported on %s", runtime.GOOS)
	}
}

func uninstall() error {
	switch runtime.GOOS {
	case "darwin":
		return uninstallDarwin()
	case "linux":
		return uninstallLinux()
	default:
		return fmt.Errorf("service is not supported on %s", runtime.GOOS)
	}
}

func status() error {
	switch runtime.GOOS {
	case "darwin":
		return statusDarwin()
	case "linux":
		return statusLinux()
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
