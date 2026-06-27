package audit

import (
	"errors"
	"testing"
)

// sandbox points the user data dir at a temp dir via $HOME (os.UserConfigDir keys
// off it), so tests never touch the real audit log.
func sandbox(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func TestRecordAndRead(t *testing.T) {
	sandbox(t)
	Record("container.list", nil, nil)
	Record("container.stop", map[string]any{"id": "web"}, nil)
	Record("container.remove", map[string]any{"id": "db"}, errors.New("destructive action locked"))

	got := Read(10)
	if len(got) != 3 {
		t.Fatalf("Read = %d entries, want 3", len(got))
	}
	// Newest first.
	if got[0].Tool != "container.remove" || got[2].Tool != "container.list" {
		t.Errorf("order wrong: %q … %q", got[0].Tool, got[2].Tool)
	}
	if got[0].OK || got[0].Error == "" {
		t.Errorf("failed call should record OK=false + error, got OK=%v err=%q", got[0].OK, got[0].Error)
	}
	if got[1].Args["id"] != "web" {
		t.Errorf("args not recorded: %v", got[1].Args)
	}
}

func TestMaskArgs(t *testing.T) {
	sandbox(t)
	Record("x.y", map[string]any{"name": "web", "token": "sk-secret-value", "force": true}, nil)
	a := Read(1)[0].Args
	if a["name"] != "web" {
		t.Errorf("benign arg masked: %v", a["name"])
	}
	if a["token"] == "sk-secret-value" {
		t.Error("secret-shaped arg must be masked")
	}
	if a["force"] != true {
		t.Errorf("non-string arg changed: %v", a["force"])
	}
}

func TestTrim(t *testing.T) {
	sandbox(t)
	maxBytes, keepN = 200, 5
	t.Cleanup(func() { maxBytes, keepN = 1<<20, 1000 })
	for i := 0; i < 50; i++ {
		Record("container.list", map[string]any{"n": i}, nil)
	}
	got := Read(1000)
	if len(got) > keepN {
		t.Errorf("after trim, Read = %d, want <= %d", len(got), keepN)
	}
	if len(got) == 0 {
		t.Fatal("trim dropped everything")
	}
	// The most recent entry must survive the trim.
	if got[0].Args["n"] == nil {
		t.Errorf("newest entry missing after trim: %v", got[0])
	}
}
