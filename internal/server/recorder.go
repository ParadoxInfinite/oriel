package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/ParadoxInfinite/oriel/internal/docker"
	"github.com/ParadoxInfinite/oriel/internal/userdata"
)

const (
	recordInterval = time.Second
	historyCap     = 1800            // ~30 minutes at 1s resolution
	flushInterval  = 5 * time.Minute // periodically persist the buffer to disk

	// A startup time-jump beyond this is treated as Oriel having been
	// offline (logged as an outage), not a normal restart.
	offlineThreshold     = 30 * time.Second
	defaultRetentionDays = 30

	// Same-kind outages separated by less uptime than this are merged into one,
	// so a flapping VM reads as a single sustained outage rather than slivers.
	mergeGap = 60 * time.Second
)

// Outage is one recorded downtime, retained far longer than the 30-min pulse
// buffer. Kind is "down" (colima unreachable while we watched) or "offline"
// (Oriel itself was not running, inferred from a gap on restart).
type Outage struct {
	Kind  string `json:"kind"`
	Start int64  `json:"start"` // unix ms
	End   int64  `json:"end"`   // unix ms
}

// HistoryPoint is one aggregate sample of total CPU% and memory at a moment.
// Down marks a tick where colima/docker was unreachable while the recorder was
// running — distinct from Oriel itself being offline, which records nothing
// at all and so leaves a time gap between points. (omitempty keeps the common
// "up" case compact and lets older persisted files load as up.)
type HistoryPoint struct {
	T    int64   `json:"t"` // unix milliseconds
	CPU  float64 `json:"cpu"`
	Mem  int64   `json:"mem"`
	Down bool    `json:"down,omitempty"`
}

// recorder is a single always-on sampler. It feeds the live /api/stats stream
// (latest per-container snapshot) and the /api/history ring buffer (aggregate
// over the last ~30 min), so history survives browser refreshes and accrues
// even with no tab open. One sampler means correct, un-doubled CPU deltas.
type recorder struct {
	sampler   *docker.Sampler
	mu        sync.Mutex
	latest    []docker.Stat
	history   []HistoryPoint
	path      string // where the history buffer is persisted
	outages   []Outage
	downSince int64 // unix ms an in-progress colima outage began, 0 if up
	outPath   string
	retention time.Duration
}

func newRecorder(dc *docker.Client) *recorder {
	r := &recorder{
		sampler:   docker.NewSampler(dc),
		path:      dataPath("history.json"),
		outPath:   dataPath("outages.json"),
		retention: retentionWindow(),
	}
	r.load()
	r.loadOutages()
	r.detectStartupOffline()
	return r
}

// dataPath returns a stable per-user file. Thin alias over userdata.Path, kept
// for the many existing call sites in this package.
func dataPath(name string) string { return userdata.Path(name) }

// retentionWindow is how long outages are kept, configurable via
// ORIEL_OUTAGE_RETENTION_DAYS (default 30 days).
func retentionWindow() time.Duration {
	days := defaultRetentionDays
	if v := os.Getenv("ORIEL_OUTAGE_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			days = n
		}
	}
	return time.Duration(days) * 24 * time.Hour
}

// detectStartupOffline logs an Oriel outage when the buffer resumes after a
// gap longer than offlineThreshold — i.e. we weren't running for a while.
func (r *recorder) detectStartupOffline() {
	if len(r.history) == 0 {
		return
	}
	last := r.history[len(r.history)-1].T
	now := time.Now().UnixMilli()
	if now-last > offlineThreshold.Milliseconds() {
		r.appendOutageLocked(Outage{Kind: "offline", Start: last, End: now})
		r.pruneOutagesLocked()
		r.flushOutages()
	}
}

func (r *recorder) run(ctx context.Context) {
	t := time.NewTicker(recordInterval)
	defer t.Stop()
	flush := time.NewTicker(flushInterval)
	defer flush.Stop()
	for {
		select {
		case <-ctx.Done():
			// persist on graceful shutdown so a restart resumes
			r.closeOpenOutage()
			r.flush()
			r.flushOutages()
			return
		case <-t.C:
			r.tick(ctx)
		case <-flush.C:
			r.flush()
			r.flushOutages()
		}
	}
}

// load reads a previously persisted buffer, capped to the retention window.
func (r *recorder) load() {
	b, err := os.ReadFile(r.path)
	if err != nil {
		return
	}
	var hist []HistoryPoint
	if json.Unmarshal(b, &hist) != nil {
		return
	}
	if len(hist) > historyCap {
		hist = hist[len(hist)-historyCap:]
	}
	r.history = hist
}

// flush writes the current buffer atomically (temp + rename) so a crash mid-write
// never corrupts the file. Best-effort: persistence is a nicety, not critical.
func (r *recorder) flush() {
	r.mu.Lock()
	if len(r.history) == 0 {
		r.mu.Unlock()
		return
	}
	hist := make([]HistoryPoint, len(r.history))
	copy(hist, r.history)
	r.mu.Unlock()

	b, err := json.Marshal(hist)
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(r.path), 0o755) != nil {
		return
	}
	tmp := r.path + ".tmp"
	if os.WriteFile(tmp, b, 0o644) != nil {
		return
	}
	_ = os.Rename(tmp, r.path)
}

func (r *recorder) tick(ctx context.Context) {
	stats, err := r.sampler.Sample(ctx)
	now := time.Now().UnixMilli()
	if err != nil {
		// colima/docker unreachable: record a down marker so the outage is
		// visible and distinguishable from Oriel being offline (no record).
		r.mu.Lock()
		r.latest = nil
		if r.downSince == 0 {
			r.downSince = now
		}
		r.appendLocked(HistoryPoint{T: now, Down: true})
		r.mu.Unlock()
		return
	}
	var cpu float64
	var mem int64
	for _, s := range stats {
		cpu += s.CPU
		mem += s.Mem
	}
	recovered := false
	r.mu.Lock()
	r.latest = stats
	if r.downSince != 0 { // colima just came back — close the outage
		r.appendOutageLocked(Outage{Kind: "down", Start: r.downSince, End: now})
		r.downSince = 0
		r.pruneOutagesLocked()
		recovered = true
	}
	r.appendLocked(HistoryPoint{T: now, CPU: cpu, Mem: mem})
	r.mu.Unlock()
	if recovered {
		r.flushOutages()
	}
}

// appendLocked adds a point and trims to the retention window. Caller holds r.mu.
func (r *recorder) appendLocked(p HistoryPoint) {
	r.history = append(r.history, p)
	if len(r.history) > historyCap {
		r.history = r.history[len(r.history)-historyCap:]
	}
}

// latestSnapshot returns the most recent per-container stats (copy-safe: the
// slice is replaced wholesale each tick, never mutated in place).
func (r *recorder) latestSnapshot() []docker.Stat {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.latest
}

// latestPoint returns the most recent aggregate sample, ok=false if none yet.
func (r *recorder) latestPoint() (HistoryPoint, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.history) == 0 {
		return HistoryPoint{}, false
	}
	return r.history[len(r.history)-1], true
}

func (r *recorder) historyCopy() []HistoryPoint {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]HistoryPoint, len(r.history))
	copy(out, r.history)
	return out
}

func (r *recorder) outagesCopy() []Outage {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Outage, len(r.outages))
	copy(out, r.outages)
	return out
}

// closeOpenOutage ends an in-progress colima outage at "now" — called on
// shutdown so a downtime spanning a restart is still recorded.
func (r *recorder) closeOpenOutage() {
	r.mu.Lock()
	if r.downSince != 0 {
		r.appendOutageLocked(Outage{Kind: "down", Start: r.downSince, End: time.Now().UnixMilli()})
		r.downSince = 0
	}
	r.mu.Unlock()
}

// appendOutageLocked adds an outage, merging it into the previous one when they
// are the same kind and separated by less than mergeGap of uptime. Caller holds r.mu.
func (r *recorder) appendOutageLocked(o Outage) {
	if n := len(r.outages); n > 0 {
		last := &r.outages[n-1]
		if last.Kind == o.Kind && o.Start-last.End <= mergeGap.Milliseconds() {
			if o.End > last.End {
				last.End = o.End
			}
			return
		}
	}
	r.outages = append(r.outages, o)
}

// pruneOutagesLocked drops outages older than the retention window. Caller holds r.mu.
func (r *recorder) pruneOutagesLocked() {
	cut := time.Now().UnixMilli() - r.retention.Milliseconds()
	kept := r.outages[:0]
	for _, o := range r.outages {
		if o.End == 0 || o.End >= cut {
			kept = append(kept, o)
		}
	}
	r.outages = kept
}

func (r *recorder) loadOutages() {
	b, err := os.ReadFile(r.outPath)
	if err != nil {
		return
	}
	var out []Outage
	if json.Unmarshal(b, &out) != nil {
		return
	}
	r.outages = out
	r.pruneOutagesLocked() // safe: load runs single-threaded at construction
}

func (r *recorder) flushOutages() {
	r.mu.Lock()
	out := make([]Outage, len(r.outages))
	copy(out, r.outages)
	r.mu.Unlock()
	if len(out) == 0 {
		return
	}
	b, err := json.Marshal(out)
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(r.outPath), 0o755) != nil {
		return
	}
	tmp := r.outPath + ".tmp"
	if os.WriteFile(tmp, b, 0o644) != nil {
		return
	}
	_ = os.Rename(tmp, r.outPath)
}
