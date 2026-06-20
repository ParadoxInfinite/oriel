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
	job := m.start("test", "Test", func(ctx context.Context, emit func(string)) error {
		emit("one")
		close(started)
		<-release
		emit("two")
		return nil
	})

	<-started // "one" is emitted; job is blocked before "two"
	snapshot, ch, done, _, _, unsub := job.subscribe()
	defer unsub()
	if done {
		t.Fatal("job should not be done yet")
	}
	if len(snapshot) != 1 || snapshot[0] != "one" {
		t.Fatalf("snapshot = %v, want [one]", snapshot)
	}

	close(release)
	if got := <-ch; got != "two" {
		t.Fatalf("live line = %q, want two", got)
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
	job := m.start("test", "Test", func(ctx context.Context, emit func(string)) error {
		emit("a")
		emit("b")
		return nil
	})
	waitDone(t, job)

	snapshot, ch, done, ok, errMsg, unsub := job.subscribe()
	defer unsub()
	if !done || ch != nil {
		t.Fatalf("late subscribe: done=%v ch=%v, want true/nil", done, ch != nil)
	}
	if !ok || errMsg != "" {
		t.Fatalf("late subscribe state ok=%v err=%q", ok, errMsg)
	}
	if len(snapshot) != 2 || snapshot[0] != "a" || snapshot[1] != "b" {
		t.Fatalf("snapshot = %v, want [a b]", snapshot)
	}
}

// Cancelling a running job stops it and records a non-ok "cancelled" state.
func TestJobCancel(t *testing.T) {
	m := newJobManager()
	job := m.start("test", "Test", func(ctx context.Context, emit func(string)) error {
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
	finished := m.start("k", "t", func(ctx context.Context, emit func(string)) error { return nil })
	waitDone(t, finished)

	if got := len(m.active()); got != 0 {
		t.Fatalf("active() = %d, want 0 (finished job excluded)", got)
	}

	release := make(chan struct{})
	running := m.start("k", "t", func(ctx context.Context, emit func(string)) error {
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
