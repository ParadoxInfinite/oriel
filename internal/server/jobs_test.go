package server

import (
	"context"
	"testing"
	"time"
)

func jobDone(j *Job) bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.done
}

func waitDone(t *testing.T, j *Job) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if jobDone(j) {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("job did not finish in time")
}

// A subscriber that attaches mid-flight sees a snapshot of progress so far, then
// live lines, then the channel closes with a successful final state.
func TestJobSnapshotThenLiveThenDone(t *testing.T) {
	m := newJobManager()
	started := make(chan struct{})
	release := make(chan struct{})
	job := m.start("test", "Test", func(ctx context.Context, rep Reporter) error {
		rep.Line("one")
		close(started)
		<-release
		rep.Line("two")
		return nil
	})

	<-started // "one" is emitted; job is blocked before "two"
	snap, ch, unsub := job.subscribe()
	defer unsub()
	if snap.done {
		t.Fatal("job should not be done yet")
	}
	if len(snap.lines) != 1 || snap.lines[0] != "one" {
		t.Fatalf("snapshot = %v, want [one]", snap.lines)
	}

	close(release)
	if ev := <-ch; ev.line != "two" {
		t.Fatalf("live event = %q, want line two", ev.line)
	}
	if _, more := <-ch; more {
		t.Fatal("channel should close when the job finishes")
	}
	if ok, errMsg := job.finalState(); !ok || errMsg != "" {
		t.Fatalf("finalState ok=%v err=%q, want true/empty", ok, errMsg)
	}
}

// Subscribing after a job has already finished replays the full progress and
// reports the final state with no live channel.
func TestJobLateSubscribeReplays(t *testing.T) {
	m := newJobManager()
	job := m.start("test", "Test", func(ctx context.Context, rep Reporter) error {
		rep.Line("a")
		rep.Line("b")
		return nil
	})
	waitDone(t, job)

	snap, ch, unsub := job.subscribe()
	defer unsub()
	if !snap.done || ch != nil {
		t.Fatalf("late subscribe: done=%v ch=%v, want true/nil", snap.done, ch != nil)
	}
	if !snap.ok || snap.errMsg != "" {
		t.Fatalf("late subscribe state ok=%v err=%q", snap.ok, snap.errMsg)
	}
	if len(snap.lines) != 2 || snap.lines[0] != "a" || snap.lines[1] != "b" {
		t.Fatalf("snapshot = %v, want [a b]", snap.lines)
	}
}

// Progress updates arrive as progress events (not log lines) and the latest
// counter is captured in a late subscriber's snapshot.
func TestJobProgress(t *testing.T) {
	m := newJobManager()
	started := make(chan struct{})
	release := make(chan struct{})
	job := m.start("test", "Test", func(ctx context.Context, rep Reporter) error {
		rep.Progress(3, 10)
		close(started)
		<-release
		rep.Progress(7, 10)
		return nil
	})

	<-started
	snap, ch, unsub := job.subscribe()
	defer unsub()
	if snap.cur != 3 || snap.total != 10 {
		t.Fatalf("snapshot progress = %d/%d, want 3/10", snap.cur, snap.total)
	}
	close(release)
	if ev := <-ch; ev.kind != "progress" || ev.cur != 7 || ev.tot != 10 {
		t.Fatalf("live event = %+v, want progress 7/10", ev)
	}
}

// Cancelling a running job stops it and records a non-ok "cancelled" state.
func TestJobCancel(t *testing.T) {
	m := newJobManager()
	job := m.start("test", "Test", func(ctx context.Context, rep Reporter) error {
		<-ctx.Done()
		return ctx.Err()
	})
	job.cancel()
	waitDone(t, job)

	if ok, errMsg := job.finalState(); ok || errMsg != "cancelled" {
		t.Fatalf("after cancel ok=%v err=%q, want false/cancelled", ok, errMsg)
	}
}

// active() lists only running jobs, and starting a new job reaps finished ones.
func TestJobActiveAndReap(t *testing.T) {
	m := newJobManager()
	finished := m.start("k", "t", func(ctx context.Context, rep Reporter) error { return nil })
	waitDone(t, finished)

	if got := len(m.active()); got != 0 {
		t.Fatalf("active() = %d, want 0 (finished job excluded)", got)
	}

	release := make(chan struct{})
	running := m.start("k", "t", func(ctx context.Context, rep Reporter) error {
		<-release
		return nil
	})
	defer close(release)

	act := m.active()
	if len(act) != 1 || act[0].ID != running.ID {
		t.Fatalf("active() = %v, want one entry for %s", act, running.ID)
	}
	if m.get(finished.ID) != nil {
		t.Fatal("finished job should have been reaped when the next job started")
	}
}
