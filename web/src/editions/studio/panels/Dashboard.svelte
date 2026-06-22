<script>
  import { onMount } from 'svelte'
  import {
    status,
    containers,
    stacks,
    images,
    volumes,
    networks,
    self,
    fmt,
    lifecycle,
    refreshImages,
    refreshVolumes,
    refreshNetworks,
    DashboardStats,
  } from '../../../platform/index.js'
  import Icon from '../lib/Icon.svelte'
  import Chart from '../lib/Chart.svelte'
  import SystemPrune from '../lib/SystemPrune.svelte'

  let showPrune = $state(false)

  let { navigate } = $props()

  onMount(() => {
    refreshImages()
    refreshVolumes()
    refreshNetworks()
  })

  // Shared telemetry (CPU/mem utilisation, sparkline); chrome below is Studio's.
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

  // Inventory cards follow sidebar order: Containers, Images, Volumes, Networks, Stacks.
  const cards = $derived([
    { label: 'Containers', value: running.length, sub: `of ${containers.list.length} total`, icon: 'box', to: 'Containers', tint: '#5b5bd6', bg: 'color-mix(in srgb, #5b5bd6 12%, var(--panel))' },
    { label: 'Images', value: images.list.length, sub: 'on disk', icon: 'harddrive', to: 'Images', tint: '#b06f1a', bg: 'color-mix(in srgb, #d98a1a 14%, var(--panel))' },
    { label: 'Volumes', value: volumes.list.length, sub: 'mounted', icon: 'database', to: 'Volumes', tint: '#2563eb', bg: 'color-mix(in srgb, #2563eb 12%, var(--panel))' },
    { label: 'Networks', value: networks.list.length, sub: 'configured', icon: 'network', to: 'Networks', tint: '#9333ea', bg: 'color-mix(in srgb, #9333ea 12%, var(--panel))' },
    { label: 'Stacks', value: stacks.list.length, sub: `${servicesUp} services up`, icon: 'layers', to: 'Stacks', tint: '#0e8f6e', bg: 'color-mix(in srgb, #0e8f6e 14%, var(--panel))' },
  ])

  const specs = $derived(
    isDocker
      ? [
          { k: 'CPUs', v: String(s?.cpu ?? '—') },
          { k: 'Memory', v: fmt.bytes(s?.memory) },
          { k: 'Version', v: s?.version || '—' },
          { k: 'Driver', v: s?.driver || '—' },
        ]
      : [
          { k: 'CPUs', v: String(s?.cpu ?? '—') },
          { k: 'Memory', v: fmt.bytes(s?.memory) },
          { k: 'Disk', v: fmt.bytes(s?.disk) },
          { k: 'Runtime', v: s?.runtime || '—' },
          { k: 'Architecture', v: s?.arch || '—' },
          { k: 'Kubernetes', v: s?.kubernetes ? 'Enabled' : 'Disabled' },
        ]
  )
</script>

{#if status.loading}
  <div class="grid h-full place-items-center">
    <div class="flex items-center gap-2.5 text-sm text-[var(--text-3)]">
      <span class="h-4 w-4 animate-spin rounded-full border-2 border-[var(--border-strong)] border-t-[var(--accent)]"></span>
      Connecting…
    </div>
  </div>
{:else if !status.running}
  <!-- Friendly zero-states -->
  <div class="grid h-full place-items-center px-6">
    <div class="rise card max-w-md p-8 text-center" style="box-shadow:var(--shadow-md)">
      {#if status.error}
        <div class="mx-auto mb-4 grid h-12 w-12 place-items-center rounded-xl bg-[var(--red-tint)] text-[var(--red)]"><Icon name="network" size={22} /></div>
        <h2 class="text-lg font-semibold tracking-tight">Backend unreachable</h2>
        <p class="mx-auto mt-1.5 max-w-xs text-sm text-[var(--text-2)]">We can't reach the Oriel backend service right now.</p>
        <p class="mono mt-4 rounded-lg bg-[var(--panel-2)] px-3 py-2 text-xs text-[var(--red)]">{status.error}</p>
      {:else if isDocker}
        <div class="mx-auto mb-4 grid h-12 w-12 place-items-center rounded-xl bg-[var(--amber-tint)] text-[var(--amber)]"><Icon name="box" size={22} /></div>
        <h2 class="text-lg font-semibold tracking-tight">Docker engine isn't responding</h2>
        <p class="mx-auto mt-1.5 max-w-xs text-sm text-[var(--text-2)]">Start the Docker daemon, then reload.</p>
        <button class="btn btn-default mx-auto mt-5" onclick={() => location.reload()}><Icon name="restart" size={15} /> Reload</button>
      {:else}
        <div class="mx-auto mb-4 grid h-12 w-12 place-items-center rounded-xl bg-[var(--accent-tint)] text-[var(--accent)]"><Icon name="box" size={22} /></div>
        <h2 class="text-lg font-semibold tracking-tight">Colima is stopped</h2>
        <p class="mx-auto mt-1.5 max-w-xs text-sm text-[var(--text-2)]">Start the virtual machine to manage your containers, images and stacks.</p>
        <button class="btn btn-primary mx-auto mt-6" onclick={() => lifecycle('start')}><Icon name="play" size={15} /> Start Colima</button>
      {/if}
    </div>
  </div>
{:else}
  <div class="mx-auto flex max-w-5xl flex-col gap-5">
    <!-- Status header -->
    <div class="rise card flex flex-wrap items-center gap-4 p-5">
      <span class="beacon"></span>
      <div>
        <h2 class="text-[15px] font-semibold tracking-tight">{isDocker ? 'Docker engine is running' : 'Colima is running'}</h2>
        <p class="mt-0.5 text-[13px] text-[var(--text-2)]">
          {#if isDocker}
            {s.driver || 'docker'} · {s.arch}{s.version ? ` · v${s.version}` : ''}
          {:else}
            Profile <span class="mono text-[var(--text)]">{s.profile}</span> · {s.runtime} · {s.arch}
          {/if}
        </p>
      </div>
      <div class="ml-auto flex gap-2">
        <button class="btn btn-default" onclick={() => (showPrune = true)}><Icon name="broom" size={14} /> Reclaim space</button>
        {#if !isDocker}
          <button class="btn btn-default" onclick={() => lifecycle('restart')}><Icon name="restart" size={14} /> Restart</button>
          <button class="btn btn-danger" onclick={() => lifecycle('stop')}><Icon name="stop" size={14} /> Stop</button>
        {/if}
      </div>
    </div>

    <!-- Inventory cards -->
    <div class="rise grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5" style="animation-delay:40ms">
      {#each cards as c (c.label)}
        <button class="card card-link p-4 text-left" onclick={() => navigate?.(c.to)}>
          <div class="flex items-center gap-3">
            <div class="grid h-9 w-9 shrink-0 place-items-center rounded-lg" style="background:{c.bg};color:{c.tint}"><Icon name={c.icon} size={18} /></div>
            <div class="tnum text-2xl font-semibold tracking-tight">{c.value}</div>
          </div>
          <div class="mt-2.5 text-[13px] font-medium text-[var(--text)]">{c.label}</div>
          <div class="mt-0.5 text-xs text-[var(--text-3)]">{c.sub}</div>
        </button>
      {/each}
    </div>

    <!-- Resource usage + CPU history, side by side -->
    <div class="rise grid gap-4 lg:grid-cols-2" style="animation-delay:80ms">
      <div class="card flex flex-col p-5">
        <span class="eyebrow">Resource usage</span>
        <div class="mt-4 flex flex-1 flex-col justify-center gap-4">
          <div>
            <div class="flex items-baseline justify-between">
              <span class="text-[13px] font-medium">CPU</span>
              <span class="mono tnum text-[13px] text-[var(--text-2)]">{cpuPct.toFixed(0)}% <span class="text-[var(--text-3)]">· {s.cpu} cores</span></span>
            </div>
            <div class="meter mt-2"><span style="width:{cpuPct}%;background:var(--accent)"></span></div>
          </div>
          <div>
            <div class="flex items-baseline justify-between">
              <span class="text-[13px] font-medium">Memory</span>
              <span class="mono tnum text-[13px] text-[var(--text-2)]">{fmt.bytes(usedMem)} <span class="text-[var(--text-3)]">/ {fmt.bytes(memLimit)}</span></span>
            </div>
            <div class="meter mt-2"><span style="width:{memPct}%;background:#2563eb"></span></div>
          </div>
        </div>
      </div>

      <div class="card flex flex-col p-5">
        <div class="flex items-baseline justify-between">
          <span class="eyebrow">CPU · last 30 min</span>
          <span class="mono tnum text-[13px] text-[var(--text-2)]">{cpuPct.toFixed(0)}% <span class="text-[var(--text-3)]">now</span></span>
        </div>
        <div class="mt-3 flex-1 pl-7">
          <Chart points={pulse} height={120} />
        </div>
      </div>
    </div>

    <!-- VM details -->
    <div class="rise card overflow-hidden" style="animation-delay:120ms">
      <div class="border-b border-[var(--border)] px-5 py-3"><span class="eyebrow">{isDocker ? 'Docker engine' : 'Virtual machine'}</span></div>
      <div class="grid grid-cols-2 sm:grid-cols-3">
        {#each specs as spec, i}
          <div class="px-5 py-3.5 {i % 2 === 0 ? 'border-r border-[var(--border)]' : ''} {i % 3 !== 2 ? 'sm:border-r sm:border-[var(--border)]' : 'sm:border-r-0'} {i >= 2 ? 'border-t border-[var(--border)]' : ''} {i >= 3 ? 'sm:border-t' : 'sm:border-t-0'}">
            <div class="text-xs text-[var(--text-3)]">{spec.k}</div>
            <div class="mono tnum mt-1 text-[13px] font-medium">{spec.v}</div>
          </div>
        {/each}
      </div>
      <div class="flex items-center gap-3 border-t border-[var(--border)] px-5 py-3.5">
        <Icon name="network" size={16} class="shrink-0 text-[var(--text-3)]" />
        <div class="min-w-0">
          <div class="text-xs text-[var(--text-3)]">Docker socket</div>
          <div class="mono mt-0.5 truncate text-[13px] text-[var(--text-2)]">{s.dockerSocket}</div>
        </div>
      </div>
    </div>

    {#if showPrune}<SystemPrune onClose={() => (showPrune = false)} />{/if}

    {#if self.rss}
      <p class="rise pb-1 text-center text-xs text-[var(--text-3)]" style="animation-delay:160ms">
        Oriel backend · <span class="mono tnum">{fmt.bytes(self.rss)}</span> memory · {self.goroutines} goroutines
      </p>
    {/if}
  </div>
{/if}
