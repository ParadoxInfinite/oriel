// Package tools is the canonical action layer. Every mutating action — whether
// triggered by a UI button, the command palette, or a future NL provider —
// routes through Registry.Execute, which validates arguments and entity
// references before running the handler. Safety lives here, in the base.
package tools

import (
	"context"
	"errors"
	"fmt"
)

// ErrUnknownTool is returned when a tool name is not registered.
var ErrUnknownTool = errors.New("unknown tool")

// Handler performs an action. args have already been schema-validated.
type Handler func(ctx context.Context, args map[string]any) (any, error)

// EntityRef declares that one argument references a live entity that must exist
// before the handler runs. The executor enforces existence via the resolver.
type EntityRef struct {
	Param string // argument key holding the id/name
	Kind  string // "container", "image", "volume", "network", "stack"
}

// Tool is a single registered action.
type Tool struct {
	Name        string     `json:"name"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Schema      Schema     `json:"schema"`
	Entity      *EntityRef `json:"-"`
	Destructive bool       `json:"destructive"`
	Handler     Handler    `json:"-"`
}

// EntityResolver checks whether a referenced entity exists in live state.
type EntityResolver interface {
	Exists(ctx context.Context, kind, idOrName string) (bool, error)
}

// Registry holds the tool set and the optional entity resolver.
type Registry struct {
	tools    map[string]*Tool
	resolver EntityResolver
}

func NewRegistry(resolver EntityResolver) *Registry {
	return &Registry{tools: map[string]*Tool{}, resolver: resolver}
}

// Register adds a tool. It panics on duplicate names — a programming error.
func (r *Registry) Register(t *Tool) {
	if _, dup := r.tools[t.Name]; dup {
		panic("tools: duplicate registration " + t.Name)
	}
	r.tools[t.Name] = t
}

// List returns the tools sorted by name, for the palette and provider context.
func (r *Registry) List() []*Tool {
	out := make([]*Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	sortByName(out)
	return out
}

// Execute validates and runs a tool call. This is the single execution path.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (any, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownTool, name)
	}
	if args == nil {
		args = map[string]any{}
	}
	if err := t.Schema.Validate(args); err != nil {
		return nil, err
	}
	if t.Entity != nil && r.resolver != nil {
		id, _ := args[t.Entity.Param].(string)
		exists, err := r.resolver.Exists(ctx, t.Entity.Kind, id)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("%s %q not found", t.Entity.Kind, id)
		}
	}
	return t.Handler(ctx, args)
}

func sortByName(ts []*Tool) {
	for i := 1; i < len(ts); i++ {
		for j := i; j > 0 && ts[j-1].Name > ts[j].Name; j-- {
			ts[j-1], ts[j] = ts[j], ts[j-1]
		}
	}
}
