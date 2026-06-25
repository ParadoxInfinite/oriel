import { status } from './status.svelte.js'
import { containers } from './containers.svelte.js'
import { stats, history } from './live.svelte.js'
import { stacks } from './stacks.svelte.js'

// Derived telemetry behind the dashboard headline + sparkline. Shared so both
// editions read the same CPU/memory utilisation and only differ in chrome.
export class DashboardStats {
  sys = $derived(status.data)
  isDocker = $derived(status.data?.engine === 'docker')
  running = $derived(containers.list.filter((c) => c.state === 'running'))
  samples = $derived(this.running.map((c) => stats.byId[c.id]).filter(Boolean))
  totalCpu = $derived(this.samples.reduce((a, x) => a + x.cpu, 0))
  usedMem = $derived(this.samples.reduce((a, x) => a + x.mem, 0))
  memLimit = $derived(this.samples.find((x) => x.memLimit)?.memLimit || status.data?.memory || 0)
  cpuCap = $derived((status.data?.cpu || 1) * 100)
  cpuPct = $derived(Math.min(100, (this.totalCpu / this.cpuCap) * 100))
  memPct = $derived(this.memLimit ? Math.min(100, (this.usedMem / this.memLimit) * 100) : 0)
  // History as utilisation % of total capacity, matches the headline number.
  pulse = $derived(
    history.points.map((p) => ({
      t: p.t,
      cpu: this.cpuCap ? Math.min(100, (p.cpu / this.cpuCap) * 100) : 0,
      down: p.down,
    }))
  )
  servicesUp = $derived(stacks.list.reduce((a, x) => a + x.running, 0))
}
