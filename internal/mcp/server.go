// Package mcp exposes Oriel's validated tool Registry as a Model Context
// Protocol server over stdio. Every MCP client (Claude Desktop, Claude Code,
// Cursor, a local Ollama-backed host) speaks the same JSON-RPC-over-stdio
// transport, so a single `oriel mcp` process lets any of them drive Docker and
// Colima through the exact same execution path the UI uses — schema-validated,
// entity-checked, with the same secret masking. No model ships in the binary;
// the model lives in the client.
//
// Tools map one-to-one to the registry. Destructive tools carry a destructive
// hint and are gated by the time-boxed grant: the MCP path never sets consent,
// so `Registry.Execute` locks remove/prune unless a grant window is open (wired
// in mcp_cmd.go via SetDestructiveWindow).
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ParadoxInfinite/oriel/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Serve runs the MCP server over stdio until the client disconnects (EOF) or
// ctx is cancelled. It registers one MCP tool per registry tool that `include`
// admits (a nil include exposes everything) — that's how `oriel mcp --read-only`
// and the allow/deny lists scope the surface handed to a client.
func Serve(ctx context.Context, reg *tools.Registry, version string, include func(*tools.Tool) bool) error {
	return newServer(reg, version, include).Run(ctx, &mcp.StdioTransport{})
}

// newServer builds the MCP server with one tool per admitted registry tool.
// Split out so tests can drive it over an in-memory transport instead of stdio.
func newServer(reg *tools.Registry, version string, include func(*tools.Tool) bool) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "oriel", Version: version}, nil)
	for _, t := range reg.List() {
		if include != nil && !include(t) {
			continue
		}
		s.AddTool(toolFor(t), handlerFor(reg, t.Name))
	}
	addContext(s, reg, include)
	return s
}

// toolFor translates a registry tool into its MCP descriptor, including the
// read-only / destructive hints clients use to decide whether to auto-approve.
func toolFor(t *tools.Tool) *mcp.Tool {
	destructive := t.Destructive
	desc := t.Description
	if destructive {
		desc += " (destructive — modifies state)"
	}
	return &mcp.Tool{
		Name:        t.Name,
		Title:       t.Title,
		Description: desc,
		InputSchema: inputSchema(t.Schema),
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    t.ReadOnly,
			DestructiveHint: &destructive,
		},
	}
}

// inputSchema renders our compact Schema as a JSON Schema object. The SDK
// marshals whatever we hand it, so a plain map keeps this dependency-light.
func inputSchema(s tools.Schema) map[string]any {
	props := map[string]any{}
	for name, p := range s.Props {
		entry := map[string]any{"type": p.Type}
		if p.Description != "" {
			entry["description"] = p.Description
		}
		if len(p.Enum) > 0 {
			entry["enum"] = p.Enum
		}
		props[name] = entry
	}
	schema := map[string]any{"type": "object", "properties": props}
	if len(s.Required) > 0 {
		schema["required"] = s.Required
	}
	return schema
}

// handlerFor adapts the registry's Execute into an MCP tool handler. Argument
// and entity validation already live in Execute, so this only marshals in/out.
func handlerFor(reg *tools.Registry, name string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := map[string]any{}
		if raw := req.Params.Arguments; len(raw) > 0 {
			if err := json.Unmarshal(raw, &args); err != nil {
				return errResult(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
		}
		out, err := reg.Execute(ctx, name, args)
		if err != nil {
			// Tool errors are returned in-band (IsError) so the model sees them
			// and can self-correct, per the MCP guidance.
			return errResult(err.Error()), nil
		}
		return textResult(out), nil
	}
}

func textResult(v any) *mcp.CallToolResult {
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errResult(fmt.Sprintf("encode result: %v", err))
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(body)}}}
}

func errResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: msg}}}
}
