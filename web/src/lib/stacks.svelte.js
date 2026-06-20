import { apiGet } from './api.js'

// Compose stacks, derived from container labels server-side. Refreshed on demand
// and whenever container events fire (see live.svelte.js).
export const stacks = $state({ list: [], loading: false, error: null })

export async function refreshStacks() {
  stacks.loading = true
  try {
    stacks.list = await apiGet('/api/stacks')
    stacks.error = null
  } catch (e) {
    stacks.error = e.message
  } finally {
    stacks.loading = false
  }
}
