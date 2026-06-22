// The destructive-grant window: lets MCP / the assistant run destructive tools
// for a bounded time. The UI itself never needs it (human clicks carry consent);
// this panel just opens/closes the window for automated callers.
import { apiGet, apiPost, apiDelete } from './api.js'

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

export async function lockGrant() {
  grant.busy = true
  try {
    apply(await apiDelete('/api/grant'))
  } finally {
    grant.busy = false
  }
}
