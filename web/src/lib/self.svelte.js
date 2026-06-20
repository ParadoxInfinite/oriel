import { apiGet } from './api.js'

// The GUI's own backend footprint, polled at low frequency for the dashboard.
export const self = $state({ rss: 0, goroutines: 0, heapAlloc: 0 })

let timer
export async function refreshSelf() {
  try {
    const d = await apiGet('/api/self')
    self.rss = d.rss
    self.goroutines = d.goroutines
    self.heapAlloc = d.heapAlloc
  } catch {
    /* non-critical */
  }
}
export function startSelfPolling(ms = 10000) {
  refreshSelf()
  timer = setInterval(refreshSelf, ms)
}
export function stopSelfPolling() {
  clearInterval(timer)
}
