// Package tools is the canonical action layer. Single-entity mutations, from a
// UI button, the command palette, or the MCP server, route through
// Registry.Execute, which validates arguments and entity
// references and gates destructive tools before running the handler. (Bulk
// prune runs as a background job over a user-selected list; see
// internal/server/ops.go.) Safety lives here, in the base.
package tools

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// ErrDestructiveLocked is returned when a Destructive tool is invoked by a
// non-interactive caller (no consent) while no grant window is open. The message
// tells an MCP client / assistant how to unlock.
var ErrDestructiveLocked = errors.New("destructive action locked: open a grant window (`oriel ai allow-destructive --for 15m`) or run it from the Oriel UI")

// consentKey marks a context as a trusted, human-initiated call (e.g. a UI/
// palette action behind a confirm dialog). Such calls bypass the grant window;
// agent callers (MCP) never set it.
type consentKey struct{}

// WithConsent marks ctx as a human-confirmed call, allowing Destructive tools
// without a grant window. Set it only on genuinely interactive surfaces.
func WithConsent(ctx context.Context) context.Context {
	return context.WithValue(ctx, consentKey{}, true)
}

func consented(ctx context.Context) bool {
	v, _ := ctx.Value(consentKey{}).(bool)
	return v
}

// HasConsent reports whether ctx was marked human-confirmed via WithConsent.
// Lets handlers give non-consented (agent / MCP) callers a stricter
// floor than an interactive user.
func HasConsent(ctx context.Context) bool { return consented(ctx) }

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
	ReadOnly    bool       `json:"readOnly"` // pure read, no state change (start/stop mutate but aren't destructive)
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
	// windowOpen reports whether a destructive grant window is currently open.
	// nil means "never open", destructive calls then require consent.
	windowOpen func() bool
	// audit, if set, records each non-consented (agent / MCP) call. nil = off.
	audit func(name string, args map[string]any, err error)
}

func NewRegistry(resolver EntityResolver) *Registry {
	return &Registry{tools: map[string]*Tool{}, resolver: resolver}
}

// SetDestructiveWindow injects the grant-window check used to authorize
// Destructive tools for non-interactive callers.
func (r *Registry) SetDestructiveWindow(open func() bool) { r.windowOpen = open }

// SetAuditLog injects a recorder for agent tool calls. Consented (UI) calls are
// never recorded; the operator's own clicks aren't audited.
func (r *Registry) SetAuditLog(fn func(name string, args map[string]any, err error)) { r.audit = fn }

func (r *Registry) windowActive() bool { return r.windowOpen != nil && r.windowOpen() }

// Register adds a tool. It panics on a duplicate name or a malformed entity ref
// , both programming errors caught at startup rather than at call time. The
// handlers rely on the schema guaranteeing the entity param is a present string,
// so enforce that invariant here.
func (r *Registry) Register(t *Tool) {
	if _, dup := r.tools[t.Name]; dup {
		panic("tools: duplicate registration " + t.Name)
	}
	// A tool whose name reads as an irreversible action must carry Destructive,
	// or it would inherit the ungated default and run for an MCP/agent caller with
	// no grant. This is a startup tripwire so a new `*.remove`/`*.prune`/… tool
	// can't ship unflagged; the golden classification test (internal/actions)
	// covers the cases this name heuristic can't see.
	if !t.Destructive && destructiveName(t.Name) {
		panic("tools: " + t.Name + ": name implies a destructive action but Destructive is false; set Destructive: true (or rename)")
	}
	if t.Entity != nil {
		p, ok := t.Schema.Props[t.Entity.Param]
		if !ok || p.Type != "string" || !slices.Contains(t.Schema.Required, t.Entity.Param) {
			panic("tools: " + t.Name + ": entity param " + t.Entity.Param + " must be a required string in the schema")
		}
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

// Execute gates, validates, and runs a single tool call, then records it to the
// audit log when the caller is an agent (non-consented). The operator's own
// UI/palette calls carry consent and are not recorded.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (any, error) {
	res, err := r.execute(ctx, name, args)
	if r.audit != nil && !consented(ctx) {
		r.audit(name, args, err) // every agent call: success, gate-block, or error
	}
	return res, err
}

// execute is the gate/validate/run path: it locks destructive tools without
// consent or an open grant window, schema-validates args, and checks entity
// existence before invoking the handler.
func (r *Registry) execute(ctx context.Context, name string, args map[string]any) (any, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownTool, name)
	}
	if t.Destructive && !consented(ctx) && !r.windowActive() {
		return nil, ErrDestructiveLocked
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
	slices.SortFunc(ts, func(a, b *Tool) int { return cmp.Compare(a.Name, b.Name) })
}

// destructiveVerbs are name suffixes that denote an irreversible action. A tool
// named `<noun>.<verb>` (or bare `<verb>`) ending in one of these must be
// flagged Destructive. Reversible mutations (start/stop/restart/create/connect/
// tag/alias) are deliberately absent, they stay ungated by design.
var destructiveVerbs = []string{"remove", "delete", "prune", "destroy", "rm", "kill", "down", "wipe", "purge", "drop"}

// destructiveName reports whether a tool name's verb denotes an irreversible
// action. The verb is the segment after the last dot, or the whole name.
func destructiveName(name string) bool {
	verb := name
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		verb = name[i+1:]
	}
	return slices.Contains(destructiveVerbs, verb)
}
