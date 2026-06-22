package tools

import (
	"context"
	"errors"
	"testing"
)

func gateRegistry(windowOpen func() bool) *Registry {
	r := NewRegistry(nil)
	r.SetDestructiveWindow(windowOpen)
	r.Register(&Tool{
		Name: "safe.read", Description: "read",
		Handler: func(context.Context, map[string]any) (any, error) { return "ok", nil },
	})
	r.Register(&Tool{
		Name: "danger.wipe", Description: "wipe", Destructive: true,
		Handler: func(context.Context, map[string]any) (any, error) { return "wiped", nil },
	})
	return r
}

func TestDestructiveGate(t *testing.T) {
	open := false
	r := gateRegistry(func() bool { return open })
	ctx := context.Background()

	// Read tools are never gated.
	if _, err := r.Execute(ctx, "safe.read", nil); err != nil {
		t.Fatalf("read tool should never be gated: %v", err)
	}

	// Destructive + no consent + closed window → locked.
	if _, err := r.Execute(ctx, "danger.wipe", nil); !errors.Is(err, ErrDestructiveLocked) {
		t.Fatalf("want ErrDestructiveLocked, got %v", err)
	}

	// Consent (human/UI surface) bypasses the window.
	if _, err := r.Execute(WithConsent(ctx), "danger.wipe", nil); err != nil {
		t.Fatalf("consented destructive call should run: %v", err)
	}

	// Open window authorizes an agent (no consent).
	open = true
	if _, err := r.Execute(ctx, "danger.wipe", nil); err != nil {
		t.Fatalf("destructive call inside an open window should run: %v", err)
	}
}

func TestDestructiveLockedByDefaultWithoutWindow(t *testing.T) {
	// A registry with no window checker set treats every destructive agent call
	// as locked (fail-closed), so forgetting to wire the grant can't open a hole.
	r := NewRegistry(nil)
	r.Register(&Tool{
		Name: "danger.wipe", Destructive: true,
		Handler: func(context.Context, map[string]any) (any, error) { return nil, nil },
	})
	if _, err := r.Execute(context.Background(), "danger.wipe", nil); !errors.Is(err, ErrDestructiveLocked) {
		t.Fatalf("want fail-closed lock, got %v", err)
	}
}
