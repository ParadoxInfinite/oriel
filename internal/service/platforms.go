package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ---- macOS (launchd LaunchAgent) ----

const darwinPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>{{.Label}}</string>
  <key>ProgramArguments</key>
  <array>
    <string>{{.Bin}}</string>
    <string>--no-open</string>
    <string>--port</string>
    <string>{{.Port}}</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>EnvironmentVariables</key>
  <dict>
    <key>PATH</key><string>{{.Path}}</string>
  </dict>
  <key>StandardOutPath</key><string>{{.Log}}</string>
  <key>StandardErrorPath</key><string>{{.Log}}</string>
</dict>
</plist>
`

func darwinPlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist"), nil
}

func installDarwin(bin string, port int) error {
	plistPath, err := darwinPlistPath()
	if err != nil {
		return err
	}
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, "Library", "Logs", "oriel.log")

	// The agent runs with a minimal PATH; colima/docker live in Homebrew dirs,
	// so include them (plus the binary's own dir) or lifecycle calls would fail.
	pathEnv := filepath.Dir(bin) + ":/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

	// The plist is XML; a path with &, <, or > would otherwise produce an invalid
	// document that `launchctl bootstrap` rejects.
	if err := render(plistPath, darwinPlist, map[string]any{
		"Label": label, "Bin": xmlEscape(bin), "Port": port, "Path": xmlEscape(pathEnv), "Log": xmlEscape(logPath),
	}); err != nil {
		return err
	}

	domain := fmt.Sprintf("gui/%d", os.Getuid())
	_ = run("launchctl", "bootout", domain+"/"+label) // ignore: may not be loaded
	if err := run("launchctl", "bootstrap", domain, plistPath); err != nil {
		// Fall back to the legacy verb for older macOS.
		if err2 := run("launchctl", "load", "-w", plistPath); err2 != nil {
			return fmt.Errorf("launchctl bootstrap failed: %w", err)
		}
	}

	fmt.Printf("✓ installed LaunchAgent: %s\n", plistPath)
	fmt.Printf("✓ Oriel is running at http://127.0.0.1:%d and will start on login\n", port)
	fmt.Printf("  logs: %s\n", logPath)
	return nil
}

func uninstallDarwin() error {
	plistPath, err := darwinPlistPath()
	if err != nil {
		return err
	}
	domain := fmt.Sprintf("gui/%d/%s", os.Getuid(), label)
	_ = run("launchctl", "bootout", domain) // ignore: may not be loaded
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	fmt.Println("✓ Oriel service removed")
	return nil
}

func statusDarwin() error {
	out, err := exec.Command("launchctl", "print", fmt.Sprintf("gui/%d/%s", os.Getuid(), label)).CombinedOutput()
	if err != nil {
		fmt.Println("○ Oriel service is not installed")
		return nil
	}
	state := "loaded"
	if strings.Contains(string(out), "state = running") {
		state = "running"
	}
	fmt.Printf("● Oriel service is installed (%s)\n", state)
	return nil
}

// ---- Linux (systemd: per-user unit, or system unit with --system / as root) ----

const linuxUnit = `[Unit]
Description=Oriel
After=network.target{{if .System}} docker.service
Wants=docker.service{{end}}

[Service]
ExecStart={{.Bin}} --no-open --port {{.Port}}
Restart=on-failure
RestartSec=3

[Install]
WantedBy={{if .System}}multi-user.target{{else}}default.target{{end}}
`

const systemUnitPath = "/etc/systemd/system/oriel.service"

func linuxUserUnitPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "systemd", "user", "oriel.service"), nil
}

// useSystem picks a system unit when asked, or implicitly when running as root
// (root has no user session bus, so a user unit can't work).
func useSystem(system bool) bool { return system || os.Geteuid() == 0 }

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// manageUnit picks the unit to act on for status/uninstall. An explicit --system
// wins; otherwise it prefers whichever scope's unit file actually exists, so a
// root/sudo invocation doesn't silently target a nonexistent system unit while
// the user unit is the one really installed. Falls back to the euid default.
func manageUnit(system bool) (sys bool, unitPath string) {
	if system {
		return true, systemUnitPath
	}
	userPath, _ := linuxUserUnitPath()
	sysExists := fileExists(systemUnitPath)
	userExists := userPath != "" && fileExists(userPath)
	switch {
	case userExists && !sysExists:
		return false, userPath
	case sysExists && !userExists:
		return true, systemUnitPath
	case useSystem(system):
		return true, systemUnitPath
	default:
		return false, userPath
	}
}

// xmlEscape escapes the five XML metacharacters for safe interpolation into the
// launchd plist template.
func xmlEscape(s string) string { return xmlReplacer.Replace(s) }

var xmlReplacer = strings.NewReplacer(
	"&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;",
)

// sctl prefixes systemctl with --user for per-user units.
func sctl(system bool, args ...string) []string {
	if system {
		return args
	}
	return append([]string{"--user"}, args...)
}

const userBusHint = `
Oriel installs a systemd *user* service by default, but this session has no user
bus, common over SSH/sudo, and always the case for root.

  • Headless or root-only box → install a system service instead (starts on boot):
        oriel service install --system

  • Or give yourself a user session and retry:
        sudo loginctl enable-linger "$USER"
        export XDG_RUNTIME_DIR="/run/user/$(id -u)"
        oriel service install`

func isUserBusError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "user scope bus") ||
		strings.Contains(s, "XDG_RUNTIME_DIR") ||
		strings.Contains(s, "DBUS_SESSION_BUS_ADDRESS")
}

func installLinux(bin string, port int, system bool) error {
	sys := useSystem(system)
	unitPath := systemUnitPath
	if !sys {
		p, err := linuxUserUnitPath()
		if err != nil {
			return err
		}
		unitPath = p
	}
	if err := render(unitPath, linuxUnit, map[string]any{"Bin": bin, "Port": port, "System": sys}); err != nil {
		return err
	}

	_ = run("systemctl", sctl(sys, "daemon-reload")...)
	// enable for start-on-boot/login; restart (rather than `enable --now`, which
	// won't touch an already-running unit) so re-running the installer to upgrade
	// actually swaps the live service onto the freshly-downloaded binary.
	if err := run("systemctl", sctl(sys, "enable", "oriel.service")...); err != nil {
		if !sys && isUserBusError(err) {
			fmt.Fprintln(os.Stderr, userBusHint)
		}
		return err
	}
	if err := run("systemctl", sctl(sys, "restart", "oriel.service")...); err != nil {
		return err
	}

	kind, sub, start := "user", "--user ", "login"
	if sys {
		kind, sub, start = "system", "", "boot"
	}
	fmt.Printf("✓ installed systemd %s service: %s\n", kind, unitPath)
	fmt.Printf("✓ Oriel is running at http://127.0.0.1:%d and will start on %s\n", port, start)
	fmt.Printf("  logs: journalctl %s-u oriel -f\n", sub)
	return nil
}

func uninstallLinux(system bool) error {
	sys, unitPath := manageUnit(system)
	existed := fileExists(unitPath)
	_ = run("systemctl", sctl(sys, "disable", "--now", "oriel.service")...)
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	_ = run("systemctl", sctl(sys, "daemon-reload")...)
	if !existed {
		fmt.Println("○ No Oriel service was installed")
		return nil
	}
	fmt.Println("✓ Oriel service removed")
	return nil
}

func statusLinux(system bool) error {
	sys, unitPath := manageUnit(system)
	if !fileExists(unitPath) {
		fmt.Println("○ Oriel service is not installed")
		return nil
	}
	out, _ := exec.Command("systemctl", sctl(sys, "is-active", "oriel.service")...).CombinedOutput()
	switch strings.TrimSpace(string(out)) {
	case "active":
		fmt.Println("● Oriel service is installed (running)")
	case "inactive", "failed":
		fmt.Printf("● Oriel service is installed (%s)\n", strings.TrimSpace(string(out)))
	default:
		fmt.Println("○ Oriel service is not installed")
	}
	return nil
}
