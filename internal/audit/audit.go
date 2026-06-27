// Package audit records the tool calls an AI/MCP client makes, so the operator
// can always see what an assistant did to their containers. It is shared across
// processes: both the GUI server and the standalone `oriel mcp` process append to
// the same JSONL file (under the user data dir), and the server reads it for the
// UI. The operator's own UI clicks are NOT recorded, only non-interactive (agent)
// calls reach Record.
package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/ParadoxInfinite/oriel/internal/secrets"
	"github.com/ParadoxInfinite/oriel/internal/userdata"
)

// Entry is one recorded agent tool call.
type Entry struct {
	Time  string         `json:"time"` // RFC3339 UTC
	Tool  string         `json:"tool"`
	Args  map[string]any `json:"args,omitempty"`
	OK    bool           `json:"ok"`
	Error string         `json:"error,omitempty"`
}

// Bounds the log. Vars (not consts) so tests can shrink them.
var (
	maxBytes int64 = 1 << 20 // ~1 MB; the log is trimmed past this
	keepN          = 1000    // entries kept on trim, and the max Read returns
)

// mu serializes this process's reads/writes. Across processes the appends are
// O_APPEND (atomic for these small lines) and trimming is best-effort.
var mu sync.Mutex

func path() string { return userdata.Path("audit.jsonl") }

// Record appends one entry for an agent tool call. String args that look like
// secrets are masked, so a tool argument can't leak a credential into the log.
// Best-effort: a logging failure must never break the tool call, so errors are
// swallowed.
func Record(tool string, args map[string]any, err error) {
	e := Entry{
		Time: time.Now().UTC().Format(time.RFC3339),
		Tool: tool,
		Args: maskArgs(args),
		OK:   err == nil,
	}
	if err != nil {
		e.Error = err.Error()
	}
	line, jerr := json.Marshal(e)
	if jerr != nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	_ = os.MkdirAll(dirOf(path()), 0o755)
	trimLocked()
	f, ferr := os.OpenFile(path(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if ferr != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(line, '\n'))
}

// Read returns the most recent entries, newest first (at most keepN).
func Read(limit int) []Entry {
	if limit <= 0 || limit > keepN {
		limit = keepN
	}
	mu.Lock()
	defer mu.Unlock()
	all := readAllLocked()
	out := make([]Entry, 0, limit)
	for i := len(all) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, all[i])
	}
	return out
}

func maskArgs(args map[string]any) map[string]any {
	if len(args) == 0 {
		return nil
	}
	out := make(map[string]any, len(args))
	for k, v := range args {
		if s, ok := v.(string); ok && secrets.IsSensitive(k, s) {
			out[k] = secrets.MaskValue(s)
		} else {
			out[k] = v
		}
	}
	return out
}

// readAllLocked parses every entry in file order (oldest first). Caller holds mu.
func readAllLocked() []Entry {
	f, err := os.Open(path())
	if err != nil {
		return nil
	}
	defer f.Close()
	var all []Entry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		var e Entry
		if json.Unmarshal(sc.Bytes(), &e) == nil {
			all = append(all, e)
		}
	}
	return all
}

// trimLocked rewrites the log with only the most recent keepN entries once it
// grows past maxBytes. Caller holds mu. Best-effort across processes: a
// concurrent append during the rewrite could be lost, acceptable for an audit
// log, and rare given the size threshold.
func trimLocked() {
	fi, err := os.Stat(path())
	if err != nil || fi.Size() < maxBytes {
		return
	}
	all := readAllLocked()
	if len(all) > keepN {
		all = all[len(all)-keepN:]
	}
	tmp, terr := os.CreateTemp(dirOf(path()), "audit-*.jsonl.tmp")
	if terr != nil {
		return
	}
	w := bufio.NewWriter(tmp)
	for _, e := range all {
		b, _ := json.Marshal(e)
		w.Write(append(b, '\n'))
	}
	w.Flush()
	tmp.Close()
	_ = os.Rename(tmp.Name(), path())
}

func dirOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return "."
}
