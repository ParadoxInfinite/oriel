import { streamPost, apiGet, apiPost, sse } from './api.js'
import { toast } from './toast.svelte.js'
import { refreshContainers } from './containers.svelte.js'
import { refreshImages, refreshVolumes, refreshNetworks } from './resources.svelte.js'

// Operation tracker. Several operations can run at once; `list` holds them all and
// `focused` is the one shown in the modal overlay (null = modal closed). The rest
// surface in the sidebar tray. Two flavours share the model:
//   • request-tied (runOp): colima lifecycle, compose, stream lives with the POST.
//   • background jobs (attachJob): prune, run server-side, survive a refresh, and
//     can be cancelled. `jobId` is set only for these.
export const ops = $state({ list: [], focused: null })

let seq = 0
const streams = new Map() // entry id → EventSource (background jobs only)

const find = (id) => ops.list.find((o) => o.id === id)

function add(entry) {
  ops.list.push(entry)
  return entry
}

function drop(id) {
  const i = ops.list.findIndex((o) => o.id === id)
  if (i >= 0) ops.list.splice(i, 1)
  if (ops.focused === id) ops.focused = null
}

// runOp streams a request-tied operation (the stream lives with the POST, so it
// can't outlive the page). Kept for colima lifecycle + compose.
export async function runOp(title, path, onDone) {
  const id = `r${++seq}`
  add({ id, jobId: null, kind: 'task', title, lines: [], cur: 0, total: 0, done: false, error: null, cancelling: false })
  ops.focused = id
  try {
    await streamPost(path, {
      onEvent: (name, data) => {
        const o = find(id)
        if (!o) return
        if (name === 'line') o.lines.push(data.line)
        else if (name === 'done') finishEntry(o, data.ok, data.error)
      },
    })
  } catch (e) {
    const o = find(id)
    if (o) finishEntry(o, false, e.message)
  } finally {
    onDone?.()
  }
}

// finishEntry marks an op done and, on success, auto-dismisses it shortly after so
// finished ops don't pile up in the modal/tray. Failures stay until dismissed.
function finishEntry(o, ok, error) {
  o.done = true
  o.cancelling = false
  if (!ok) {
    o.error = error || 'cancelled'
    return
  }
  const last = o.lines[o.lines.length - 1]
  if (last && o.kind?.includes('prune')) toast(last, 'ok')
  o._dismissTimer = setTimeout(() => dismissOp(o.id), AUTO_DISMISS_MS)
}

const AUTO_DISMISS_MS = 4000

// attachJob streams a server-side background job. The stream sends a "snapshot" of
// progress so far (so reconnect/refresh catches up without duplicates), then live
// lines, then "done". focus=false adds it to the tray without opening the modal
// (used when re-attaching after a refresh).
export function attachJob(jobId, title, kind, focus = true) {
  if (streams.has(jobId)) {
    if (focus) ops.focused = jobId
    return
  }
  if (!find(jobId)) {
    add({ id: jobId, jobId, kind, title, lines: [], cur: 0, total: 0, done: false, error: null, cancelling: false })
  }
  if (focus) ops.focused = jobId
  const es = sse(`/api/ops/${jobId}/stream`, ['snapshot', 'line', 'progress', 'done'], (name, data) => {
    const o = find(jobId)
    if (!o) return
    if (name === 'snapshot') {
      o.lines = data.lines || []
      o.cur = data.cur || 0
      o.total = data.total || 0
    } else if (name === 'line') o.lines.push(data.line)
    else if (name === 'progress') {
      o.cur = data.cur
      o.total = data.total
    } else if (name === 'done') {
      closeStream(jobId)
      refreshForKind(kind)
      finishEntry(o, data.ok, data.error)
    }
  })
  streams.set(jobId, es)
}

// startJob kicks off a background op (POST returns its id), then attaches to it.
async function startJob(path, body, title, kind) {
  let res
  try {
    res = await apiPost(path, body)
  } catch (e) {
    const id = `r${++seq}`
    add({ id, jobId: null, kind, title, lines: [e.message], cur: 0, total: 0, done: true, error: e.message, cancelling: false })
    ops.focused = id
    return
  }
  if (res?.id) attachJob(res.id, title, kind)
}

// sel: { containers, images, networks, cache, volumes } booleans.
export function startSystemPrune(sel) {
  const q = new URLSearchParams(Object.fromEntries(Object.entries(sel).map(([k, v]) => [k, v ? 'true' : 'false'])))
  return startJob(`/api/ops/system-prune?${q}`, null, 'Reclaiming disk space', 'system-prune')
}

// items: [{ id, size }]. The server reports how much was reclaimed.
export function startImagePrune(items) {
  return startJob('/api/ops/image-prune', { items }, `Pruning ${items.length} image${items.length === 1 ? '' : 's'}`, 'image-prune')
}
export function startVolumePrune(items) {
  return startJob('/api/ops/volume-prune', { items }, `Pruning ${items.length} volume${items.length === 1 ? '' : 's'}`, 'volume-prune')
}

// cancelOp asks the server to stop a background job; its "done" event follows.
export async function cancelOp(id) {
  const o = find(id)
  if (!o?.jobId || o.done) return
  o.cancelling = true
  try {
    await apiPost(`/api/ops/${o.jobId}/cancel`, null)
  } catch {
    o.cancelling = false
  }
}

export function focusOp(id) {
  ops.focused = id
}

// minimizeOp closes the modal but keeps every operation running; they show in the
// sidebar tray until done and dismissed.
export function minimizeOp() {
  ops.focused = null
}

// dismissOp removes a finished operation from the tracker (and its modal/tray row).
export function dismissOp(id) {
  const o = ops.list.find((x) => x.id === id)
  if (o?._dismissTimer) clearTimeout(o._dismissTimer)
  closeStream(id)
  drop(id)
}

// resumeOps re-attaches to any still-running background jobs after a page load, so
// in-flight prunes reappear in the tray (not as a modal) across a refresh.
export async function resumeOps() {
  let jobs
  try {
    jobs = await apiGet('/api/ops')
  } catch {
    return
  }
  for (const j of jobs ?? []) attachJob(j.id, j.title, j.kind, false)
}

function closeStream(id) {
  const es = streams.get(id)
  if (es) {
    es.close()
    streams.delete(id)
  }
}

function refreshForKind(kind) {
  switch (kind) {
    case 'system-prune':
      refreshContainers()
      refreshImages()
      refreshVolumes()
      refreshNetworks()
      break
    case 'image-prune':
      refreshImages()
      break
    case 'volume-prune':
      refreshVolumes()
      break
  }
}
