package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolve(t *testing.T) {
	// `name:` wins, services counted, ownName true.
	dir := t.TempDir()
	f := filepath.Join(dir, "compose.yaml")
	write(t, f, "name: My-Stack\nservices:\n  web: {image: nginx}\n  db: {image: postgres}\n")
	if name, own, svc := Resolve(f, dir); name != "my-stack" || !own || svc != 2 {
		t.Errorf("Resolve(name) = %q,%v,%d; want my-stack,true,2", name, own, svc)
	}

	// No `name:` → normalized directory basename, ownName false.
	dir2 := filepath.Join(t.TempDir(), "My App_1")
	f2 := filepath.Join(dir2, "docker-compose.yml")
	write(t, f2, "services:\n  solo: {image: alpine}\n")
	if name, own, svc := Resolve(f2, dir2); name != "myapp_1" || own || svc != 1 {
		t.Errorf("Resolve(dir) = %q,%v,%d; want myapp_1,false,1", name, own, svc)
	}

	// Unparseable → services -1, never panics.
	dir3 := filepath.Join(t.TempDir(), "broken")
	f3 := filepath.Join(dir3, "compose.yml")
	write(t, f3, "- not\n- a\n- mapping\n")
	if _, _, svc := Resolve(f3, dir3); svc != -1 {
		t.Errorf("Resolve(broken) services = %d; want -1", svc)
	}
}

func TestScan(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "projA", "compose.yaml"), "name: alpha\nservices:\n  x: {image: alpine}\n")
	write(t, filepath.Join(root, "nested", "deep", "projB", "docker-compose.yml"), "services:\n  y: {image: alpine}\n")
	write(t, filepath.Join(root, "node_modules", "skip", "compose.yml"), "services:\n  z: {image: alpine}\n")

	names := func(res ScanResult) map[string]bool {
		m := map[string]bool{}
		for _, s := range res.Stacks {
			m[s.Name] = true
		}
		return m
	}

	// Traverse: finds alpha + projb, skips node_modules.
	res := Scan(Config{Roots: []Root{{ID: "r1", Path: root, Traverse: true, Enabled: true}}})
	got := names(res)
	if !got["alpha"] || !got["projb"] || got["skip"] {
		t.Errorf("traverse names = %v; want alpha+projb, no skip", got)
	}
	if res.Roots[0].Found != 2 {
		t.Errorf("found = %d; want 2", res.Roots[0].Found)
	}

	// Non-traverse on a dir with no compose file → nothing.
	if r := Scan(Config{Roots: []Root{{ID: "r1", Path: root, Traverse: false, Enabled: true}}}); len(r.Stacks) != 0 {
		t.Errorf("non-traverse root = %d; want 0", len(r.Stacks))
	}
	// Non-traverse on the project dir itself → one.
	if r := Scan(Config{Roots: []Root{{ID: "r1", Path: filepath.Join(root, "projA"), Traverse: false, Enabled: true}}}); len(r.Stacks) != 1 || r.Stacks[0].Name != "alpha" {
		t.Errorf("non-traverse projA = %v; want [alpha]", r.Stacks)
	}
	// Disabled root → skipped.
	if r := Scan(Config{Roots: []Root{{ID: "r1", Path: root, Traverse: true, Enabled: false}}}); len(r.Stacks) != 0 {
		t.Errorf("disabled root = %d; want 0", len(r.Stacks))
	}
	// Missing path → error reported, no panic.
	if r := Scan(Config{Roots: []Root{{ID: "r1", Path: filepath.Join(root, "nope"), Traverse: true, Enabled: true}}}); r.Roots[0].Error == "" {
		t.Error("missing path should report an error")
	}
	// Overlapping roots → deduped by file path.
	res6 := Scan(Config{Roots: []Root{
		{ID: "r1", Path: root, Traverse: true, Enabled: true},
		{ID: "r2", Path: filepath.Join(root, "projA"), Traverse: false, Enabled: true},
	}})
	count := 0
	for _, s := range res6.Stacks {
		if s.Name == "alpha" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("alpha appeared %d times across overlapping roots; want 1", count)
	}
	// Aliases applied.
	res7 := Scan(Config{Roots: []Root{{ID: "r1", Path: filepath.Join(root, "projA"), Traverse: false, Enabled: true}}, Aliases: map[string]string{"alpha": "Alpha!"}})
	if res7.Stacks[0].Alias != "Alpha!" {
		t.Errorf("alias = %q; want Alpha!", res7.Stacks[0].Alias)
	}
}

func TestFilterAllows(t *testing.T) {
	d := Discovered{Name: "web-api", Alias: "My API", Dir: "/home/me/lab/web-api"}
	cases := []struct {
		mode string
		pats []string
		want bool
	}{
		{"off", nil, true},
		{"allow", []string{"web-*"}, true},
		{"allow", []string{"db-*"}, false},
		{"deny", []string{"web-*"}, false},
		{"deny", []string{"db-*"}, true},
		{"deny", []string{"my api"}, false},          // alias match (case-insensitive)
		{"deny", []string{"/home/me/lab/**"}, false}, // path glob
		{"deny", []string{"/home/me/other/**"}, true},
	}
	for i, c := range cases {
		if got := (Filter{Mode: c.mode, Patterns: c.pats}).Allows(d); got != c.want {
			t.Errorf("case %d (mode=%s pats=%v): got %v want %v", i, c.mode, c.pats, got, c.want)
		}
	}
}

func TestListDirs(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"alpha", "beta", ".hidden"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write(t, filepath.Join(root, "file.txt"), "x")

	// Lists directories (not files), hides dotdirs when no prefix typed.
	if _, dirs, err := ListDirs(root + string(os.PathSeparator)); err != nil || len(dirs) != 2 {
		t.Errorf("ListDirs(root) = %v, err %v; want 2 dirs", dirs, err)
	}
	// Prefix filter.
	if _, dirs, _ := ListDirs(filepath.Join(root, "al")); len(dirs) != 1 {
		t.Errorf("ListDirs(prefix 'al') = %v; want 1", dirs)
	}
}
