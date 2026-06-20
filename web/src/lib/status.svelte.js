import { apiGet } from './api.js'

// Shared reactive Colima status. Kept current by the live stream (see live.svelte);
// also the gate for the running/stopped zero-state across the whole UI.
export const status = $state({
  loading: true,
  running: false,
  data: null,
  error: null,
})

// applyStatus updates the store from a live-stream "status" event (a statusResult
// wrapper: { ok, status?, error? }). ok=false means the engine was unreachable.
export function applyStatus(res) {
  status.loading = false
  if (res?.ok && res.status) {
    status.data = res.status
    status.running = !!res.status.running
    status.error = null
  } else {
    status.error = res?.error || 'unreachable'
    status.running = false
  }
}

// refreshStatus does a one-off fetch (e.g. right after a start/stop); the live
// stream also keeps status current.
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
