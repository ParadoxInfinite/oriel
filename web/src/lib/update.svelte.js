// Lazy, run-once update check — the backend pings GitHub (cached) only when asked.
// Self-update (apply + restart) is offered only for service-managed installs.
import { apiGet, apiPost } from './api.js'

export const update = $state({
  checked: false,
  available: false,
  managed: false,
  current: '',
  latest: '',
  url: '',
  // self-update flow: '' | 'applying' | 'done' | 'restarting'
  phase: '',
  error: '',
})

let started = false

export async function checkUpdate() {
  if (started) return
  started = true
  try {
    const d = await apiGet('/api/update')
    update.current = d.current || ''
    update.latest = d.latest || ''
    update.url = d.url || ''
    update.available = !!d.updateAvailable
    update.managed = !!d.managed
  } catch {
    // Offline or GitHub unreachable — stay silent, no nag.
  } finally {
    update.checked = true
  }
}

// applyUpdate downloads + verifies + installs the new binary. Returns true when a
// restart is needed to finish.
export async function applyUpdate() {
  update.phase = 'applying'
  update.error = ''
  try {
    const d = await apiPost('/api/update/apply')
    if (d?.updated) {
      update.phase = 'done'
      return true
    }
    update.error = d?.message || 'Already up to date.'
    update.phase = ''
    return false
  } catch (e) {
    update.error = e.message
    update.phase = ''
    return false
  }
}

// restartService restarts the managed service so the new binary takes effect. The
// connection drops as it goes down and the live stream reconnects when it's back.
export async function restartService() {
  update.phase = 'restarting'
  update.error = ''
  try {
    await apiPost('/api/update/restart')
  } catch {
    // Expected — the server is going down mid-request.
  }
}
