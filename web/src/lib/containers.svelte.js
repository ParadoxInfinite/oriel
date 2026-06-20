import { apiGet } from './api.js'

// Shared container list. Refreshed on demand now; M4 wires it to live events.
export const containers = $state({ list: [], loading: false, error: null })

export async function refreshContainers() {
  containers.loading = true
  try {
    containers.list = await apiGet('/api/containers')
    containers.error = null
  } catch (e) {
    containers.error = e.message
  } finally {
    containers.loading = false
  }
}
