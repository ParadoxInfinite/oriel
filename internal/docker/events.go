package docker

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types/events"
)

// Event is a trimmed docker event used to trigger live UI refreshes.
type Event struct {
	Type   string `json:"type"`
	Action string `json:"action"`
	ID     string `json:"id"`
	Name   string `json:"name"`
}

// relevantTypes are the object kinds whose changes the UI cares about.
var relevantTypes = map[events.Type]bool{
	events.ContainerEventType: true,
	events.ImageEventType:     true,
	events.VolumeEventType:    true,
	events.NetworkEventType:   true,
}

// isNoisyAction reports actions that fire constantly but change no list the UI
// shows — chiefly the exec_* trio every health-check emits per container, plus
// other inspection/interactive actions. Dropping them avoids a refetch storm.
func isNoisyAction(a string) bool {
	if strings.HasPrefix(a, "exec_") {
		return true
	}
	switch a {
	case "top", "attach", "resize", "export", "save", "commit":
		return true
	}
	return false
}

// StreamEvents emits relevant docker events until ctx is cancelled. The error
// channel yields a terminal error (e.g. the daemon going away).
func (c *Client) StreamEvents(ctx context.Context) (<-chan Event, <-chan error, error) {
	cli, err := c.api(ctx)
	if err != nil {
		return nil, nil, err
	}
	msgs, errs := cli.Events(ctx, events.ListOptions{})

	out := make(chan Event, 16)
	go func() {
		defer close(out)
		for m := range msgs {
			if !relevantTypes[m.Type] || isNoisyAction(string(m.Action)) {
				continue
			}
			ev := Event{
				Type:   string(m.Type),
				Action: string(m.Action),
				ID:     m.Actor.ID,
				Name:   m.Actor.Attributes["name"],
			}
			select {
			case out <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, errs, nil
}
