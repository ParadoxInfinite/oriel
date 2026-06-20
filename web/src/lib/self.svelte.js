// The GUI's own backend footprint, kept current by the live stream (see live.svelte).
export const self = $state({ rss: 0, goroutines: 0, heapAlloc: 0 })

// applySelf updates the store from a live-stream "self" event.
export function applySelf(d) {
  if (!d) return
  self.rss = d.rss
  self.goroutines = d.goroutines
  self.heapAlloc = d.heapAlloc
}
