# Building editions & themes for Oriel

Oriel's front end is a plugin. The Go backend and the **platform SDK** form a
stable contract; everything you see is an **edition** rendered on top of it. This
guide is for anyone building a new edition, recoloring an existing one, or
shipping a theme others can drop in.

There are three levels of customization, from cheapest to deepest:

1. **Accent / appearance**: recolor Studio (light/dark + a custom accent) from
   **Settings**. No code. See [Appearance](#appearance).
2. **Installed theme**: ship a whole new UI as an ES module, drop it in the
   themes directory, no rebuild. See [Installed themes](#installed-themes).
3. **Built-in edition**: add a first-class edition to the source tree. See
   [Add a built-in edition](#add-a-built-in-edition).

Two editions ship in the box:

| id        | look                                              |
| --------- | ------------------------------------------------- |
| `studio`  | Clean, native-feel; light/dark/system (default)   |

## The contract: the platform SDK

An edition is a Svelte component that renders the whole app from one import
surface: **`src/platform/index.js`**. Import from there and nothing else; if it
isn't re-exported from the platform SDK, it isn't contract and may change.

```js
import { status, containers, invoke, lifecycle, fmt } from '../../platform/index.js'
```

The host owns data fetching (a single push-based live stream, no polling) and
the global overlays (command palette, confirm dialog, toasts, operation-progress).
Your edition just consumes state and renders.

### Reactive state

Every export below is a live Svelte `$state` object; read it in `$derived` or
markup and it updates as the backend does. Don't reassign; mutate via the actions.

| Export | Shape |
| --- | --- |
| `status` | `{ loading, running, error, data }`; `data` = `{ engine, profile, runtime, arch, cpu, memory, disk, kubernetes, driver, version, dockerSocket }` |
| `containers` | `{ list, loading, error }`; item `{ id, name, image, imageId, state, status, created, project, ports:[{public,private,type}] }` |
| `images` | `{ list, … }`; item `{ id, tags:[string], size, containers, created }`: for a digest-pinned image `tags` holds its `repo@sha256:…` |
| `volumes` | `{ list, … }`; item `{ name, driver, mountpoint }` |
| `networks` | `{ list, … }`; item `{ id, name, driver, scope, internal }` |
| `stacks` | `{ list, … }`; item `{ name, running, total, workingDir, containers:[{id,name,state}] }` |
| `stats` | `{ byId: { [id]: { id, cpu, mem, memLimit } } }` |
| `history` | `{ points: [{ t, cpu, mem, down }] }` (rolling ~30 min) |
| `self` | `{ rss, goroutines, heapAlloc }` |
| `outages` | `{ list: [{ start, end, kind }] }`: `kind` ∈ `down` \| `offline` |
| `ops` | `{ list, focused }`: operation tracker driving the progress modal (`focused`) and the sidebar tray (the rest); items can be cancelled/resumed |
| `nav` | `{ view, target }`: the active view id + an optional deep-link intent. Bind your view switch to `nav.view`. See [Navigation](#navigation). |

Refreshers: `refreshContainers`, `refreshImages`, `refreshVolumes`,
`refreshNetworks`, `refreshStacks`, `refreshStatus`.

### Actions

| Export | Use |
| --- | --- |
| `invoke(tool, args, { success })` | run a backend tool; tools include `container.start\|stop\|restart\|remove`, `image.remove\|tag`, `volume.remove`, `network.remove`. |
| `lifecycle('start'\|'stop'\|'restart')` | Colima lifecycle, wired to the op overlay. |
| `stackOp(name, 'start'\|'stop'\|'restart'\|'down', onDone)` | compose actions. |
| `runOp(title, path, onDone)` | drive the op overlay for any streaming endpoint. |
| `startSystemPrune(sel)` / `startImagePrune(items)` / `startVolumePrune(items)` | kick off a server-side prune as a background job (survives refresh, shows in the tray, cancellable). |
| `cancelOp(id)` / `focusOp(id)` / `minimizeOp()` / `resumeOps()` | control entries in `ops` (cancel, open in the modal, hide to the tray, re-attach on load). |
| `confirm({ title, message, confirmLabel })` | → `Promise<boolean>`. |
| `toast(message, 'ok'\|'error'\|'info')` | transient notice. |
| `openPalette()` / `togglePalette()` | command palette. |
| `navigate(view, target?)` | switch the active view; pass a `target` to deep-open an entity at the destination. See [Navigation](#navigation). |
| `takeTarget(view)` | claim + clear a pending intent addressed to `view` (e.g. open a container's logs). |
| `setOverlayTheme(scheme, accent)` | tell the host how to theme the global overlays for your edition: `'base'` (no override, the global palette), or `'light'`/`'dark'` + an accent. |

### Helpers

- `fmt`: `bytes`, `duration`, `timeOnly`, `dateTime`, `relativeTime`, `shortRef` (trim a digest ref for display).
- `icons`: raw Lucide inner-SVG strings, keyed by name.
- `createSort`, `toggleSort`, `sortRows`: sortable-table state.
- `containersForImage(id)`, `isPinnedImage(image)`, `suggestTag(image)`: link an image to the containers using it and propose a tag for a digest-pinned one.
- `REGISTRY_SOURCES`, `searchRegistry`, `listImageTags`: pull-dialog registry data.
- `apiGet`, `apiPost`, `streamPost`, `sse`: low-level fetch/stream for endpoints
  without a dedicated store.
- `PullController` / `LogsController`: headless controllers for the image-pull
  dialog (registry select, live search, tag suggestions, streaming pull) and the
  log viewer (100-line tail, scroll-up lazy-load of older lines, memory bounded).
  Construct one and render its reactive fields; both built-in editions are pure
  markup over them. The model for keeping behavior in the SDK and look in the edition.
- `discovery` + `PathField`: compose directory-discovery store (config, scan,
  filter, deploy) and the headless path-typeahead controller. Both built-in
  editions' Settings + Stacks views render the same store; another model for
  behavior-in-SDK, look-in-edition.
- `AliasEditor`: headless "rename in Oriel" controller. An alias is a display-only
  label keyed by the real compose project name, so one instance drives renaming on
  any stack, running or discovered. `display(name)`, `start/save/cancel`,
  `editing`, `draft`.

## Navigation

The active view is shared state, not an edition's private flag, because the
command palette (a host overlay) and any deep-link have to move *whatever* edition
is mounted. Participating is one binding:

- **MUST:** render your view switch from `nav.view`, and point your nav controls
  at `navigate(view)`:

  ```svelte
  import { nav, navigate, VIEWS } from '../../platform/index.js'

  {#each VIEWS as v}
    <button class:on={nav.view === v} onclick={() => navigate(v)}>{v}</button>
  {/each}

  {#if nav.view === 'Containers'} <Containers /> {:else if …}
  ```

  With this alone, `⌘K → "stop postgres"` and `⌘K → "go to images"` move your
  edition for free. `VIEWS` is the canonical id list.

- **SHOULD:** to honour a deep-link (e.g. "view logs of postgres"), have the
  destination view claim the pending intent and act on it:

  ```svelte
  $effect(() => {
    const t = takeTarget('Containers')
    if (t?.open === 'logs') selected = t.container // open your own logs drawer
  })
  ```

  An edition that skips this still lands on the right view; deep opens are
  graceful, never required. Targets are `{ kind, ...payload, open? }` by convention.

## Add a built-in edition

1. Create `src/editions/<your-id>/Root.svelte`, built on the platform SDK.
2. Register it in `src/editions/registry.svelte.js`:

   ```js
   import Mine from './mine/Root.svelte'
   const BUILTIN = [
     { id: 'mine', name: 'Mine', tagline: 'My take', accent: '#ff7ac6', component: Mine },
     // …existing
   ]
   ```

A manifest is `{ id, name, tagline, accent, component }`. Switch to it from
**Settings → Editions**.

**Scope your CSS.** Each edition nests everything under a root class (Studio uses
`.studio-root`) so themes never leak into one another. Pick a unique root class
and prefix your styles with it.

## Installed themes

Ship an edition as a single ES-module bundle (`*.js`) that **default-exports** a
manifest, then drop it into Oriel's themes directory: no rebuild, no URLs.
Oriel discovers it at startup, serves it same-origin, and lists it under
**Settings → Editions & themes**. The exact directory is shown there (typically
`~/.config/oriel/themes`, or `~/Library/Application Support/oriel/themes` on macOS).

```js
// my-theme.js
export default { id: 'neon', name: 'Neon', tagline: 'Installed theme', accent: '#39ff14', component: NeonRoot }
```

`component` must be a Svelte component constructor compiled against the same
Svelte runtime. Themes are **code you choose to install**: there's deliberately
no load-by-URL, so a link can't trick anyone into running a malicious theme.

For ephemeral/dev use you can also pre-register before mount via
`window.__orielThemes = [{ id, name, component, … }]`.

## Appearance

Studio governs its own look independently of the registry: **light / dark /
system**, plus a **per-theme accent** (light defaults to indigo, dark to teal),
and user-created custom accents, all from **Settings → Appearance**, persisted
locally under `oriel.appearance`. If you build your own edition and want the same
capability, model it on `src/editions/studio/theme.svelte.js`.

## Notes

- The global overlays (command palette, confirm, toasts, op-progress) are
  host-provided. Call `setOverlayTheme(scheme, accent)` from your edition (e.g. in
  an `$effect`) and the host restyles them to match. Studio does this for its
  light/dark + accent.
- The active view is shared state (`nav`), so the palette and deep-links can move
  whichever edition is mounted. Bind your view switch to `nav.view`. See
  [Navigation](#navigation). It's the one piece of cross-cutting wiring an edition
  must adopt.
- Keep editions thin: business logic belongs behind the platform SDK so every
  edition benefits. If you find yourself reimplementing backend behavior, that's a
  signal it should be lifted into the SDK instead.
