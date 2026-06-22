package server

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// finishedJobTTL is how long a completed job lingers so a reconnecting client can
// still read its final snapshot before it's reaped.
const finishedJobTTL = 30 * time.Second

// A Job is a long-running operation (prune, …) that runs on a background context
// decoupled from any request, so it survives client refresh/disconnect. Progress
// is buffered for replay and broadcast live to attached SSE subscribers, and the
// job can be cancelled. Progress comes in two forms: log lines (a few step
// messages) and a numeric counter (cur/total) that drives a progress bar.
type Job struct {
	ID    string
	Kind  string
	Title string

	mu     sync.Mutex
	lines  []string
	cur    int
	total  int
	done   bool
	ok     bool
	errMsg string
	subs   map[chan jobEvent]struct{}
	cancel context.CancelFunc
}

// jobEvent is one live update: a log line or a progress counter.
type jobEvent struct {
	kind string // "line" | "progress"
	line string
	cur  int
	tot  int
}

// jobSnapshot is the state replayed to a (re)connecting subscriber.
type jobSnapshot struct {
	lines  []string
	cur    int
	total  int
	done   bool
	ok     bool
	errMsg string
}

// Reporter is what a job's work function uses to report progress. Line adds a log
// message; Progress sets the cur/total counter (drives the bar).
type Reporter struct{ job *Job }

func (r Reporter) Line(s string)           { r.job.emit(s) }
func (r Reporter) Progress(cur, total int) { r.job.setProgress(cur, total) }

// jobView is the JSON shape for listing jobs (the Job fields are unexported).
type jobView struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
}

// broadcast sends ev to every subscriber without blocking; the caller holds mu.
func (j *Job) broadcast(ev jobEvent) {
	for ch := range j.subs {
		select {
		case ch <- ev:
		default: // slow subscriber; replay/snapshot covers it on reconnect
		}
	}
}

func (j *Job) emit(line string) {
	j.mu.Lock()
	j.lines = append(j.lines, line)
	j.broadcast(jobEvent{kind: "line", line: line})
	j.mu.Unlock()
}

func (j *Job) setProgress(cur, total int) {
	j.mu.Lock()
	j.cur, j.total = cur, total
	j.broadcast(jobEvent{kind: "progress", cur: cur, tot: total})
	j.mu.Unlock()
}

func (j *Job) finish(ok bool, errMsg string) {
	j.mu.Lock()
	j.done, j.ok, j.errMsg = true, ok, errMsg
	for ch := range j.subs {
		close(ch)
	}
	j.subs = nil
	j.mu.Unlock()
}

// subscribe atomically snapshots progress so far and registers for future events.
// If already finished, ch is nil and the snapshot carries the final state.
func (j *Job) subscribe() (jobSnapshot, chan jobEvent, func()) {
	j.mu.Lock()
	defer j.mu.Unlock()
	snap := jobSnapshot{
		lines:  append([]string(nil), j.lines...),
		cur:    j.cur,
		total:  j.total,
		done:   j.done,
		ok:     j.ok,
		errMsg: j.errMsg,
	}
	if j.done {
		return snap, nil, func() {}
	}
	ch := make(chan jobEvent, 256)
	if j.subs == nil {
		j.subs = map[chan jobEvent]struct{}{}
	}
	j.subs[ch] = struct{}{}
	unsub := func() {
		j.mu.Lock()
		if _, ok := j.subs[ch]; ok {
			delete(j.subs, ch)
			close(ch)
		}
		j.mu.Unlock()
	}
	return snap, ch, unsub
}

func (j *Job) finalState() (ok bool, errMsg string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.ok, j.errMsg
}

// jobManager tracks running jobs.
type jobManager struct {
	mu   sync.Mutex
	jobs map[string]*Job
	seq  int
}

func newJobManager() *jobManager { return &jobManager{jobs: map[string]*Job{}} }

// start spawns fn on a cancellable background context and returns the live Job.
func (m *jobManager) start(kind, title string, fn func(ctx context.Context, rep Reporter) error) *Job {
	m.mu.Lock()
	// Reap finished jobs; a still-running one is what a refreshed client re-attaches to.
	for id, j := range m.jobs {
		j.mu.Lock()
		fin := j.done
		j.mu.Unlock()
		if fin {
			delete(m.jobs, id)
		}
	}
	m.seq++
	id := fmt.Sprintf("op%d", m.seq)
	ctx, cancel := context.WithCancel(context.Background())
	j := &Job{ID: id, Kind: kind, Title: title, cancel: cancel}
	m.jobs[id] = j
	m.mu.Unlock()

	go func() {
		err := fn(ctx, Reporter{job: j})
		switch {
		case err == nil:
			j.finish(true, "")
		case ctx.Err() != nil:
			j.finish(false, "cancelled")
		default:
			j.finish(false, err.Error())
		}
		// Reap this job after a grace window even if no other job starts, so the
		// map can't grow unbounded from completed work.
		time.AfterFunc(finishedJobTTL, func() { m.remove(id) })
	}()
	return j
}

// remove drops a job from the map (used by the post-finish reaper).
func (m *jobManager) remove(id string) {
	m.mu.Lock()
	delete(m.jobs, id)
	m.mu.Unlock()
}

func (m *jobManager) get(id string) *Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.jobs[id]
}

// active returns jobs that haven't finished (what a client resumes after refresh).
func (m *jobManager) active() []jobView {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []jobView{}
	for _, j := range m.jobs {
		j.mu.Lock()
		if !j.done {
			out = append(out, jobView{ID: j.ID, Kind: j.Kind, Title: j.Title})
		}
		j.mu.Unlock()
	}
	return out
}
