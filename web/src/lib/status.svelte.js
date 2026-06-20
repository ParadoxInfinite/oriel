import { apiGet } from './api.js'

// Shared reactive Colima status. Polled while the app is open; also the gate
// for the running/stopped zero-state across the whole UI.
export const status = $state({
  loading: true,
  running: false,
  data: null,
  error: null,
})

export async function refreshStatus() {
  try {
    const d = await apiGet('/api/colima/status')
    status.data = d
    status.running = d.running
    status.error = null
  } catch (e) {
    status.error = e.message
    status.running = false
  } finally {
    status.loading = false
  }
}

let timer
export function startStatusPolling(ms = 10000) {
  refreshStatus()
  timer = setInterval(refreshStatus, ms)
}
export function stopStatusPolling() {
  clearInterval(timer)
}
