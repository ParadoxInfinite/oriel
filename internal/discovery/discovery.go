// Package discovery finds Docker Compose projects on disk under user-configured
// roots, so the UI can offer "available but not yet deployed" stacks alongside
// the label-derived running ones. It is read-only: it parses just enough of each
// compose file (its name and service count) and never executes anything.
package discovery

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Root is one configured search location.
type Root struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	Traverse bool   `json:"traverse"`
	Enabled  bool   `json:"enabled"`
}

// Filter restricts which discovered stacks surface. Mode "off" shows all;
// "allow" shows only matches; "deny" hides matches. Running stacks are never
// affected — the filter is applied to discovery results only.
type Filter struct {
	Mode     string   `json:"mode"`
	Patterns []string `json:"patterns"`
}

// Config is the persisted discovery configuration.
type Config struct {
	Roots   []Root            `json:"roots"`
	Filter  Filter            `json:"filter"`
	Aliases map[string]string `json:"aliases"` // normalized project name → Oriel display name
}

// Discovered is one compose project found on disk.
type Discovered struct {
	Name     string `json:"name"`     // normalized project name (the identity)
	Alias    string `json:"alias"`    // Oriel display name, if set (display only)
	Dir      string `json:"dir"`      // project directory
	File     string `json:"file"`     // canonical compose file
	Services int    `json:"services"` // service count, -1 if unparseable
}

// RootResult reports per-root scan feedback for the Settings UI.
type RootResult struct {
	ID    string `json:"id"`
	Found int    `json:"found"`
	Error string `json:"error,omitempty"`
}

// ScanResult is the full output of a scan.
type ScanResult struct {
	Stacks []Discovered `json:"stacks"`
	Roots  []RootResult `json:"roots"`
}

// Compose file names, in the precedence Compose itself uses.
var composeFiles = []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"}

// Directories never worth walking into.
var skipDirs = map[string]bool{
	"node_modules": true, ".git": true, "vendor": true, "dist": true, "build": true,
	".next": true, ".nuxt": true, "target": true, ".venv": true, "venv": true,
	"__pycache__": true, ".idea": true, ".vscode": true, ".terraform": true,
	".cache": true, "coverage": true, ".svn": true, ".hg": true,
}

const maxDepth = 6 // levels below a traversed root

var (
	nonNameChars = regexp.MustCompile(`[^a-z0-9_-]`)
	leadingSep   = regexp.MustCompile(`^[_-]+`)
)

// normalizeName mirrors Compose's project-name normalization so a discovered
// stack's name matches the com.docker.compose.project label of a deployed one.
func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonNameChars.ReplaceAllString(s, "")
	return leadingSep.ReplaceAllString(s, "")
}

func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

func canonicalCompose(dir string) string {
	for _, n := range composeFiles {
		p := filepath.Join(dir, n)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

// Resolve parses a compose file for its project name and service count. If the
// file declares a top-level `name:`, that wins (ownName=true → don't pass -p on
// up); otherwise the name derives from the directory.
func Resolve(file, dir string) (name string, ownName bool, services int) {
	services = -1
	if data, err := os.ReadFile(file); err == nil {
		var doc struct {
			Name     string               `yaml:"name"`
			Services map[string]yaml.Node `yaml:"services"`
		}
		if yaml.Unmarshal(data, &doc) == nil {
			services = len(doc.Services)
			if doc.Name != "" {
				return normalizeName(doc.Name), true, services
			}
		}
	}
	return normalizeName(filepath.Base(dir)), false, services
}

func discover(dir string, seen map[string]bool) (Discovered, bool) {
	file := canonicalCompose(dir)
	if file == "" || seen[file] {
		return Discovered{}, false
	}
	seen[file] = true
	name, _, services := Resolve(file, dir)
	return Discovered{Name: name, Dir: dir, File: file, Services: services}, true
}

func scanRoot(root Root, seen map[string]bool) ([]Discovered, error) {
	path := expandHome(root.Path)
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("not a directory")
	}

	var out []Discovered
	if !root.Traverse {
		if d, ok := discover(path, seen); ok {
			out = append(out, d)
		}
		return out, nil
	}

	base := strings.Count(filepath.Clean(path), string(os.PathSeparator))
	_ = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable subtree — skip it, keep going
		}
		if !d.IsDir() { // symlinks report as non-dir, so we never follow them
			return nil
		}
		if p != path && skipDirs[d.Name()] {
			return filepath.SkipDir
		}
		if strings.Count(filepath.Clean(p), string(os.PathSeparator))-base > maxDepth {
			return filepath.SkipDir
		}
		if found, ok := discover(p, seen); ok {
			out = append(out, found)
		}
		return nil
	})
	return out, nil
}

// Scan walks all enabled roots, de-duplicates by compose-file path, applies
// display aliases, and reports per-root feedback. Filtering and deployed-stack
// exclusion happen in the caller (which has the live container list).
func Scan(cfg Config) ScanResult {
	seen := map[string]bool{}
	var res ScanResult
	for _, root := range cfg.Roots {
		rr := RootResult{ID: root.ID}
		if !root.Enabled {
			res.Roots = append(res.Roots, rr)
			continue
		}
		found, err := scanRoot(root, seen)
		if err != nil {
			rr.Error = err.Error()
		} else {
			rr.Found = len(found)
			res.Stacks = append(res.Stacks, found...)
		}
		res.Roots = append(res.Roots, rr)
	}
	for i := range res.Stacks {
		if a := cfg.Aliases[res.Stacks[i].Name]; a != "" {
			res.Stacks[i].Alias = a
		}
	}
	sort.Slice(res.Stacks, func(i, j int) bool {
		return res.Stacks[i].displayKey() < res.Stacks[j].displayKey()
	})
	return res
}

func (d Discovered) displayKey() string {
	if d.Alias != "" {
		return strings.ToLower(d.Alias)
	}
	return d.Name
}

// Allows reports whether a discovered stack passes the filter. Patterns match a
// project's name, its Oriel alias, or its directory path (path globs may use **).
func (f Filter) Allows(d Discovered) bool {
	if f.Mode != "allow" && f.Mode != "deny" {
		return true
	}
	matched := false
	for _, pat := range f.Patterns {
		if matchPattern(pat, d.Name, d.Alias, d.Dir) {
			matched = true
			break
		}
	}
	if f.Mode == "allow" {
		return matched
	}
	return !matched
}

func matchPattern(pat, name, alias, dir string) bool {
	pat = strings.TrimSpace(pat)
	if pat == "" {
		return false
	}
	if strings.ContainsAny(pat, "/~") { // a path pattern
		p, d := expandHome(pat), expandHome(dir)
		if i := strings.Index(p, "**"); i >= 0 {
			return strings.HasPrefix(d, strings.TrimRight(p[:i], "/"))
		}
		ok, _ := filepath.Match(p, d)
		return ok
	}
	if ok, _ := filepath.Match(pat, name); ok {
		return true
	}
	if alias != "" {
		if ok, _ := filepath.Match(strings.ToLower(pat), strings.ToLower(alias)); ok {
			return true
		}
	}
	return false
}

// ListDirs returns the immediate child directories of the typed path, for the
// Settings path typeahead (Radarr-style). It splits a partial input into a
// directory plus a name prefix and lists matching subdirectories.
func ListDirs(input string) (string, []string, error) {
	input = expandHome(strings.TrimSpace(input))
	if input == "" {
		if home, err := os.UserHomeDir(); err == nil {
			input = home + string(os.PathSeparator)
		} else {
			input = string(os.PathSeparator)
		}
	}
	dir, prefix := input, ""
	if !strings.HasSuffix(input, string(os.PathSeparator)) {
		dir, prefix = filepath.Dir(input), filepath.Base(input)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return dir, nil, err
	}
	var out []string
	lp := strings.ToLower(prefix)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		n := e.Name()
		if prefix == "" && strings.HasPrefix(n, ".") {
			continue // hide dotfiles until the user types a dot
		}
		if lp == "" || strings.HasPrefix(strings.ToLower(n), lp) {
			out = append(out, filepath.Join(dir, n))
		}
	}
	sort.Strings(out)
	if len(out) > 50 {
		out = out[:50]
	}
	return dir, out, nil
}
