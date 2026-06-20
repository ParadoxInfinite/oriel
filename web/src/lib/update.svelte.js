// Lazy, run-once update check — the backend pings GitHub (cached) only when asked.
// Self-update (apply + restart) is offered only for service-managed installs.
import { apiGet, apiPost } from './api.js'

export const update = $state({
  checked: false,
  checking: false, // a manual "check for updates" is in flight
  available: false,
  managed: false,
  current: '',
  latest: '',
  url: '',
  // self-update flow: '' | 'applying' | 'done' | 'restarting'
  phase: '',
  error: '',
})

function applyInfo(d) {
  update.current = d.current || ''
  update.latest = d.latest || ''
  update.url = d.url || ''
  update.available = !!d.updateAvailable
  update.managed = !!d.managed
}

// The backend caches the GitHub result for a day; this interval just keeps a
// long-open tab in sync with that daily refresh (fresh mounts check immediately).
const RECHECK_MS = 3 * 60 * 60 * 1000
let timer = null

export async function checkUpdate() {
  try {
    applyInfo(await apiGet('/api/update'))
  } catch {
    // Offline or GitHub unreachable — stay silent, no nag.
  } finally {
    update.checked = true
  }
}

// checkNow is the manual "check for updates" action — it forces a fresh check,
// but the backend rate-limits forced checks to once an hour.
export async function checkNow() {
  if (update.checking) return
  update.checking = true
  try {
    applyInfo(await apiGet('/api/update?force=1'))
  } catch {
    /* offline — leave the last known state */
  } finally {
    update.checking = false
    update.checked = true
  }
}

// startUpdateChecks runs an immediate check on mount, then re-checks every few
// hours — skipping while a self-update is mid-flow so it can't clobber that state.
export function startUpdateChecks() {
  checkUpdate()
  timer = setInterval(() => {
    if (!update.phase) checkUpdate()
  }, RECHECK_MS)
}

export function stopUpdateChecks() {
  if (timer) clearInterval(timer)
  timer = null
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
