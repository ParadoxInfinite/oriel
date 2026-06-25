package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/ParadoxInfinite/oriel/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// addContext registers MCP resources and prompts on top of the tools. Resources
// let a client attach a container's logs or inspect output as context, each is
// backed by the same validated tool, so masking and the read-only/allow/deny
// scoping still apply (a resource is only offered when its tool is in scope).
// Prompts are canned diagnostics: advisory text that points the model at the
// right tools, so they're always available (they never run anything themselves).
func addContext(s *mcp.Server, reg *tools.Registry, include func(*tools.Tool) bool) {
	inScope := map[string]bool{}
	for _, t := range reg.List() {
		if include == nil || include(t) {
			inScope[t.Name] = true
		}
	}

	if inScope["container.logs"] {
		s.AddResourceTemplate(&mcp.ResourceTemplate{
			Name: "container-logs", URITemplate: "oriel://container/{id}/logs",
			Description: "Recent log lines for a container", MIMEType: "application/json",
		}, resourceFromTool(reg, "container.logs", "oriel://container/", "/logs"))
	}
	if inScope["container.inspect"] {
		s.AddResourceTemplate(&mcp.ResourceTemplate{
			Name: "container-inspect", URITemplate: "oriel://container/{id}/inspect",
			Description: "Full config for a container (env values masked)", MIMEType: "application/json",
		}, resourceFromTool(reg, "container.inspect", "oriel://container/", "/inspect"))
	}

	s.AddPrompt(&mcp.Prompt{
		Name: "diagnose-container", Description: "Investigate why a container is unhealthy or crashing",
		Arguments: []*mcp.PromptArgument{{Name: "container", Title: "container id or name"}},
	}, func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		name := req.Params.Arguments["container"]
		return promptText("Diagnose "+name,
			"Investigate the Docker container \""+name+"\". Read its recent logs (container.logs) and its config "+
				"(container.inspect), then explain what's wrong and recommend a fix. Don't run any destructive action "+
				"without the user opening a grant window."), nil
	})
	s.AddPrompt(&mcp.Prompt{
		Name: "fix-docker-connection", Description: "Fix tools that can't find Docker on a colima machine",
	}, func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return promptText("Fix Docker connection",
			"A tool (e.g. Testcontainers, a docker SDK client) can't find Docker. Call docker.env to get the real "+
				"socket, then tell the user the exact DOCKER_HOST and TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE to export "+
				", colima's socket isn't at /var/run/docker.sock."), nil
	})
	s.AddPrompt(&mcp.Prompt{
		Name: "reclaim-disk", Description: "Find what's safe to prune to reclaim Docker disk",
	}, func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return promptText("Reclaim disk",
			"Call system.df to see Docker disk usage, then list what's safe to remove (dangling images, unused "+
				"volumes). Propose the prune commands but don't run them until the user opens a grant window."), nil
	})
}

// resourceFromTool serves a resource by running the backing read tool: it pulls
// the {id} out of the concrete URI and hands the tool's result back as JSON text.
func resourceFromTool(reg *tools.Registry, tool, prefix, suffix string) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		id := strings.TrimSuffix(strings.TrimPrefix(req.Params.URI, prefix), suffix)
		if id == "" || id == req.Params.URI {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		out, err := reg.Execute(ctx, tool, map[string]any{"id": id})
		if err != nil {
			return nil, err
		}
		body, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{URI: req.Params.URI, MIMEType: "application/json", Text: string(body)}},
		}, nil
	}
}

func promptText(desc, text string) *mcp.GetPromptResult {
	return &mcp.GetPromptResult{
		Description: desc,
		Messages:    []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: text}}},
	}
}
