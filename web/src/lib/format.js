// Human-readable formatting helpers.

const UNITS = ['B', 'KiB', 'MiB', 'GiB', 'TiB']

export function bytes(n) {
  if (!n || n < 0) return '0 B'
  let i = 0
  let v = n
  while (v >= 1024 && i < UNITS.length - 1) {
    v /= 1024
    i++
  }
  const digits = v >= 100 || i === 0 ? 0 : 1
  return `${v.toFixed(digits)} ${UNITS[i]}`
}

// Unambiguous date/time — "DD MMM YYYY" so no one mistakes day for month.
const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
const pad = (n) => String(n).padStart(2, '0')

export function timeOnly(ms) {
  const d = new Date(ms)
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}
export function dateTime(ms) {
  const d = new Date(ms)
  return `${d.getDate()} ${MONTHS[d.getMonth()]} ${d.getFullYear()}, ${timeOnly(ms)}`
}

// Compact duration from milliseconds: "45s", "3m 12s", "1h 4m".
export function duration(ms) {
  if (!ms || ms < 0) return '0s'
  const s = Math.round(ms / 1000)
  if (s < 60) return `${s}s`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ${s % 60}s`
  const h = Math.floor(m / 60)
  return `${h}h ${m % 60}m`
}

export function relativeTime(unixSeconds) {
  if (!unixSeconds) return '—'
  const diff = Date.now() / 1000 - unixSeconds
  if (diff < 60) return 'just now'
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
  return `${Math.floor(diff / 86400)}d ago`
}
