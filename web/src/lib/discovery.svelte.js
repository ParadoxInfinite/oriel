import { apiGet, apiPut, apiPost } from './api.js'
import { runOp } from './op.svelte.js'
import { refreshStacks } from './stacks.svelte.js'
import { confirm } from './confirm.svelte.js'
import { self } from './self.svelte.js'

// Compose-discovery state, shared by both editions' Settings + Stacks views so
// the directory config, scan results, filter and deploy live in one place. The
// config is persisted server-side (settings.json); writes are debounced and
// trigger a rescan.
export const discovery = $state({
  config: { roots: [], filter: { mode: 'off', patterns: [] }, aliases: {} },
  stacks: [], // discovered, not-yet-deployed, passing the filter
  roots: [], // per-root { id, found, error } feedback
  hidden: 0, // discovered stacks hidden by the filter
  loading: false,
  loaded: false,
})

export async function loadDiscovery() {
  try {
    discovery.config = await apiGet('/api/discovery')
  } catch {
    /* keep defaults */
  }
  return rescan()
}

// Load once on first view open; later opens reuse the existing scan.
export function ensureDiscovery() {
  if (!discovery.loaded && !discovery.loading) return loadDiscovery()
}

// Reveal a project's directory in Finder / the file manager (best-effort).
// fs/open runs on the SERVER, so it only makes sense for a local instance.
export function openDir(path) {
  return apiPost(`/api/fs/open?path=${encodeURIComponent(path)}`).catch(() => {})
}

// True when the UI is viewed from another machine, where opening a folder on the
// server is meaningless.
export function isRemote() {
  const h = location.hostname
  return !(h === 'localhost' || h === '127.0.0.1' || h === '::1' || h === '[::1]')
}

// The verb for the host-folder action, by access mode and server OS.
export function revealLabel() {
  if (isRemote()) return 'Copy path'
  return self.os === 'darwin' ? 'Reveal in Finder' : 'Open folder'
}

// Open the directory on a local server, or copy the path when viewing remotely.
export async function revealOrCopy(path) {
  if (isRemote()) {
    try {
      await navigator.clipboard.writeText(path)
    } catch {
      /* clipboard may be unavailable over plain http — best-effort */
    }
    return
  }
  return openDir(path)
}

export async function rescan() {
  discovery.loading = true
  try {
    const res = await apiGet('/api/discovery/scan')
    discovery.stacks = res.stacks || []
    discovery.roots = res.roots || []
    discovery.hidden = res.hidden || 0
  } catch {
    discovery.stacks = []
  } finally {
    discovery.loading = false
    discovery.loaded = true
  }
}

let saveTimer
function persist() {
  clearTimeout(saveTimer)
  saveTimer = setTimeout(async () => {
    try {
      await apiPut('/api/discovery', $state.snapshot(discovery.config))
    } catch {
      /* best-effort */
    }
    rescan()
  }, 400)
}

// ── Roots ─────────────────────────────────────────────────────────────────────
export function addRoot(path) {
  const p = (path || '').trim()
  if (!p) return
  const id = 'r' + Math.random().toString(36).slice(2, 8)
  discovery.config.roots = [...discovery.config.roots, { id, path: p, traverse: false, enabled: true }]
  persist()
}
export function updateRoot(id, patch) {
  discovery.config.roots = discovery.config.roots.map((r) => (r.id === id ? { ...r, ...patch } : r))
  persist()
}
export function removeRoot(id) {
  discovery.config.roots = discovery.config.roots.filter((r) => r.id !== id)
  persist()
}
export const rootResult = (id) => discovery.roots.find((r) => r.id === id)

// ── Filter ────────────────────────────────────────────────────────────────────
export function setFilter(patch) {
  discovery.config.filter = { ...discovery.config.filter, ...patch }
  persist()
}
export function addPattern(p) {
  const v = (p || '').trim()
  if (!v || discovery.config.filter.patterns.includes(v)) return
  setFilter({ patterns: [...discovery.config.filter.patterns, v] })
}
export function removePattern(p) {
  setFilter({ patterns: discovery.config.filter.patterns.filter((x) => x !== p) })
}
// Per-stack hide: add the project name to a deny list (switching 'off'→'deny').
// Allow-list mode is managed from Settings instead, so this no-ops there.
export function hideStack(name) {
  if (discovery.config.filter.mode === 'allow') return
  const patterns = discovery.config.filter.patterns.includes(name)
    ? discovery.config.filter.patterns
    : [...discovery.config.filter.patterns, name]
  setFilter({ mode: 'deny', patterns })
}

const HIDE_SKIP_KEY = 'oriel.hideSkipConfirm'
// Hide with a confirmation, honouring a persisted "Don't ask again" choice.
export async function confirmHide(d) {
  let skip = false
  try {
    skip = localStorage.getItem(HIDE_SKIP_KEY) === '1'
  } catch {
    /* ignore */
  }
  if (!skip) {
    const res = await confirm({
      title: `Hide “${d.alias || d.name}”?`,
      message: 'It will be removed from the Available list (added to the deny filter). Unhide it any time in Settings → Compose discovery. Running stacks are never affected.',
      confirmLabel: 'Hide',
      danger: false,
      checkbox: "Don't ask again",
    })
    if (!res?.ok) return
    if (res.checked) {
      try {
        localStorage.setItem(HIDE_SKIP_KEY, '1')
      } catch {
        /* ignore */
      }
    }
  }
  hideStack(d.name)
}

// ── Aliases ───────────────────────────────────────────────────────────────────
export function setAlias(name, alias) {
  const a = { ...discovery.config.aliases }
  const v = (alias || '').trim()
  if (v) a[name] = v
  else delete a[name]
  discovery.config.aliases = a
  persist()
}

// ── Deploy + path typeahead ───────────────────────────────────────────────────
export function deployStack(d) {
  const q = `dir=${encodeURIComponent(d.dir)}&file=${encodeURIComponent(d.file)}`
  return runOp(`Deploying ${d.alias || d.name}`, `/api/stacks/up?${q}`, () => {
    refreshStacks()
    rescan()
  })
}

export async function listDirs(path) {
  try {
    return await apiGet(`/api/fs/list?path=${encodeURIComponent(path || '')}`)
  } catch {
    return { dir: path, entries: [] }
  }
}
