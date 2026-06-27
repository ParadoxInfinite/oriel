// The app's own backend footprint + build version, kept current by the live
// stream (see live.svelte).
export const self = $state({ version: '', os: '', rss: 0, goroutines: 0, heapAlloc: 0, maskEnv: 'all', maskLogs: 'sensitive', envReveal: 'local', sessionTTLMinutes: 0, loginFreeAttempts: 0, updateChannel: 'stable' })

// applySelf updates the store from a live-stream "self" event.
export function applySelf(d) {
  if (!d) return
  if (d.version) self.version = d.version
  if (d.os) self.os = d.os
  self.rss = d.rss
  self.goroutines = d.goroutines
  self.heapAlloc = d.heapAlloc
  if (d.maskEnv) self.maskEnv = d.maskEnv
  if (d.maskLogs) self.maskLogs = d.maskLogs
  if (d.envReveal) self.envReveal = d.envReveal
  if (d.sessionTTLMinutes != null) self.sessionTTLMinutes = d.sessionTTLMinutes
  if (d.loginFreeAttempts != null) self.loginFreeAttempts = d.loginFreeAttempts
  if (d.updateChannel) self.updateChannel = d.updateChannel
}
