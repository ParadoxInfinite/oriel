// Lazy, run-once update check — the backend pings GitHub (cached) only when asked.
// Self-update (apply + restart) is offered only for service-managed installs.
import { apiGet, apiPost } from './api.js'
import { self } from './self.svelte.js'
import { confirm } from './confirm.svelte.js'

export const update = $state({
  checked: false,
  checking: false, // a manual "check for updates" is in flight
  available: false,
  managed: false,
  packageManager: '', // e.g. 'homebrew' → update via the package manager, not in-app
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
  update.packageManager = d.packageManager || ''
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
    // The backend usually answers instantly from cache; hold "Checking…" for a
    // beat so the action reads as deliberate instead of a one-frame flicker.
    const [info] = await Promise.all([apiGet('/api/update?force=1'), new Promise((r) => setTimeout(r, 600))])
    applyInfo(info)
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

// restartService restarts the managed service, waits for the new binary to come
// back, then reloads so the whole UI reflects the new version. A non-OK reply
// (vs the expected dropped connection) is a real failure and is surfaced.
export async function restartService() {
  update.phase = 'restarting'
  update.error = ''
  const prevVersion = self.version
  try {
    // The endpoint replies 200 *then* restarts ~500ms later, so a 200 here means
    // "restart scheduled". A throw is a genuine error (e.g. not managed).
    await apiPost('/api/update/restart')
  } catch (e) {
    update.error = e.message || 'Restart failed'
    update.phase = ''
    return
  }
  // Poll until the server is back on a different version, then hard-reload.
  for (let i = 0; i < 40; i++) {
    await new Promise((r) => setTimeout(r, 1500))
    try {
      const d = await apiGet('/api/self')
      if (d?.version && d.version !== prevVersion) break
    } catch {
      /* still restarting */
    }
  }
  location.reload()
}

// canSelfUpdate is true only for a managed install not owned by a package
// manager — Homebrew updates via `brew upgrade`, not in-app. Others should be
// sent to the Updates panel for instructions.
export function canSelfUpdate() {
  return update.managed && !update.packageManager
}

// promptUpdate runs the confirm dialog + apply + optional restart. Shared by the
// Settings button and the siderail pill so both hit the same gate.
export async function promptUpdate() {
  const res = await confirm({
    title: 'Update Oriel?',
    message: `Download v${update.latest}, verify its checksum, and replace the binary. Oriel must restart to apply.`,
    confirmLabel: 'Update',
    danger: false,
    checkbox: 'Restart automatically when done',
    checked: true,
  })
  if (!res || !res.ok) return
  const ok = await applyUpdate()
  if (ok && res.checked) await restartService()
}
