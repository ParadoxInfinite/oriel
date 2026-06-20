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

	if err := render(plistPath, darwinPlist, map[string]any{
		"Label": label, "Bin": bin, "Port": port, "Path": pathEnv, "Log": logPath,
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

// ---- Linux (systemd user service) ----

const linuxUnit = `[Unit]
Description=Oriel
After=network.target

[Service]
ExecStart={{.Bin}} --no-open --port {{.Port}}
Restart=on-failure
RestartSec=3

[Install]
WantedBy=default.target
`

func linuxUnitPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "systemd", "user", "oriel.service"), nil
}

func installLinux(bin string, port int) error {
	unitPath, err := linuxUnitPath()
	if err != nil {
		return err
	}
	if err := render(unitPath, linuxUnit, map[string]any{"Bin": bin, "Port": port}); err != nil {
		return err
	}
	_ = run("systemctl", "--user", "daemon-reload")
	if err := run("systemctl", "--user", "enable", "--now", "oriel.service"); err != nil {
		return err
	}
	fmt.Printf("✓ installed systemd user service: %s\n", unitPath)
	fmt.Printf("✓ Oriel is running at http://127.0.0.1:%d and will start on login\n", port)
	fmt.Println("  logs: journalctl --user -u oriel -f")
	return nil
}

func uninstallLinux() error {
	unitPath, err := linuxUnitPath()
	if err != nil {
		return err
	}
	_ = run("systemctl", "--user", "disable", "--now", "oriel.service")
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	_ = run("systemctl", "--user", "daemon-reload")
	fmt.Println("✓ Oriel service removed")
	return nil
}

func statusLinux() error {
	out, _ := exec.Command("systemctl", "--user", "is-active", "oriel.service").CombinedOutput()
	state := strings.TrimSpace(string(out))
	if state == "active" {
		fmt.Println("● Oriel service is installed (running)")
	} else if state == "inactive" || state == "failed" {
		fmt.Printf("● Oriel service is installed (%s)\n", state)
	} else {
		fmt.Println("○ Oriel service is not installed")
	}
	return nil
}
