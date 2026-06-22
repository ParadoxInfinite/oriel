# Editions (theme plugins)

An **edition** is a complete, swappable front-end for Oriel. The host mounts
exactly one at a time; **Settings** changes it live and remembers the choice in
`localStorage`. The data/control layer is fixed; the presentation is a plugin.

Two ship in the box:

| id        | look                                                  |
| --------- | ----------------------------------------------------- |
| `studio`  | Clean, native-feel light/dark control panel (default) |
| `classic` | Calm teal dark control panel                          |

## The contract

Editions talk to the backend **only** through the platform SDK:

```js
import { status, containers, stats, invoke, lifecycle, fmt } from '@platform'
// (relative: ../../platform/index.js)
```

That module is the entire stable surface. Highlights:

- **Reactive state:** `status`, `containers`, `images`, `volumes`, `networks`,
  `stacks`, `stats`, `history`, `self`, `outages`, `ops` (the multi-op tracker
  that drives the progress modal + sidebar tray).
- **Actions:** `invoke`, `lifecycle`, `stackOp`, `confirm`, `toast`,
  `openPalette`; prune starters `startSystemPrune` / `startImagePrune` /
  `startVolumePrune`; op controls `cancelOp` / `resumeOps` / `focusOp` /
  `minimizeOp`.
- **Headless controllers:** `PullController`, `LogsController` own the stateful
  logic so editions stay presentational.
- **Helpers:** `fmt`, `icons`, `containersForImage`, `isPinnedImage`,
  `suggestTag`, `setOverlayTheme`, plus the discovery store.

Never reach into `../lib/*` directly; if it isn't re-exported from
`platform/index.js`, it isn't contract.

An edition is a Svelte component that renders the whole app from that state. It
does **not** own data fetching or the global overlays (op progress, command
palette, confirm dialog, toasts). The host owns the single push-based live
stream (no polling) and mounts the overlays around the edition. It *does* publish
how those overlays should look via `setOverlayTheme(scheme, accent)` so they
match the active edition (see `lib/overlayTheme.svelte.js`).

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

Push a manifest onto `window.__orielThemes` before the app mounts, e.g. from
a `<script>` in `index.html` or an injected bundle. `component` must be a mounted
Svelte component constructor:

```js
window.__orielThemes = [
  { id: 'neon', name: 'Neon', tagline: 'Dropped in at runtime', accent: '#39ff14', component: NeonRoot },
]
```

It appears in the switcher alongside the built-ins.
