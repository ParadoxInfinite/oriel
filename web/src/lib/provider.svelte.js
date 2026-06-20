import { apiGet, apiPost } from './api.js'

// NL provider state: whether a resolver is configured (gates the palette's
// free-text mode) and the current endpoint URL (shown/edited in Settings → AI).
export const provider = $state({ enabled: false, url: '' })

export async function checkProvider() {
  try {
    const r = await apiGet('/api/provider')
    provider.enabled = !!r.enabled
    provider.url = r.url || ''
  } catch {
    provider.enabled = false
    provider.url = ''
  }
}

// Swap the resolver endpoint at runtime; '' returns the seam to dormant.
export async function setProvider(url) {
  const r = await apiPost('/api/provider', { url })
  provider.enabled = !!r.enabled
  provider.url = r.url || ''
  return r
}

export function resolveText(text) {
  return apiPost('/api/resolve', { text })
}
