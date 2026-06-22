// ============================================================================
// PLATFORM SDK — the stable contract every edition is built against.
//
// An "edition" is a complete, swappable front-end for Oriel. The host loads
// exactly one edition and hands it this module: a single, documented surface
// of reactive state + actions. As
// long as an edition only imports from `@platform`, the backend, the polling,
// and the live event plumbing can all change underneath it without the edition
// noticing — and a third party can ship a totally different look by consuming
// the same seam.
//
// RULE FOR EDITION AUTHORS: import from here, never from ../lib/* directly.
// ============================================================================

// ── Reactive state ──────────────────────────────────────────────────────────
// Every export below is a live Svelte $state object; read it in $derived/markup
// and it updates as the backend does. Do not reassign the containers; mutate
// through the action helpers instead.

/** Colima/engine status. `{ loading, running, error, data }`.
 *  data → { engine, profile, runtime, arch, cpu, memory, disk, kubernetes,
 *           driver, version, dockerSocket } */
export { status, refreshStatus } from '../lib/status.svelte.js'

/** Containers. `{ list, loading, error }`.
 *  item → { id, name, image, state, status, created, project, ports:[{public,private,type}] } */
export { containers, refreshContainers, containersForImage, isPinnedImage, suggestTag } from '../lib/containers.svelte.js'

/** Image/volume/network inventories, each `{ list, loading, error }`.
 *  image  → { id, tags:[string], size, containers, created }
 *  volume → { name, driver, mountpoint }
 *  network→ { id, name, driver, scope, internal } */
export {
  images,
  volumes,
  networks,
  refreshImages,
  refreshVolumes,
  refreshNetworks,
} from '../lib/resources.svelte.js'

/** Compose stacks. `{ list, loading, error }`.
 *  item → { name, running, total, workingDir, containers:[{id,name,state}] } */
export { stacks, refreshStacks } from '../lib/stacks.svelte.js'

/** Live telemetry from the event streams.
 *  stats.byId[id] → { id, cpu, mem, memLimit }
 *  history.points → [{ t, cpu, mem, down }] (rolling ~30 min) */
export { stats, history, connection } from '../lib/live.svelte.js'

/** The Oriel backend's own footprint + build version: `{ version, rss, goroutines, heapAlloc }`. */
export { self } from '../lib/self.svelte.js'

/** Remote-access allow-list (non-loopback Hosts permitted to reach /api).
 *  `RemoteHostForm` is the headless "add a host" input both editions render. */
export { remote, loadRemote, addRemoteHost, removeRemoteHost, RemoteHostForm } from '../lib/remote.svelte.js'

/** Update check + self-update (service installs): `update` state, `checkUpdate()`,
 *  `startUpdateChecks()`/`stopUpdateChecks()` (mount + 3h re-check),
 *  `applyUpdate()` (download+verify+replace), `restartService()`. */
export { update, checkUpdate, checkNow, startUpdateChecks, stopUpdateChecks, applyUpdate, restartService, canSelfUpdate, promptUpdate } from '../lib/update.svelte.js'

/** Persisted downtime log (~30-day window). outages.list → [{ start, end, kind }]
 *  where kind is 'down' (colima unreachable) or 'offline' (gui itself wasn't running). */
export { outages } from '../lib/outages.svelte.js'

/** Operation tracker: lifecycle/compose ops plus background prune jobs that run
 *  concurrently, survive a refresh, and can be cancelled. `ops.list` feeds the
 *  modal overlay (ops.focused) and the sidebar tray (the rest). */
export { ops, runOp, dismissOp, cancelOp, resumeOps, focusOp, minimizeOp, startSystemPrune, startImagePrune, startVolumePrune } from '../lib/op.svelte.js'

// ── Actions ─────────────────────────────────────────────────────────────────

/** invoke(tool, args, { success }) → runs a backend tool, toasts the outcome.
 *  Tools: container.start|stop|restart|remove, image.remove, volume.remove,
 *  network.remove. Returns the result, or false on failure. */
export { invoke } from '../lib/invoke.js'

/** toast(message, level) where level ∈ 'ok'|'error'|'info'. */
export { toast } from '../lib/toast.svelte.js'

/** confirm({ title, message, confirmLabel }) → Promise<boolean>. */
export { confirm } from '../lib/confirm.svelte.js'

/** Command-palette controls, shared across editions. */
export { openPalette, togglePalette } from '../lib/palette.svelte.js'

/** Column-sort state + helpers for tables: createSort, toggleSort, sortRows. */
export { createSort, toggleSort, sortRows } from '../lib/sort.svelte.js'

/** Natural-language provider (the AI seam): state + runtime config + resolver.
 *  provider → { enabled, url }. setProvider(url) swaps + persists the endpoint. */
export { provider, checkProvider, setProvider, resolveText } from '../lib/provider.svelte.js'

/** Formatting helpers: bytes, duration, timeOnly, dateTime, relativeTime. */
export * as fmt from '../lib/format.js'

/** Headless image tag/used-by/remove controller, shared by both editions. */
export { ImageActions } from '../lib/imageActions.svelte.js'

/** Headless AI-provider Settings controller (resolver URL + test box). */
export { ProviderSettings } from '../lib/providerSettings.svelte.js'

/** Raw Lucide icon inner-SVG strings, keyed by name (see lib/icons.js). */
export { icons } from '../lib/icons.js'

/** Low-level fetch helpers for endpoints without a dedicated store: apiGet/apiPost
 *  (e.g. /api/volumes/prune/preview) and streamPost for SSE-over-POST actions
 *  (e.g. /api/images/pull). Prefer the typed stores where they exist. */
export { apiGet, apiPost, apiPut, apiDelete, streamPost, sse } from '../lib/api.js'

/** Destructive-grant window for MCP/assistant: `grant` state + `loadGrant()`,
 *  `openGrant(hours)`, `lockGrant()`. */
export { grant, loadGrant, openGrant, requestGrant, lockGrant, fmtRemaining } from '../lib/grant.svelte.js'

/** Public-registry helpers for the pull dialog: the source list plus search and
 *  tag-listing proxies (Docker Hub, Quay, AWS ECR Public). */
export { REGISTRY_SOURCES, REGISTRY_HOSTS, searchRegistry, listImageTags } from '../lib/registry.js'

/** Headless image-pull controller: all the dialog behaviour (registry select,
 *  live search, tag suggestions, streaming pull) so editions render only markup. */
export { PullController, fmtStars } from '../lib/pull.svelte.js'

/** Headless log buffer: seeds 100 latest, tails live, lazy-loads older, trims memory. */
export { LogsController } from '../lib/logs.svelte.js'

/** Tell the host how to theme the global overlays for this edition:
 *  setOverlayTheme('classic' | 'light' | 'dark', accent?). */
export { setOverlayTheme } from '../lib/overlayTheme.svelte.js'

/** Compose discovery: find/configure/deploy compose projects on disk. `discovery`
 *  holds { config, stacks, roots, hidden }; the rest mutate it and re-scan. */
export {
  discovery,
  loadDiscovery,
  ensureDiscovery,
  rescan,
  addRoot,
  updateRoot,
  removeRoot,
  rootResult,
  setFilter,
  addPattern,
  removePattern,
  hideStack,
  confirmHide,
  setAlias,
  deployStack,
  openDir,
  revealLabel,
  revealOrCopy,
  listDirs,
  FILTER_MODES,
  DiscoveryForm,
} from '../lib/discovery.svelte.js'

/** Headless directory typeahead controller (Radarr-style) for path inputs. */
export { PathField, baseName } from '../lib/pathfield.svelte.js'

/** Canonical external links (repo, theme-authoring guide). */
export { REPO_URL, THEMES_DOC_URL } from '../lib/links.js'

// ── Lifecycle convenience ────────────────────────────────────────────────────
// The start/stop/restart control every edition needs, wired to the shared op
// overlay and a status refresh on completion.
import { runOp } from '../lib/op.svelte.js'
import { refreshStatus } from '../lib/status.svelte.js'

const LIFECYCLE_TITLES = {
  start: 'Starting Colima',
  stop: 'Stopping Colima',
  restart: 'Restarting Colima',
}

/** lifecycle('start'|'stop'|'restart') → drives the op overlay, refreshes status. */
export function lifecycle(action) {
  return runOp(LIFECYCLE_TITLES[action], `/api/colima/${action}`, refreshStatus)
}

/** stackOp(name, 'start'|'stop'|'restart'|'down') → drives the op overlay. */
export function stackOp(name, action, onDone) {
  const verb = action[0].toUpperCase() + action.slice(1)
  return runOp(`${verb} ${name}`, `/api/stacks/${name}/${action}`, onDone)
}
