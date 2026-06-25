package actions

import (
	"context"

	"github.com/ParadoxInfinite/oriel/internal/colima"
	"github.com/ParadoxInfinite/oriel/internal/tools"
)

// registerColima wires Colima VM lifecycle as tools so an agent can manage the
// machine, not just read colima.status. Each blocks until the action finishes
// (synchronous wrapper over colima.Stream; the UI streams for live progress).
// stop and restart take the whole VM, and every container, down, so they're
// Destructive (grant-gated); start only brings it up, so it stays free.
func registerColima(r *tools.Registry) {
	run := func(action string) func(context.Context, map[string]any) (any, error) {
		return func(ctx context.Context, _ map[string]any) (any, error) {
			out, err := colima.Run(ctx, action)
			if err != nil {
				return nil, err
			}
			return map[string]any{"ok": true, "output": out}, nil
		}
	}

	r.Register(&tools.Tool{
		Name: "colima.start", Title: "Start Colima", Description: "Start the Colima VM",
		Handler: run("start"),
	})
	r.Register(&tools.Tool{
		Name: "colima.stop", Title: "Stop Colima", Description: "Stop the Colima VM, this stops every container on it",
		Destructive: true,
		Handler:     run("stop"),
	})
	r.Register(&tools.Tool{
		Name: "colima.restart", Title: "Restart Colima", Description: "Restart the Colima VM, this bounces every container on it",
		Destructive: true,
		Handler:     run("restart"),
	})
}
