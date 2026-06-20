// The app's own backend footprint + build version, kept current by the live
// stream (see live.svelte).
export const self = $state({ version: '', rss: 0, goroutines: 0, heapAlloc: 0 })

// applySelf updates the store from a live-stream "self" event.
export function applySelf(d) {
  if (!d) return
  if (d.version) self.version = d.version
  self.rss = d.rss
  self.goroutines = d.goroutines
  self.heapAlloc = d.heapAlloc
}
