import { sse } from './api.js'
import { refreshContainers } from './containers.svelte.js'
import { refreshImages, refreshVolumes, refreshNetworks } from './resources.svelte.js'
import { refreshStacks } from './stacks.svelte.js'
import { applyStatus } from './status.svelte.js'
import { applySelf } from './self.svelte.js'
import { applyOutages } from './outages.svelte.js'

// Latest per-container stats, keyed by id (for tables and cards).
export const stats = $state({ byId: {} })

// Live-stream health: `ok` flips false when the SSE stream drops, true once data
// flows again, so the UI can flag stale state instead of silently showing it.
export const connection = $state({ ok: true })

// Rolling ~30-min aggregate series {t, cpu, mem, down}. Seeded by the stream's
// "history" event on connect, then appended one "point" per second, no polling.
export const history = $state({ points: [] })
const HISTORY_CAP = 1800 // ~30 min at 1s resolution; matches the backend buffer

const refreshers = {
  container: [refreshContainers, refreshStacks], // stacks derive from containers
  image: [refreshImages],
  volume: [refreshVolumes],
  network: [refreshNetworks],
}

let esEvents
let esLive
const timers = {}

// schedule coalesces refetches: a burst of docker events for one type triggers a
// single list refresh shortly after the burst settles.
function schedule(type) {
  const fns = refreshers[type]
  if (!fns) return
  clearTimeout(timers[type])
  timers[type] = setTimeout(() => fns.forEach((fn) => fn()), 250)
}

function appendPoint(p) {
  if (!p) return
  const pts = history.points
  const last = pts[pts.length - 1]
  if (last && p.t <= last.t) return // dedup / out-of-order guard
  pts.push({ t: p.t, cpu: p.cpu, mem: p.mem, down: !!p.down })
  if (pts.length > HISTORY_CAP) pts.splice(0, pts.length - HISTORY_CAP)
}

export function startLive() {
  stopLive() // idempotent: never leak a previous pair of EventSources

  // Docker events drive list refreshes (change-triggered, already push-based).
  esEvents = sse('/api/events', ['event'], (_name, data) => schedule(data?.type))

  // One consolidated stream for everything periodic, no polling anywhere.
  esLive = sse(
    '/api/live',
    ['history', 'stats', 'point', 'status', 'self', 'outages'],
    (name, data) => {
      connection.ok = true // any frame means the stream is alive
      switch (name) {
      case 'history':
        history.points = (data || []).map((p) => ({ t: p.t, cpu: p.cpu, mem: p.mem, down: !!p.down }))
        break
      case 'point':
        appendPoint(data)
        break
      case 'stats': {
        const map = {}
        for (const s of data || []) map[s.id] = s
        stats.byId = map
        break
      }
      case 'status':
        applyStatus(data)
        break
      case 'self':
        applySelf(data)
        break
      case 'outages':
        applyOutages(data)
        break
      }
    },
    (readyState) => {
      connection.ok = readyState === 1 // OPEN; CONNECTING/CLOSED means dropped
    },
  )
}

export function stopLive() {
  esEvents?.close()
  esLive?.close()
  esEvents = esLive = undefined
  connection.ok = true
  for (const t of Object.values(timers)) clearTimeout(t)
}
