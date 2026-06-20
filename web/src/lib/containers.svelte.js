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

const stripDigest = (s) => (s || '').replace(/^sha256:/, '')

// Match by digest (not tag) so even a dangling <none> image resolves to the
// containers actually holding it.
export function containersForImage(imageId) {
  const id = stripDigest(imageId)
  if (!id) return []
  return containers.list.filter((c) => stripDigest(c.imageId) === id)
}
