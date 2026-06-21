import { sse, apiGet } from './api.js'

// Headless logs controller shared by both editions. Seeds with the last FOLLOW_CAP
// lines and tails live; older lines load on demand. Memory is bounded: while
// following, the buffer trims back to FOLLOW_CAP, dropping lazily-loaded history.
const FOLLOW_CAP = 100 // lines kept while tailing live ("100 latest")
const BATCH = 100 // older lines fetched per lazy load
const HARD_MAX = 1000 // ceiling while scrolled up, so history can't grow unbounded

export class LogsController {
  lines = $state([])
  loadingOlder = $state(false)
  noMore = $state(false) // reached the start of available history
  error = $state(null)
  connected = $state(false) // SSE stream is open (vs still connecting)

  #es = null
  #id = null
  #following = true

  start(id) {
    this.stop()
    this.#id = id
    this.lines = []
    this.noMore = false
    this.error = null
    this.connected = false
    this.#following = true
    this.#es = sse(`/api/containers/${id}/logs`, ['log', 'error'], (name, data) => {
      if (name === 'error') {
        this.error = data.error || 'stream error'
        return
      }
      this.lines.push({ stream: data.stream, ts: data.ts, line: data.line })
      // While following, keep only the freshest FOLLOW_CAP lines in memory.
      if (this.#following && this.lines.length > FOLLOW_CAP) {
        this.lines.splice(0, this.lines.length - FOLLOW_CAP)
      }
    })
    // onopen fires once the stream is established — lets the UI tell an empty
    // (but live) stream apart from one that's still connecting.
    this.#es.onopen = () => (this.connected = true)
  }

  // setFollowing(true) when scrolled to the live tail: drop the older history we
  // lazily loaded so memory returns to FOLLOW_CAP. Older lines can be reloaded.
  setFollowing(on) {
    this.#following = on
    if (on && this.lines.length > FOLLOW_CAP) {
      this.lines.splice(0, this.lines.length - FOLLOW_CAP)
      this.noMore = false
    }
  }

  // loadOlder prepends one batch of lines before the current earliest. Returns the
  // count added (0 if none / at a cap) so the caller can preserve scroll position.
  async loadOlder() {
    if (this.loadingOlder || this.noMore || this.lines.length >= HARD_MAX) return 0
    const earliest = this.lines.find((l) => l.ts)?.ts
    if (!earliest) return 0
    this.loadingOlder = true
    try {
      const older = await apiGet(`/api/containers/${this.#id}/logs/before?before=${encodeURIComponent(earliest)}&limit=${BATCH}`)
      // Filter to strictly-older (RFC3339Nano sorts lexically) to dedupe the cursor boundary.
      const fresh = (older || []).filter((l) => !l.ts || l.ts < earliest)
      if (fresh.length === 0) {
        this.noMore = true
        return 0
      }
      this.lines.unshift(...fresh)
      if (fresh.length < BATCH) this.noMore = true
      return fresh.length
    } catch (e) {
      this.error = e.message
      return 0
    } finally {
      this.loadingOlder = false
    }
  }

  stop() {
    this.#es?.close()
    this.#es = null
    this.connected = false
  }
}
