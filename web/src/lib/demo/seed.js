// Synthetic, self-contained data for the live demo (GitHub Pages). NOTHING here
// comes from a real Docker host — it's hand-authored to look like a believable
// dev machine. A page refresh re-imports this module, resetting any mutations.

const hex = (seed) => {
  // Deterministic 64-char hex id from a short seed (no Math.random at import).
  let h = ''
  let x = 0
  for (let i = 0; i < seed.length; i++) x = (x * 31 + seed.charCodeAt(i)) >>> 0
  for (let i = 0; i < 64; i++) {
    x = (x * 1103515245 + 12345) & 0x7fffffff
    h += ((x >> 8) & 0xf).toString(16)
  }
  return h
}
const sha = (seed) => 'sha256:' + hex(seed)
const now = 1782060000 // fixed "now" (unix sec) so timestamps read sensibly
const ago = (sec) => now - sec
const GiB = 1024 * 1024 * 1024

// ---- containers (a web-app stack + monitoring stack + a couple standalone) ----
export function makeContainers() {
  return [
    { id: hex('web'), name: 'web-app-nginx-1', image: 'nginx:1.27-alpine', imageId: sha('nginx'), state: 'running', status: 'Up 2 days', created: ago(172800), ports: [{ private: 80, public: 8080, type: 'tcp' }], project: 'web-app' },
    { id: hex('api'), name: 'web-app-api-1', image: 'web-app/api:latest', imageId: sha('apiimg'), state: 'running', status: 'Up 2 days', created: ago(172800), ports: [{ private: 3000, public: 3000, type: 'tcp' }], project: 'web-app' },
    { id: hex('pg'), name: 'web-app-postgres-1', image: 'postgres:16-alpine', imageId: sha('pg'), state: 'running', status: 'Up 2 days (healthy)', created: ago(172800), ports: [{ private: 5432, public: 5432, type: 'tcp' }], project: 'web-app' },
    { id: hex('redis'), name: 'web-app-redis-1', image: 'redis:7-alpine', imageId: sha('redis'), state: 'running', status: 'Up 2 days', created: ago(172800), ports: [{ private: 6379, public: 6379, type: 'tcp' }], project: 'web-app' },
    { id: hex('worker'), name: 'web-app-worker-1', image: 'web-app/api:latest', imageId: sha('apiimg'), state: 'running', status: 'Up 2 days', created: ago(172800), ports: [], project: 'web-app' },
    { id: hex('graf'), name: 'monitoring-grafana-1', image: 'grafana/grafana:11.2.0', imageId: sha('grafana'), state: 'running', status: 'Up 6 hours', created: ago(21600), ports: [{ private: 3000, public: 3001, type: 'tcp' }], project: 'monitoring' },
    { id: hex('prom'), name: 'monitoring-prometheus-1', image: 'prom/prometheus:v2.54.1', imageId: sha('prom'), state: 'running', status: 'Up 6 hours', created: ago(21600), ports: [{ private: 9090, public: 9090, type: 'tcp' }], project: 'monitoring' },
    { id: hex('mongo'), name: 'mongo', image: 'mongo:7', imageId: sha('mongo'), state: 'running', status: 'Up 5 hours', created: ago(18000), ports: [{ private: 27017, public: 27017, type: 'tcp' }], project: '' },
    { id: hex('rabbit'), name: 'rabbitmq', image: 'rabbitmq:3.13-management', imageId: sha('rabbit'), state: 'running', status: 'Up 5 hours', created: ago(18000), ports: [{ private: 5672, public: 5672, type: 'tcp' }, { private: 15672, public: 15672, type: 'tcp' }], project: '' },
    { id: hex('migrate'), name: 'web-app-migrate-1', image: 'web-app/api:latest', imageId: sha('apiimg'), state: 'exited', status: 'Exited (0) 2 days ago', created: ago(172900), ports: [], project: 'web-app' },
    { id: hex('seedjob'), name: 'pgadmin', image: 'dpage/pgadmin4:8.12', imageId: sha('pgadmin'), state: 'exited', status: 'Exited (137) 8 hours ago', created: ago(90000), ports: [], project: '' },
  ]
}

// ---- images ----
export function makeImages() {
  return [
    { id: sha('nginx'), tags: ['nginx:1.27-alpine'], size: 0.05 * GiB, created: ago(600000), containers: 1 },
    { id: sha('apiimg'), tags: ['web-app/api:latest'], size: 0.42 * GiB, created: ago(180000), containers: 3 },
    { id: sha('pg'), tags: ['postgres:16-alpine'], size: 0.27 * GiB, created: ago(900000), containers: 1 },
    { id: sha('redis'), tags: ['redis:7-alpine'], size: 0.041 * GiB, created: ago(900000), containers: 1 },
    { id: sha('grafana'), tags: ['grafana/grafana:11.2.0'], size: 0.62 * GiB, created: ago(500000), containers: 1 },
    { id: sha('prom'), tags: ['prom/prometheus:v2.54.1'], size: 0.28 * GiB, created: ago(500000), containers: 1 },
    { id: sha('mongo'), tags: ['mongo:7'], size: 0.81 * GiB, created: ago(700000), containers: 1 },
    { id: sha('rabbit'), tags: ['rabbitmq:3.13-management'], size: 0.24 * GiB, created: ago(700000), containers: 1 },
    { id: sha('node'), tags: ['node:20-slim'], size: 0.22 * GiB, created: ago(800000), containers: 0 },
    { id: sha('python'), tags: ['python:3.12-slim'], size: 0.13 * GiB, created: ago(800000), containers: 0 },
    { id: sha('pgadmin'), tags: ['dpage/pgadmin4:8.12'], size: 0.46 * GiB, created: ago(400000), containers: 0 },
    { id: sha('dangling'), tags: ['<none>'], size: 0.19 * GiB, created: ago(300000), containers: 0 },
  ]
}

// ---- volumes ----
export function makeVolumes() {
  const v = (n, s) => ({ name: n, driver: 'local', mountpoint: `/var/lib/docker/volumes/${n}/_data`, scope: 'local', createdAt: new Date((ago(s)) * 1000).toISOString() })
  return [
    v('web-app_pgdata', 172800),
    v('web-app_redisdata', 172800),
    v('monitoring_grafana', 21600),
    v('monitoring_prometheus', 21600),
    v('mongo_data', 18000),
    v(hex('orphan').slice(0, 64), 300000),
  ]
}

// ---- networks ----
export function makeNetworks() {
  return [
    { id: hex('netbridge'), name: 'bridge', driver: 'bridge', scope: 'local', internal: false, created: ago(900000) },
    { id: hex('nethost'), name: 'host', driver: 'host', scope: 'local', internal: false, created: ago(900000) },
    { id: hex('netnone'), name: 'none', driver: 'null', scope: 'local', internal: false, created: ago(900000) },
    { id: hex('netweb'), name: 'web-app_default', driver: 'bridge', scope: 'local', internal: false, created: ago(172800) },
    { id: hex('netmon'), name: 'monitoring_default', driver: 'bridge', scope: 'local', internal: false, created: ago(21600) },
  ]
}

// ---- compose stacks (derived names match container `project`) ----
export function makeStacks(containers) {
  const byProject = (p) => containers.filter((c) => c.project === p)
  const stack = (name, dir) => {
    const cs = byProject(name)
    return { name, running: cs.filter((c) => c.state === 'running').length, total: cs.length, configFiles: `${dir}/docker-compose.yml`, workingDir: dir, containers: cs }
  }
  return [stack('web-app', '/Users/dev/projects/web-app'), stack('monitoring', '/Users/dev/projects/monitoring')]
}

export const colimaStatus = {
  engine: 'colima', running: true, profile: 'default', runtime: 'docker', arch: 'aarch64',
  cpu: 4, memory: 8 * GiB, disk: 100 * GiB, kubernetes: false,
  dockerSocket: 'unix:///Users/dev/.colima/default/docker.sock', mountType: 'virtiofs', driver: 'macOS Virtualization.Framework',
}

export const self = { version: 'demo', basePath: '/', os: 'darwin', rss: 24 * 1024 * 1024, goroutines: 11, heapAlloc: 3 * 1024 * 1024 }
export const update = { current: 'demo', latest: '0.2.4', updateAvailable: false, managed: false, url: 'https://github.com/ParadoxInfinite/oriel/releases', publishedAt: '2026-06-21T14:14:30Z' }
export const provider = { enabled: false, url: '' }
export const themes = { dir: '/Users/dev/Library/Application Support/oriel/themes', themes: [] }
export const discovery = { roots: [{ id: 'r-demo', path: '/Users/dev/projects', traverse: true, enabled: true }], filter: { mode: 'off', patterns: [] }, aliases: {} }

export function makeDf(containers, images, volumes) {
  return {
    stoppedContainers: containers.filter((c) => c.state !== 'running').length, containersSize: 0.4 * GiB,
    danglingImages: images.filter((i) => i.tags.length === 0 || i.tags[0] === '<none>').length, imagesSize: 0.19 * GiB,
    buildCacheSize: 1.1 * GiB, unusedVolumes: 1, volumesSize: 0.3 * GiB,
    reclaimable: 0.4 * GiB + 0.19 * GiB + 1.1 * GiB + 0.3 * GiB,
  }
}

// A handful of past outages for the sidebar (kind/start/end in ms).
export function makeOutages() {
  const ms = now * 1000
  return [
    { kind: 'down', start: ms - 3600_000 * 15, end: ms - 3600_000 * 15 + 45_000 },
    { kind: 'offline', start: ms - 3600_000 * 6, end: ms - 3600_000 * 6 + 12_000 },
    { kind: 'offline', start: ms - 3600_000 * 2, end: ms - 3600_000 * 2 + 8_000 },
  ]
}

// Seeded CPU/mem history (~10 min) so the dashboard graph isn't empty on load.
export function makeHistory() {
  const pts = []
  const baseT = now * 1000 - 600 * 1000
  let x = 7
  for (let i = 0; i < 600; i++) {
    x = (x * 1103515245 + 12345) & 0x7fffffff
    const cpu = 4 + ((x >> 8) % 1800) / 100 // 4–22%
    pts.push({ t: baseT + i * 1000, cpu, mem: 2.0 * GiB + ((x >> 6) % 200) * 1024 * 1024, down: false })
  }
  return pts
}

export const SAMPLE_LOGS = [
  ['stdout', 'Listening on :3000'],
  ['stdout', 'Connected to postgres://web-app-postgres:5432/app'],
  ['stdout', 'Redis cache ready'],
  ['stdout', 'GET /healthz 200 1ms'],
  ['stdout', 'GET /api/users 200 14ms'],
  ['stderr', 'warn: slow query (212ms): SELECT * FROM events'],
  ['stdout', 'POST /api/orders 201 31ms'],
  ['stdout', 'GET /api/users/42 200 6ms'],
  ['stdout', 'worker: processed job #1841'],
  ['stdout', 'GET /metrics 200 2ms'],
]
