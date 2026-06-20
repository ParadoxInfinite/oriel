import { streamPost, apiGet, apiPost, sse } from './api.js'
import { refreshContainers } from './containers.svelte.js'
import { refreshImages, refreshVolumes, refreshNetworks } from './resources.svelte.js'

// Generic operation-overlay state, shared by long-running actions that emit live
// progress. Two flavours feed the same overlay:
//   • request-tied (runOp): colima lifecycle, compose — stream lives with the POST.
//   • background jobs (attachJob): prune — run server-side, survive a refresh, and
//     can be cancelled. jobId is set only for the latter (drives the Cancel button).
// One overlay at a time.
export const op = $state({
  active: false,
  title: null,
  lines: [],
  error: null,
  done: false,
  jobId: null,
  cancelling: false,
})

let es = null // the EventSource for the attached background job, if any

export async function runOp(title, path, onDone) {
  if (op.active) return
  resetState(title)
  op.jobId = null
  try {
    await streamPost(path, {
      onEvent: (name, data) => {
        if (name === 'line') op.lines = [...op.lines, data.line]
        else if (name === 'done') {
          op.done = true
          if (!data.ok) op.error = data.error
        }
      },
    })
  } catch (e) {
    op.error = e.message
    op.done = true
  } finally {
    op.active = false
    onDone?.()
  }
}

// attachJob streams a server-side background job into the overlay. The stream
// sends a "snapshot" of progress so far (so reconnect/refresh catches up without
// duplicates), then live "line" events, then "done". Re-callable to re-attach.
export function attachJob(id, title, kind) {
  closeStream()
  resetState(title)
  op.jobId = id
  es = sse(`/api/ops/${id}/stream`, ['snapshot', 'line', 'done'], (name, data) => {
    if (name === 'snapshot') op.lines = data.lines || []
    else if (name === 'line') op.lines = [...op.lines, data.line]
    else if (name === 'done') {
      op.done = true
      op.active = false
      op.cancelling = false
      if (!data.ok) op.error = data.error || 'cancelled'
      closeStream()
      refreshForKind(kind)
    }
  })
}

// startJob kicks off a background op (POST returns its id), then attaches to it.
async function startJob(path, body, title, kind) {
  if (op.active) return
  let res
  try {
    res = await apiPost(path, body)
  } catch (e) {
    resetState(title)
    op.error = e.message
    op.done = true
    op.active = false
    return
  }
  if (res?.id) attachJob(res.id, title, kind)
}

export function startSystemPrune(includeVolumes) {
  return startJob(`/api/ops/system-prune?volumes=${includeVolumes ? 'true' : 'false'}`, null, 'Reclaiming disk space', 'system-prune')
}

// items: [{ id, size }]. The server reports how much was reclaimed.
export function startImagePrune(items) {
  return startJob('/api/ops/image-prune', { items }, `Pruning ${items.length} image${items.length === 1 ? '' : 's'}`, 'image-prune')
}
export function startVolumePrune(items) {
  return startJob('/api/ops/volume-prune', { items }, `Pruning ${items.length} volume${items.length === 1 ? '' : 's'}`, 'volume-prune')
}

// cancelOp asks the server to stop the attached job; the "done" event follows.
export async function cancelOp() {
  if (!op.jobId || op.done) return
  op.cancelling = true
  try {
    await apiPost(`/api/ops/${op.jobId}/cancel`, null)
  } catch {
    op.cancelling = false
  }
}

// resumeOps re-attaches to any still-running background job after a page load,
// so an in-flight prune keeps showing its progress across a refresh.
export async function resumeOps() {
  if (op.active || op.title) return
  let jobs
  try {
    jobs = await apiGet('/api/ops')
  } catch {
    return
  }
  const j = jobs?.[0]
  if (j) attachJob(j.id, j.title, j.kind)
}

export function dismissOp() {
  closeStream()
  op.title = null
  op.lines = []
  op.error = null
  op.done = false
  op.jobId = null
  op.cancelling = false
}

function resetState(title) {
  op.active = true
  op.title = title
  op.lines = []
  op.error = null
  op.done = false
  op.cancelling = false
}

function closeStream() {
  if (es) {
    es.close()
    es = null
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
