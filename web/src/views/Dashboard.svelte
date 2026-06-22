<script>
  import { onMount } from 'svelte'
  import {
    status, refreshStatus, lifecycle, containers, stacks,
    images, volumes, networks, refreshImages, refreshVolumes, refreshNetworks,
    self, fmt, DashboardStats,
  } from '../platform/index.js'
  import { action } from '../lib/ui.js'
  import Icon from '../components/Icon.svelte'
  import Heartbeat from '../components/Heartbeat.svelte'

  const { bytes } = fmt

  let { navigate } = $props()

  // Shared telemetry (CPU/mem utilisation, sparkline); cards below are Classic's.
  const d = new DashboardStats()
  const s = $derived(d.sys)
  const isDocker = $derived(d.isDocker)
  const running = $derived(d.running)
  const usedMem = $derived(d.usedMem)
  const memLimit = $derived(d.memLimit)
  const cpuPct = $derived(d.cpuPct)
  const memPct = $derived(d.memPct)
  const pulse = $derived(d.pulse)
  const servicesUp = $derived(d.servicesUp)
  // Lead each card with its most meaningful active count; context sits at the foot.
  const inventory = $derived([
    { label: 'Containers', count: running.length, foot: `of ${containers.list.length} total`, icon: 'box', to: 'Containers' },
    { label: 'Stacks', count: stacks.list.length, foot: `${servicesUp} services up`, icon: 'layers', to: 'Stacks' },
    { label: 'Images', count: images.list.length, foot: 'on disk', icon: 'harddrive', to: 'Images' },
    { label: 'Volumes', count: volumes.list.length, foot: 'mounted', icon: 'database', to: 'Volumes' },
    { label: 'Network bridges', count: networks.list.length, foot: 'configured', icon: 'network', to: 'Networks' },
  ])

  onMount(() => {
    refreshImages()
    refreshVolumes()
    refreshNetworks()
  })
</script>

{#if status.loading}
  <div class="flex h-full items-center justify-center text-sm text-muted">
    <span class="h-4 w-4 animate-spin rounded-full border-2 border-border border-t-accent"></span>
  </div>
{:else if !status.running}
  <div class="flex h-full flex-col items-center justify-center text-center">
    <div class="mb-5 grid h-12 w-12 place-items-center rounded-xl border border-border bg-surface {status.error ? 'text-danger' : 'text-faint'}">
      <Icon name="box" size={22} />
    </div>
    {#if status.error}
      <h2 class="display text-lg font-medium tracking-tight">Backend unreachable</h2>
      <p class="mt-1.5 max-w-sm text-sm text-muted">Cannot reach the Oriel backend service.</p>
      <p class="mt-4 font-mono text-xs text-danger">{status.error}</p>
    {:else if isDocker}
      <h2 class="display text-lg font-medium tracking-tight">Docker engine unreachable</h2>
      <p class="mt-1.5 max-w-sm text-sm text-muted">Start the Docker daemon, then retry.</p>
      <div class="mt-4 rounded-[--radius] border border-border bg-surface px-3 py-2 font-mono text-xs text-muted">
        sudo systemctl start docker <span class="text-faint">— Linux</span>
      </div>
      <button class="pop mt-5 flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm text-muted hover:text-fg" onclick={() => refreshStatus()}>
        <Icon name="restart" size={14} /> Retry
      </button>
    {:else}
      <h2 class="display text-lg font-medium tracking-tight">Colima is stopped</h2>
      <p class="mt-1.5 max-w-sm text-sm text-muted">Start the Colima VM to manage your containers, images, and stacks.</p>
      <button class="pop mt-6 flex items-center gap-2 rounded-lg bg-accent px-5 py-2 text-sm font-medium text-accent-fg shadow-[0_0_20px_-4px_var(--color-accent)]" onclick={() => lifecycle('start')}>
        <Icon name="play" size={15} stroke={2} /> Start Colima
      </button>
    {/if}
  </div>
{:else}
  <div class="mx-auto max-w-5xl pb-4">
    <!-- Header -->
    <div class="rise mb-5 flex items-center justify-between">
      <div class="flex items-center gap-2.5">
        <span class="h-2.5 w-2.5 rounded-full bg-ok pulse-ok"></span>
        <div>
          <h2 class="display text-base font-medium leading-none tracking-tight">{isDocker ? 'Docker engine running' : 'Colima is running'}</h2>
          {#if isDocker}
            <p class="mt-1.5 text-xs text-muted">{s.driver || 'docker'} · {s.arch}{s.version ? ` · v${s.version}` : ''}</p>
          {:else}
            <p class="mt-1.5 text-xs text-muted">profile <span class="font-mono text-faint">{s.profile}</span> · {s.runtime} · {s.arch}</p>
          {/if}
        </div>
      </div>
      {#if !isDocker}
        <div class="flex gap-2">
          <button class="{action('accent')} flex items-center gap-1.5" onclick={() => lifecycle('restart')}><Icon name="restart" size={13} /> Restart</button>
          <button class="{action('danger')} flex items-center gap-1.5" onclick={() => lifecycle('stop')}><Icon name="stop" size={13} /> Stop</button>
        </div>
      {/if}
    </div>

    <!-- System pulse -->
    <div class="rise card rounded-[--radius] p-5" style="animation-delay:40ms">
      <div class="flex items-baseline justify-between">
        <span class="text-[11px] uppercase tracking-[0.2em] text-faint">System pulse</span>
        <span class="tnum font-mono text-2xl">{cpuPct.toFixed(0)}<span class="text-base text-faint">%</span> <span class="ml-2 text-xs text-faint">cpu · {s.cpu} cores</span></span>
      </div>
      <div class="mt-3 h-36"><Heartbeat points={pulse} /></div>
    </div>

    <!-- Inventory (clickable) -->
    <div class="rise mt-3 grid grid-cols-2 gap-3 sm:grid-cols-5" style="animation-delay:80ms">
      {#each inventory as it (it.label)}
        <button class="card lift group rounded-[--radius] p-4 text-left" onclick={() => navigate?.(it.to)}>
          <div class="flex items-center justify-between text-faint">
            <Icon name={it.icon} size={16} />
            <span class="transition-transform duration-300 group-hover:translate-x-0.5 group-hover:text-accent">→</span>
          </div>
          <div class="tnum mt-3 font-mono text-2xl">{it.count}</div>
          <div class="text-[11px] uppercase tracking-wider text-muted">{it.label}</div>
          <div class="text-[10px] text-faint">{it.foot}</div>
        </button>
      {/each}
    </div>

    <!-- Memory -->
    <div class="rise mt-3 card rounded-[--radius] p-5" style="animation-delay:120ms">
      <div class="flex items-center justify-between">
        <span class="text-[11px] uppercase tracking-[0.2em] text-faint">Memory</span>
        <span class="text-[11px] text-faint">{bytes(usedMem)} / {bytes(memLimit)}</span>
      </div>
      <div class="mt-3 h-2 w-full overflow-hidden rounded-full bg-surface-2">
        <div class="h-full rounded-full bg-accent transition-all duration-700" style="width:{memPct}%"></div>
      </div>
      <div class="mt-1.5 text-[11px] text-faint">{memPct.toFixed(0)}% of allocation</div>
    </div>

    <!-- Engine details -->
    <div class="rise mt-6 mb-2.5 text-[11px] uppercase tracking-[0.2em] text-faint" style="animation-delay:200ms">{isDocker ? 'Docker engine' : 'Virtual machine'}</div>
    <div class="rise grid grid-cols-2 gap-3 sm:grid-cols-4" style="animation-delay:200ms">
      {#each isDocker ? [{ label: 'CPUs', value: String(s.cpu) }, { label: 'Memory', value: bytes(s.memory) }, { label: 'Version', value: s.version || '—' }, { label: 'Driver', value: s.driver || '—' }] : [{ label: 'CPUs', value: String(s.cpu) }, { label: 'Memory', value: bytes(s.memory) }, { label: 'Disk', value: bytes(s.disk) }, { label: 'Kubernetes', value: s.kubernetes ? 'on' : 'off' }] as c}
        <div class="card rounded-[--radius] p-4">
          <div class="text-[11px] uppercase tracking-wider text-muted">{c.label}</div>
          <div class="tnum mt-1.5 font-mono text-lg">{c.value}</div>
        </div>
      {/each}
    </div>

    <div class="rise mt-3 card flex items-center gap-3 rounded-[--radius] p-4" style="animation-delay:200ms">
      <Icon name="network" size={16} class="shrink-0 text-faint" />
      <div class="min-w-0">
        <div class="text-[11px] uppercase tracking-wider text-muted">Docker socket</div>
        <div class="mt-1 truncate font-mono text-sm text-fg/90">{s.dockerSocket}</div>
      </div>
    </div>

    {#if self.rss}
      <div class="mt-5 text-center text-[11px] text-faint/70">
        Oriel backend · <span class="tnum font-mono">{bytes(self.rss)}</span> RAM · {self.goroutines} goroutines
      </div>
    {/if}
  </div>
{/if}
