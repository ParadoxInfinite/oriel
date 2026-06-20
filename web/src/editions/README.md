# Editions (theme plugins)

An **edition** is a complete, swappable front-end for colima-gui. The host mounts
exactly one at a time; the floating switcher (bottom-left) changes it live and
remembers the choice in `localStorage`. This is the same idea as  ↔
: the data/control layer is fixed, the presentation is a plugin.

Two ship in the box:

| id        | look                                            |
| --------- | ----------------------------------------------- |
| `studio`  | Clean, native-feel light control panel (default) |
| `classic` | Calm teal dark control panel                    |

## The contract

Editions talk to the backend **only** through the platform SDK:

```js
import { status, containers, stats, invoke, lifecycle, fmt } from '@platform'
// (relative: ../../platform/index.js)
```

That module is the entire stable surface — reactive state (`status`,
`containers`, `images`, `volumes`, `networks`, `stacks`, `stats`, `history`,
`self`, `op`) plus actions (`invoke`, `lifecycle`, `stackOp`, `confirm`,
`toast`, `openPalette`) and helpers (`fmt`, `icons`). Never reach into `../lib/*`
directly; if it isn't re-exported from `platform/index.js`, it isn't contract.

An edition is just a Svelte component that renders the whole app from that state.
It does not own polling or the global overlays (op progress, command palette,
confirm dialog, toasts) — the host provides those around it.

## Add a built-in edition

1. Create `editions/<your-id>/Root.svelte`, built on the SDK above.
2. Register it in `editions/registry.svelte.js`:

   ```js
   import Mine from './mine/Root.svelte'
   const BUILTIN = [
     { id: 'mine', name: 'Mine', tagline: 'My take', accent: '#ff7ac6', component: Mine },
     // …existing
   ]
   ```

Scope your CSS (Studio nests everything under `.studio-root`) so themes never
leak into one another.

## Add a runtime edition (no rebuild)

Push a manifest onto `window.__orielThemes` before the app mounts — e.g. from
a `<script>` in `index.html` or an injected bundle. `component` must be a mounted
Svelte component constructor:

```js
window.__orielThemes = [
  { id: 'neon', name: 'Neon', tagline: 'Dropped in at runtime', accent: '#39ff14', component: NeonRoot },
]
```

It appears in the switcher alongside the built-ins.
