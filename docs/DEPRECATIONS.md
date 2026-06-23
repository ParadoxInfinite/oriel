# Deprecations

Things being removed from Oriel, and what to do about it. Anything listed here
sticks around for at least one release after it's announced, so there's time to
move.

## In-app natural-language resolver (the AI provider seam)

Deprecated in **v0.5.0**. Removed in **v0.6.0**.

It's a niche feature that never saw much use or testing, so it's going on a short
clock.

**Want to keep using it for now?** Stay on the **v0.5.x** series, the last one
that ships it.

**Think it should stay in Oriel for good?** Say so. Open an
[issue](https://github.com/ParadoxInfinite/oriel/issues) (or a discussion) and
tell us what you use it for. 👍 reactions count as votes. If enough people want
it, we'll push the removal back or keep it.

### What this is

The Settings → AI panel where you set a "Provider URL", plus the palette's
free-text "Interpret" option that appears once a provider is set. Underneath,
that's the `/api/resolve` and `/api/provider` endpoints, the `ORIEL_PROVIDER_URL`
setting, and the `provider`, `setProvider`, and `resolveText` exports in the
edition SDK.

The normal command palette is untouched. Typing `stop postgres` and picking an
action works exactly as before.

### Why it's going

It never really earned its place. There are already two ways to act:

- The command palette (⌘K) matches the tool you mean and runs it, with no model
  and no setup.
- The MCP server (`oriel mcp`) hands the same tools to your own model, local or
  hosted, through any MCP client. That model can reason across multiple steps.

The provider seam was a one-shot thing wedged between them: send one sentence to
an external model, get back one tool call. It runs with the same permissions as
MCP (gated by the destructive-grant window) but does less. It can't chain calls,
and it can't look at a result and decide what to do next. Whatever you'd point it
at, a local model included, the MCP server handles better. A third path that's
weaker than the other two isn't worth keeping around.

### What to use instead

- For quick, exact actions: the command palette (⌘K). It now covers containers,
  images, volumes, networks, prune, and jumping between views.
- For anything conversational or multi-step: the MCP server. Point your client
  (Claude Desktop, Claude Code, Cursor, a local Ollama setup) at `oriel mcp`.
  Setup is in [MCP.md](./MCP.md).

### Version by version

| Version | State |
| --- | --- |
| v0.4.x and earlier | Available, supported. |
| v0.5.x | Deprecated. Still works; Settings shows a notice. Last series that includes it. |
| v0.6.0 | Removed: the AI settings panel, the "Interpret" mode, `/api/resolve`, `/api/provider`, and the SDK `provider` exports. |

Need a hand moving over? The MCP server does everything this did. Open an issue
and we'll help you switch.
