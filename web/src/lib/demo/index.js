// In-memory mock backend for the GitHub Pages "try it live" build. Satisfies the
// api.js contract so the whole UI runs with zero network. Guarded behind VITE_DEMO
// in api.js (so it tree-shakes out of the real binary); `db` re-seeds on every
// page load, so a refresh resets all demo mutations — by design.
import * as seed from './seed.js'

const clone = (x) => (x == null ? x : JSON.parse(JSON.stringify(x)))
const delay = (ms = 90) => new Promise((r) => setTimeout(r, ms))
const rnd = (a, b) => a + Math.random() * (b - a)
const GiB = 1024 * 1024 * 1024

// ── in-memory store (reset on refresh) ──────────────────────────────────────
const db = {
  containers: seed.makeContainers(),
  images: seed.makeImages(),
  volumes: seed.makeVolumes(),
  networks: seed.makeNetworks(),
  colima: clone(seed.colimaStatus),
  provider: clone(seed.provider),
  remoteHosts: [],
  discovery: clone(seed.discovery),
}

let seq = 0
const uid = (p) => `${p}-${++seq}`

// Dangling = untagged. Docker (and the real API) report these as a single
// '<none>' tag, so match both that and a truly empty tag list.
const isDangling = (i) => !(i.tags || []).length || i.tags[0] === '<none>'

// The /api/events + /api/live SSE streams register their callbacks here, so
// store mutations can push refresh-events and status changes back to the UI.
let eventsCb = null
let liveCb = null
// Real wall clock — live points must fall in the chart's Date.now()-30min window.
let clock = Date.now()

const emit = (type) => eventsCb && eventsCb('event', { type })
const emitStatus = () => liveCb && liveCb('status', { ok: true, status: clone(db.colima) })

const running = () => db.containers.filter((c) => c.state === 'running')
const statsList = () =>
  running().map((c) => ({
    id: c.id,
    name: c.name,
    cpu: db.colima.running ? +rnd(0.3, 22).toFixed(2) : 0,
    mem: Math.round(rnd(20, 420)) * 1024 * 1024,
  }))
// One system-level sample on the same scale as makeHistory, so the chart's live
// edge is continuous with the seeded history (not a per-container sum that spikes).
const sysSample = () => ({ cpu: +rnd(6, 20).toFixed(1), mem: Math.round(rnd(2.0, 2.4) * GiB) })

// ── GET ─────────────────────────────────────────────────────────────────────
export async function demoGet(path) {
  await delay()
  const [p, qs] = path.split('?')
  const q = new URLSearchParams(qs || '')

  switch (p) {
    case '/api/containers': return clone(db.containers)
    case '/api/images': return clone(db.images)
    case '/api/volumes': return clone(db.volumes)
    case '/api/networks': return clone(db.networks)
    case '/api/stacks': return seed.makeStacks(db.containers)
    case '/api/system/df': return seed.makeDf(db.containers, db.images, db.volumes)
    case '/api/colima/status': return clone(db.colima)
    case '/api/self': return clone(seed.self)
    case '/api/update': return clone(seed.update)
    case '/api/provider': return clone(db.provider)
    case '/api/remote': return { hosts: clone(db.remoteHosts) }
    case '/api/themes': return clone(seed.themes)
    case '/api/discovery': return clone(db.discovery)
    case '/api/discovery/scan': return scanResult()
    case '/api/ops': return [] // no jobs survive the demo's "refresh"
    case '/api/volumes/prune/preview': return prunePreview()
    case '/api/images/search': return searchResults(q.get('q') || '')
    case '/api/images/tags': return tagResults(q.get('repo') || '')
    case '/api/fs/list': return fsList(q.get('path') || '')
  }
  if (p.endsWith('/inspect')) return inspectFor(p.split('/')[3])
  if (p.includes('/logs/before')) return [] // demo carries no deep log history
  return null
}

function inspectFor(id) {
  const c = db.containers.find((x) => x.id === id)
  if (!c) return {}
  const exit = Number(c.status.match(/\((\d+)\)/)?.[1] ?? 0)
  return {
    image: c.image,
    imageId: c.imageId,
    command: 'docker-entrypoint.sh',
    workingDir: '/app',
    restartPolicy: c.project ? 'unless-stopped' : 'no',
    startedAt: new Date((1782060000 - 172800) * 1000).toISOString(),
    running: c.state === 'running',
    exitCode: exit,
    health: c.status.includes('healthy') ? 'healthy' : '',
    networks: [{ name: c.project ? `${c.project}_default` : 'bridge', ipAddress: `172.18.0.${(seq % 240) + 2}` }],
    mounts: c.project ? [{ type: 'volume', destination: '/var/lib/data', rw: true, name: `${c.project}_data`, source: '' }] : [],
    env: ['PATH=/usr/local/sbin:/usr/local/bin:/usr/local/sbin', 'NODE_ENV=production', `HOSTNAME=${c.name}`, 'TZ=UTC'],
  }
}

function scanResult() {
  return {
    stacks: [
      { name: 'analytics', alias: '', dir: '/Users/dev/projects/analytics', composeFile: 'docker-compose.yml', services: ['clickhouse', 'ingest', 'dashboard'] },
      { name: 'mail-dev', alias: '', dir: '/Users/dev/projects/mail-dev', composeFile: 'compose.yaml', services: ['mailpit'] },
    ],
    roots: [{ id: 'r-demo', path: '/Users/dev/projects', found: 4, error: '' }],
    hidden: 0,
  }
}

function prunePreview() {
  const orphan = db.volumes[db.volumes.length - 1]
  return orphan ? [{ name: orphan.name, size: 0.3 * GiB }] : []
}

function searchResults(term) {
  const base = (term || 'app').split('/').pop() || 'app'
  const r = (name, official, stars, description) => ({ name, official, stars, description })
  return [
    r(base, true, 9200, `Official build of ${base}`),
    r(`bitnami/${base}`, false, 1400, `Bitnami packaged ${base}`),
    r(`library/${base}`, false, 880, `${base}, community maintained`),
    r(`${base}-alpine`, false, 210, `Minimal ${base} on Alpine`),
  ]
}

function tagResults(repo) {
  const v = ['latest', '1', '1.2', '1.27', '1.27.1', '16-alpine', '7-alpine', 'stable', 'bookworm', 'slim']
  return repo ? v : v.slice(0, 4)
}

function fsList(path) {
  const base = path || '/Users/dev'
  const dirs = ['projects', 'work', 'sandbox', 'archive']
  return {
    path: base,
    parent: base.includes('/') ? base.slice(0, base.lastIndexOf('/')) || '/' : '/',
    entries: dirs.map((name) => ({ name, path: `${base}/${name}`, dir: true })),
  }
}

// ── POST ────────────────────────────────────────────────────────────────────
const jobs = new Map() // jobId → { kind, items, cancelled }

export async function demoPost(path, body) {
  await delay()
  const [p, qs] = path.split('?')
  const q = new URLSearchParams(qs || '')

  if (p === '/api/invoke') return { result: invokeTool(body?.tool, body?.args || {}) }
  if (p === '/api/provider') {
    db.provider = { enabled: !!body?.url, url: body?.url || '' }
    return clone(db.provider)
  }
  if (p === '/api/ops/system-prune') return startJob('system-prune', q)
  if (p === '/api/ops/image-prune') return startJob('image-prune', null, body?.items)
  if (p === '/api/ops/volume-prune') return startJob('volume-prune', null, body?.items)
  if (p.startsWith('/api/ops/') && p.endsWith('/cancel')) {
    const j = jobs.get(p.split('/')[3])
    if (j) j.cancelled = true
    return null
  }
  // /api/resolve, /api/fs/open, /api/update/* — not reachable in the demo
  // (AI resolver + self-update are disabled); answer benignly.
  return null
}

function invokeTool(tool, args) {
  const c = (id) => db.containers.find((x) => x.id === id)
  switch (tool) {
    case 'container.start': { const x = c(args.id); if (x) { x.state = 'running'; x.status = 'Up 1 second' } return done('container') }
    case 'container.restart': { const x = c(args.id); if (x) { x.state = 'running'; x.status = 'Up 1 second' } return done('container') }
    case 'container.stop': { const x = c(args.id); if (x) { x.state = 'exited'; x.status = 'Exited (0) 1 second ago' } return done('container') }
    case 'container.remove': db.containers = db.containers.filter((x) => x.id !== args.id); return done('container')
    case 'image.remove': db.images = db.images.filter((i) => i.id !== args.id && !(i.tags || []).includes(args.id)); return done('image')
    case 'image.tag': { const i = db.images.find((x) => x.id === args.id); if (i && args.tag) i.tags = [...new Set([...(i.tags || []), args.tag])]; return done('image', args.tag || true) }
    case 'image.prune': db.images = db.images.filter((i) => !isDangling(i)); return done('image')
    case 'volume.remove': db.volumes = db.volumes.filter((v) => v.name !== args.name); return done('volume')
    case 'volume.prune': db.volumes = db.volumes.slice(0, -1); return done('volume')
    case 'network.remove': db.networks = db.networks.filter((nw) => nw.id !== args.id && !['bridge', 'host', 'none'].includes(nw.name)); return done('network')
    default: return true
  }
}
function done(type, ret = true) { emit(type); return ret }

function startJob(kind, q, items) {
  const id = uid('job')
  jobs.set(id, { kind, items: items || [], cancelled: false })
  return { id }
}

// ── PUT ─────────────────────────────────────────────────────────────────────
export async function demoPut(path, body) {
  await delay()
  if (path === '/api/discovery') { db.discovery = body || db.discovery; return clone(db.discovery) }
  if (path === '/api/remote') { db.remoteHosts = body?.hosts || []; return { hosts: clone(db.remoteHosts) } }
  return null
}

// ── streamPost (request-tied SSE-over-POST) ─────────────────────────────────
export async function demoStreamPost(path, { onEvent } = {}) {
  const [p, qs] = path.split('?')
  const q = new URLSearchParams(qs || '')
  if (p.startsWith('/api/colima/')) return streamColima(p.split('/')[3], onEvent)
  if (p === '/api/stacks/up') return streamDeploy(q, onEvent)
  if (p.startsWith('/api/stacks/')) { const [, , , name, action] = p.split('/'); return streamStack(name, action, onEvent) }
  if (p === '/api/images/pull') return streamPull(q.get('ref') || '', onEvent)
}

const line = (onEvent, s) => onEvent?.('line', { line: s })

async function streamColima(action, onEvent) {
  const lines = {
    start: ['Starting Colima…', 'Provisioning VM (aarch64 · 4 CPU · 8GiB)…', 'Starting docker runtime…', 'Colima is running.'],
    stop: ['Stopping Colima…', 'Stopping docker runtime…', 'Colima stopped.'],
    restart: ['Stopping Colima…', 'Starting Colima…', 'Starting docker runtime…', 'Colima is running.'],
  }[action] || ['Working…', 'Done.']
  for (const l of lines) { line(onEvent, l); await delay(550) }
  db.colima.running = action !== 'stop'
  emitStatus()
  onEvent?.('done', { ok: true })
}

async function streamStack(name, action, onEvent) {
  const cs = db.containers.filter((c) => c.project === name)
  const verb = action === 'down' ? 'Removing' : action === 'stop' ? 'Stopping' : 'Starting'
  for (const c of cs) { line(onEvent, `${verb} ${c.name}…`); await delay(280) }
  if (action === 'down') db.containers = db.containers.filter((c) => c.project !== name)
  else {
    const up = action === 'up' || action === 'restart'
    for (const c of cs) { c.state = up ? 'running' : 'exited'; c.status = up ? 'Up 1 second' : 'Exited (0) 1 second ago' }
  }
  emit('container')
  onEvent?.('done', { ok: true })
}

async function streamDeploy(q, onEvent) {
  const name = q.get('alias') || q.get('name') || (q.get('dir') || '').split('/').pop() || 'project'
  for (const l of [`Pulling images for ${name}…`, 'Creating network…', 'Creating containers…', `${name} deployed.`]) { line(onEvent, l); await delay(480) }
  emit('container')
  onEvent?.('done', { ok: true })
}

async function streamPull(ref, onEvent) {
  const layers = ['a1b2c3d4', 'e5f60718', '293a4b5c', '6d7e8f90']
  onEvent?.('progress', { status: 'Pulling from library', id: ref })
  await delay(300)
  for (const id of layers) { onEvent?.('progress', { status: 'Downloading', id }); await delay(320) }
  for (const id of layers) { onEvent?.('progress', { status: 'Extract', id }); await delay(220) }
  onEvent?.('progress', { status: 'Pull complete', id: '' })
  const tag = ref.includes(':') ? ref : `${ref || 'image'}:latest`
  if (!db.images.some((i) => (i.tags || []).includes(tag))) {
    db.images.unshift({ id: 'sha256:' + Math.random().toString(16).slice(2).padEnd(64, '0').slice(0, 64), tags: [tag], size: rnd(0.05, 0.6) * GiB, created: Math.floor(clock / 1000), containers: 0 })
  }
  emit('image')
  onEvent?.('done', { ok: true })
}

// Fakes the bits of EventSource callers touch (.onopen, .close()); streams driven on timers.
export function demoSse(path, _events, onEvent) {
  const [p] = path.split('?')
  const es = { onopen: null, _closed: false, _timer: null, close() { this._closed = true; if (this._timer) clearInterval(this._timer) } }
  setTimeout(() => { if (!es._closed) es.onopen?.() }, 30)

  if (p === '/api/events') {
    eventsCb = onEvent
  } else if (p === '/api/live') {
    liveCb = onEvent
    setTimeout(() => {
      if (es._closed) return
      onEvent('history', seed.makeHistory())
      onEvent('stats', statsList())
      onEvent('status', { ok: true, status: clone(db.colima) })
      onEvent('self', clone(seed.self))
      onEvent('outages', seed.makeOutages())
    }, 40)
    es._timer = setInterval(() => {
      if (es._closed) return
      clock += 1000
      const a = db.colima.running ? sysSample() : { cpu: 0, mem: 0 }
      onEvent('point', { t: clock, cpu: a.cpu, mem: a.mem, down: !db.colima.running })
      onEvent('stats', statsList())
    }, 1000)
  } else if (p.endsWith('/logs')) {
    // Exited containers stay quiet, so the UI's "no logs" empty state is reachable.
    const c = db.containers.find((x) => x.id === p.split('/')[3])
    driveLogs(es, onEvent, !c || c.state !== 'running')
  } else if (p.includes('/ops/') && p.endsWith('/stream')) {
    driveJob(p.split('/')[3], es, onEvent)
  }
  return es
}

function driveLogs(es, onEvent, quiet) {
  if (quiet) return // connected but no lines → the empty-state message shows
  let i = 0
  const tick = () => {
    if (es._closed) return
    const [stream, l] = seed.SAMPLE_LOGS[i % seed.SAMPLE_LOGS.length]
    onEvent('log', { stream, ts: new Date(clock + i * 7).toISOString(), line: l })
    i++
  }
  setTimeout(() => { for (let k = 0; k < 8; k++) tick() }, 60)
  es._timer = setInterval(tick, 2600)
}

async function driveJob(id, es, onEvent) {
  const job = jobs.get(id) || { kind: 'system-prune', items: [], cancelled: false }
  await delay(40)
  if (es._closed) return
  onEvent('snapshot', { lines: [], cur: 0, total: 100 })
  const steps = jobSteps(job)
  let cur = 0
  for (const s of steps) {
    if (es._closed) return
    if (job.cancelled) { onEvent('done', { ok: false, error: 'cancelled' }); jobs.delete(id); return }
    onEvent('line', { line: s })
    cur = Math.min(95, cur + Math.ceil(100 / (steps.length + 1)))
    onEvent('progress', { cur, total: 100 })
    await delay(460)
  }
  if (es._closed) return
  applyJob(job)
  onEvent('line', { line: jobSummary(job) })
  onEvent('progress', { cur: 100, total: 100 })
  onEvent('done', { ok: true })
  jobs.delete(id)
}

function jobSteps(job) {
  if (job.kind === 'image-prune') return ['Deleting untagged images…', 'Reclaiming layers…']
  if (job.kind === 'volume-prune') return ['Removing unused volumes…']
  return ['Removing stopped containers…', 'Deleting dangling images…', 'Pruning build cache…', 'Removing unused networks…']
}
function jobSummary(job) {
  if (job.kind === 'image-prune') return 'Reclaimed 0.19GB from images.'
  if (job.kind === 'volume-prune') return 'Reclaimed 0.3GB from volumes.'
  return 'Reclaimed 1.99GB total.'
}
function applyJob(job) {
  if (job.kind === 'image-prune') {
    const ids = new Set((job.items || []).map((x) => x.id))
    db.images = ids.size ? db.images.filter((i) => !ids.has(i.id)) : db.images.filter((i) => !isDangling(i))
    emit('image')
  } else if (job.kind === 'volume-prune') {
    const names = new Set((job.items || []).map((x) => x.name))
    db.volumes = names.size ? db.volumes.filter((v) => !names.has(v.name)) : db.volumes.slice(0, -1)
    emit('volume')
  } else {
    db.containers = db.containers.filter((c) => c.state === 'running')
    db.images = db.images.filter((i) => !isDangling(i))
    db.volumes = db.volumes.slice(0, -1)
    emit('container'); emit('image'); emit('volume'); emit('network')
  }
}
