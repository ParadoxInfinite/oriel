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

// A digest-pinned image is untagged but named by digest (its only "tag" is a
// repo@sha256:… ref) — e.g. compose images pinned by digest.
export function isPinnedImage(image) {
  return image?.tags?.length === 1 && image.tags[0].includes('@sha256:')
}

// suggestTag proposes a clean repo:tag for a pinned image, recovered from a using
// container's reference (which carries the tag, e.g. "valkey/valkey:8@sha256:…").
// Falls back to the repo portion of the digest ref, tag-less, for the user to finish.
export function suggestTag(image) {
  const dropDigest = (ref) => {
    const at = (ref || '').indexOf('@')
    return at === -1 ? ref || '' : ref.slice(0, at)
  }
  const ct = containersForImage(image?.id)[0]
  if (ct?.image && ct.image.includes(':')) return dropDigest(ct.image)
  return dropDigest(image?.tags?.[0])
}
