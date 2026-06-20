// Persisted downtime log (retained ~30 days on the backend), kept current by the
// live stream (see live.svelte).
export const outages = $state({ list: [] })

// applyOutages updates the store from a live-stream "outages" event.
export function applyOutages(list) {
  outages.list = Array.isArray(list) ? list : []
}
