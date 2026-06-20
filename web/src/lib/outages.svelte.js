import { apiGet } from './api.js'

// Persisted downtime log (retained ~30 days on the backend).
export const outages = $state({ list: [] })

let timer

export async function refreshOutages() {
  try {
    const list = await apiGet('/api/outages')
    outages.list = Array.isArray(list) ? list : []
  } catch {
    /* best-effort */
  }
}

export function startOutagesPolling() {
  refreshOutages()
  timer = setInterval(refreshOutages, 60000)
}

export function stopOutagesPolling() {
  clearInterval(timer)
}
