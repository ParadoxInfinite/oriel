package server

import (
	"context"
	"fmt"
	"sync"
)

// A Job is a long-running operation (prune, …) that runs on a background context
// decoupled from any request, so it survives client refresh/disconnect. Progress
// is buffered for replay and broadcast live to attached SSE subscribers, and the
// job can be cancelled.
type Job struct {
	ID    string
	Kind  string
	Title string

	mu     sync.Mutex
	lines  []string
	done   bool
	ok     bool
	errMsg string
	subs   map[chan string]struct{}
	cancel context.CancelFunc
}

// jobView is the JSON shape for listing jobs (the Job fields are unexported).
type jobView struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func (j *Job) emit(line string) {
	j.mu.Lock()
	j.lines = append(j.lines, line)
	for ch := range j.subs {
		select {
		case ch <- line:
		default: // slow subscriber; replay/snapshot covers it on reconnect
		}
	}
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

// subscribe atomically snapshots progress so far and registers for future lines.
// If already finished, ch is nil and the final state is returned for replay.
func (j *Job) subscribe() (snapshot []string, ch chan string, done, ok bool, errMsg string, unsub func()) {
	j.mu.Lock()
	defer j.mu.Unlock()
	snapshot = append([]string(nil), j.lines...)
	if j.done {
		return snapshot, nil, true, j.ok, j.errMsg, func() {}
	}
	ch = make(chan string, 256)
	if j.subs == nil {
		j.subs = map[chan string]struct{}{}
	}
	j.subs[ch] = struct{}{}
	unsub = func() {
		j.mu.Lock()
		if _, ok := j.subs[ch]; ok {
			delete(j.subs, ch)
			close(ch)
		}
		j.mu.Unlock()
	}
	return snapshot, ch, false, false, "", unsub
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
func (m *jobManager) start(kind, title string, fn func(ctx context.Context, emit func(string)) error) *Job {
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
		err := fn(ctx, j.emit)
		switch {
		case err == nil:
			j.finish(true, "")
		case ctx.Err() != nil:
			j.finish(false, "cancelled")
		default:
			j.finish(false, err.Error())
		}
	}()
	return j
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
			out = append(out, jobView{ID: j.ID, Kind: j.Kind, Title: j.Title, Done: j.done})
		}
		j.mu.Unlock()
	}
	return out
}
