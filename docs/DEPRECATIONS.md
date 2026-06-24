# Deprecations

Things being removed from Oriel, and what to do about it. Anything listed here
sticks around for at least one release after it's announced, so there's time to
move.

## In-app natural-language resolver (the AI provider seam)

Deprecated in **v0.5.0**, **removed in v0.6.0**.

### What it was

The Settings → AI panel where you set a "Provider URL", plus the palette's
free-text "Interpret" option that appeared once a provider was set. Underneath:
the `/api/resolve` and `/api/provider` endpoints, the `ORIEL_PROVIDER_URL`
setting, and the `provider` / `setProvider` / `resolveText` exports in the
edition SDK. All gone as of v0.6.0.

The normal command palette is untouched — typing `stop postgres` and picking an
action works exactly as before.

### Why it went

It never earned its place between the two paths that already existed:

- The command palette (⌘K) matches the tool you mean and runs it, with no model
  and no setup.
- The MCP server (`oriel mcp`) hands the same tools to your own model, local or
  hosted, through any MCP client — and that model can reason across multiple
  steps.

The provider seam was a one-shot thing wedged between them: send one sentence to
an external model, get back one tool call. Same permissions as MCP (gated by the
destructive-grant window) but strictly less capable — it couldn't chain calls or
look at a result and decide what to do next. Whatever you'd point it at, a local
model included, the MCP server handles better.

### What to use instead

- For quick, exact actions: the command palette (⌘K). It covers containers,
  images, volumes, networks, prune, and jumping between views.
- For anything conversational or multi-step: the MCP server. Point your client
  (Claude Desktop, Claude Code, Cursor, a local Ollama setup) at `oriel mcp`.
  Setup is in [MCP.md](./MCP.md).

### Version by version

| Version | State |
| --- | --- |
| v0.4.x and earlier | Available, supported. |
| v0.5.x | Deprecated. Still works; Settings shows a notice. Last series that includes it. |
| v0.6.0 | Removed: the AI settings panel, the "Interpret" mode, `/api/resolve`, `/api/provider`, `ORIEL_PROVIDER_URL`, and the SDK `provider` exports. |

Still on v0.5.x and leaning on it? The MCP server does everything it did. Open an
[issue](https://github.com/ParadoxInfinite/oriel/issues) and we'll help you switch.
