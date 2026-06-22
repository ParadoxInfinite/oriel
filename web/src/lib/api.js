// Thin fetch wrappers + an SSE-over-POST reader. One place for all backend I/O.

// In the GitHub Pages demo build, every call is served by an in-memory mock
// instead of the network. __ORIEL_DEMO__ is a build-time boolean literal (see
// vite.config.js define), so in prod it's `false` — the branches below fold away
// and Rollup drops this import, keeping the demo entirely out of the real binary.
import * as demo from './demo/index.js'
const DEMO = __ORIEL_DEMO__

// Prefix backend paths with the SPA base so one build works at the host root or
// behind a reverse-proxy subpath. Vite bakes a sentinel base into the bundle;
// the Go server rewrites it to ORIEL_BASE_PATH (or "/") when serving, so BASE_URL
// is "/" at root or e.g. "/oriel/" under a subpath.
const BASE = import.meta.env.BASE_URL.replace(/\/$/, '')
const url = (path) => BASE + path

async function parseError(res) {
  try {
    const body = await res.json()
    return body.error || res.statusText
  } catch {
    return res.statusText
  }
}

export async function apiGet(path) {
  if (DEMO) return demo.demoGet(path)
  const res = await fetch(url(path))
  if (!res.ok) throw new Error(await parseError(res))
  return res.json()
}

export async function apiPost(path, body) {
  if (DEMO) return demo.demoPost(path, body)
  const res = await fetch(url(path), {
    method: 'POST',
    headers: body ? { 'Content-Type': 'application/json' } : {},
    body: body ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) throw new Error(await parseError(res))
  return res.json().catch(() => null)
}

export async function apiPut(path, body) {
  if (DEMO) return demo.demoPut(path, body)
  const res = await fetch(url(path), {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(await parseError(res))
  return res.json().catch(() => null)
}

export async function apiDelete(path) {
  if (DEMO) return demo.demoDelete(path)
  const res = await fetch(url(path), { method: 'DELETE' })
  if (!res.ok) throw new Error(await parseError(res))
  return res.json().catch(() => null)
}

// streamPost parses a text/event-stream POST response, one onEvent(name, data)
// per frame. POST (not EventSource) because the action mutates state.
export async function streamPost(path, { onEvent, signal } = {}) {
  if (DEMO) return demo.demoStreamPost(path, { onEvent, signal })
  const res = await fetch(url(path), { method: 'POST', signal })
  if (!res.ok) throw new Error(await parseError(res))
  if (!res.body) throw new Error('empty response stream')
  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buf = ''
  for (;;) {
    const { value, done } = await reader.read()
    if (done) break
    buf += decoder.decode(value, { stream: true })
    let idx
    while ((idx = buf.indexOf('\n\n')) >= 0) {
      const frame = buf.slice(0, idx)
      buf = buf.slice(idx + 2)
      let event = 'message'
      let data = ''
      for (const line of frame.split('\n')) {
        if (line.startsWith('event:')) event = line.slice(6).trim()
        else if (line.startsWith('data:')) data += line.slice(5).trim()
      }
      if (data) {
        try {
          onEvent?.(event, JSON.parse(data))
        } catch {
          /* ignore malformed frame */
        }
      }
    }
  }
}

// sse opens an EventSource, listening for the named events. onEvent receives
// (name, parsedData). Returns the EventSource so callers can close() it.
export function sse(path, events, onEvent) {
  if (DEMO) return demo.demoSse(path, events, onEvent)
  const es = new EventSource(url(path))
  for (const name of events) {
    es.addEventListener(name, (e) => {
      let data = e.data
      try {
        data = JSON.parse(e.data)
      } catch {
        /* leave as raw string */
      }
      onEvent(name, data)
    })
  }
  return es
}
