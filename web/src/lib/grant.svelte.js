// The destructive-grant window: lets MCP / the assistant run destructive tools
// for a bounded time. The UI itself never needs it (human clicks carry consent);
// this panel just opens/closes the window for automated callers.
import { apiGet, apiPost, apiDelete } from './api.js'
import { toast } from './toast.svelte.js'

export const grant = $state({
  active: false,
  expiresAt: '', // RFC3339, empty when locked
  remainingSeconds: 0,
  loaded: false,
  busy: false,
})

function apply(d) {
  if (!d) return
  grant.active = !!d.active
  grant.expiresAt = d.expiresAt || ''
  grant.remainingSeconds = d.remainingSeconds || 0
  grant.loaded = true
}

export async function loadGrant() {
  apply(await apiGet('/api/grant'))
}

// openGrant starts (or extends) the window to `hours` from now.
export async function openGrant(hours) {
  grant.busy = true
  try {
    apply(await apiPost('/api/grant', { hours }))
  } finally {
    grant.busy = false
  }
}

// requestGrant opens the window and surfaces any failure as a toast — the
// shape both editions' "open window" buttons want.
export async function requestGrant(hours) {
  try {
    await openGrant(hours)
  } catch (e) {
    toast(e?.message || 'Could not open window', 'error')
  }
}

export async function lockGrant() {
  grant.busy = true
  try {
    apply(await apiDelete('/api/grant'))
  } finally {
    grant.busy = false
  }
}

// Format a seconds count as a short "3d 4h" / "5h 12m" / "12m" window remaining.
export function fmtRemaining(s) {
  if (s <= 0) return ''
  const d = Math.floor(s / 86400),
    h = Math.floor((s % 86400) / 3600),
    m = Math.floor((s % 3600) / 60)
  if (d) return `${d}d ${h}h`
  if (h) return `${h}h ${m}m`
  return `${m}m`
}
