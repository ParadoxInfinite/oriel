package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// renderUnit is a small helper that renders the linux systemd unit template to a
// temp file and returns its contents.
func renderUnit(t *testing.T, data map[string]any) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "oriel.service")
	if err := render(p, linuxUnit, data); err != nil {
		t.Fatalf("render: %v", err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(b)
}

func TestLinuxUnitEmbedsBasePath(t *testing.T) {
	s := renderUnit(t, map[string]any{"Bin": "/usr/local/bin/oriel", "Port": 4321, "System": true, "BasePath": "/oriel", "AllowedHosts": ""})
	if !strings.Contains(s, "Environment=ORIEL_BASE_PATH=/oriel\n") {
		t.Errorf("expected ORIEL_BASE_PATH environment line, got:\n%s", s)
	}
	// The env line must precede ExecStart so the process sees it.
	if i, j := strings.Index(s, "Environment=ORIEL_BASE_PATH"), strings.Index(s, "ExecStart="); i == -1 || j == -1 || i > j {
		t.Errorf("Environment must appear before ExecStart, got:\n%s", s)
	}
	if !strings.Contains(s, "ExecStart=/usr/local/bin/oriel --no-open --port 4321\n") {
		t.Errorf("unexpected ExecStart, got:\n%s", s)
	}
}

func TestLinuxUnitNoBasePathByDefault(t *testing.T) {
	s := renderUnit(t, map[string]any{"Bin": "/usr/local/bin/oriel", "Port": 4321, "System": false, "BasePath": "", "AllowedHosts": ""})
	if strings.Contains(s, "ORIEL_BASE_PATH") {
		t.Errorf("did not expect any ORIEL_BASE_PATH line when base path is empty, got:\n%s", s)
	}
	if strings.Contains(s, "ORIEL_ALLOWED_HOSTS") {
		t.Errorf("did not expect any ORIEL_ALLOWED_HOSTS line when allowed hosts is empty, got:\n%s", s)
	}
}

func TestDarwinPlistEnv(t *testing.T) {
	base := map[string]any{"Label": label, "Bin": "/usr/local/bin/oriel", "Port": 4321, "Path": "/usr/bin", "Log": "/tmp/oriel.log"}

	// With both set: keys present and the plist stays well-formed.
	with := map[string]any{}
	for k, v := range base {
		with[k] = v
	}
	with["BasePath"], with["AllowedHosts"] = "/oriel", "oriel.example.com"
	p := filepath.Join(t.TempDir(), "with.plist")
	if err := render(p, darwinPlist, with); err != nil {
		t.Fatalf("render: %v", err)
	}
	b, _ := os.ReadFile(p)
	s := string(b)
	for _, want := range []string{
		"<key>ORIEL_BASE_PATH</key><string>/oriel</string>",
		"<key>ORIEL_ALLOWED_HOSTS</key><string>oriel.example.com</string>",
		"</dict>\n</plist>",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("plist missing %q, got:\n%s", want, s)
		}
	}

	// With neither: no env keys leak in.
	none := map[string]any{}
	for k, v := range base {
		none[k] = v
	}
	none["BasePath"], none["AllowedHosts"] = "", ""
	p2 := filepath.Join(t.TempDir(), "none.plist")
	if err := render(p2, darwinPlist, none); err != nil {
		t.Fatalf("render: %v", err)
	}
	b2, _ := os.ReadFile(p2)
	if strings.Contains(string(b2), "ORIEL_BASE_PATH") || strings.Contains(string(b2), "ORIEL_ALLOWED_HOSTS") {
		t.Errorf("did not expect env keys when unset, got:\n%s", b2)
	}
}

func TestLinuxUnitEmbedsAllowedHosts(t *testing.T) {
	s := renderUnit(t, map[string]any{
		"Bin": "/usr/local/bin/oriel", "Port": 4321, "System": true,
		"BasePath": "/oriel", "AllowedHosts": "oriel.example.com,box.tailnet.ts.net",
	})
	if !strings.Contains(s, "Environment=ORIEL_ALLOWED_HOSTS=oriel.example.com,box.tailnet.ts.net\n") {
		t.Errorf("expected ORIEL_ALLOWED_HOSTS environment line, got:\n%s", s)
	}
	// Both env lines must precede ExecStart so the process inherits them.
	if i, j := strings.Index(s, "Environment=ORIEL_ALLOWED_HOSTS"), strings.Index(s, "ExecStart="); i == -1 || j == -1 || i > j {
		t.Errorf("Environment must appear before ExecStart, got:\n%s", s)
	}
}
