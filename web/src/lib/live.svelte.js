import { sse, apiGet } from './api.js'
import { refreshContainers } from './containers.svelte.js'
import { refreshImages, refreshVolumes, refreshNetworks } from './resources.svelte.js'
import { refreshStacks } from './stacks.svelte.js'

// Latest per-container stats, keyed by id (for tables and cards).
export const stats = $state({ byId: {} })

// Rolling ~30-min aggregate series {t, cpu, mem, down}. Polled from the backend
// recorder (the source of truth for down-ticks and gaps) rather than rebuilt
// from the stats stream, so outages render correctly.
export const history = $state({ points: [] })

const refreshers = {
  container: [refreshContainers, refreshStacks], // stacks derive from containers
  image: [refreshImages],
  volume: [refreshVolumes],
  network: [refreshNetworks],
}

let esEvents
let esStats
let historyTimer
const timers = {}

async function refreshHistory() {
  try {
    const h = await apiGet('/api/history')
    history.points = h.map((p) => ({ t: p.t, cpu: p.cpu, mem: p.mem, down: !!p.down }))
  } catch {
    /* history is best-effort */
  }
}

function schedule(type) {
  const fns = refreshers[type]
  if (!fns) return
  clearTimeout(timers[type])
  timers[type] = setTimeout(() => fns.forEach((fn) => fn()), 250)
}

export async function startLive() {
  await refreshHistory()
  historyTimer = setInterval(refreshHistory, 4000)

  esEvents = sse('/api/events', ['event'], (_name, data) => schedule(data?.type))
  esStats = sse('/api/stats', ['stats'], (_name, data) => {
    const map = {}
    for (const s of data) map[s.id] = s
    stats.byId = map
  })
}

export function stopLive() {
  esEvents?.close()
  esStats?.close()
  clearInterval(historyTimer)
  for (const t of Object.values(timers)) clearTimeout(t)
}
