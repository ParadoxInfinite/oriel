<script>
  import './studio.css'

  import {
    status,
    containers,
    stats,
    fmt,
    openPalette,
    refreshImages,
    refreshVolumes,
    refreshNetworks,
    refreshStacks,
    setOverlayTheme,
    self,
    update,
  } from '../../platform/index.js'

  // Version label: real builds show "vX.Y.Z"; local builds show "dev" as-is.
  const verLabel = $derived(self.version || '')

  import { appearance, systemPref, initAppearance } from './theme.svelte.js'
  import Icon from './lib/Icon.svelte'
  import Outages from './lib/Outages.svelte'
  import OpTray from '../../components/OpTray.svelte'
  import Dashboard from './panels/Dashboard.svelte'
  import Containers from './panels/Containers.svelte'
  import Resources from './panels/Resources.svelte'
  import Stacks from './panels/Stacks.svelte'
  import Settings from './panels/Settings.svelte'

  initAppearance()
  const dark = $derived(appearance.mode === 'system' ? systemPref.dark : appearance.mode === 'dark')
  const accent = $derived(appearance.accents[dark ? 'dark' : 'light'] ?? '#5b5bd6')
  // Keep the host's global overlays in sync with Studio's appearance + accent.
  $effect(() => setOverlayTheme(dark ? 'dark' : 'light', accent))

  const NAV = [
    { name: 'Dashboard', icon: 'dashboard' },
    { name: 'Containers', icon: 'box' },
    { name: 'Images', icon: 'harddrive' },
    { name: 'Volumes', icon: 'database' },
    { name: 'Networks', icon: 'network' },
    { name: 'Stacks', icon: 'layers' },
  ]
  const SUBTITLES = {
    Dashboard: 'Overview of your colima environment',
    Containers: 'Running and stopped containers',
    Images: 'Images stored on disk',
    Volumes: 'Persistent volumes',
    Networks: 'Container networks',
    Stacks: 'Docker Compose projects',
    Settings: 'Appearance, themes and AI',
  }
  let active = $state('Dashboard')

  const loaders = { Images: refreshImages, Volumes: refreshVolumes, Networks: refreshNetworks, Stacks: refreshStacks }
  const loaded = new Set()
  $effect(() => {
    if (loaders[active] && !loaded.has(active)) {
      loaded.add(active)
      loaders[active]()
    }
  })

  const s = $derived(status.data)
  const engine = $derived(s?.engine === 'docker' ? 'Docker' : 'Colima')
  const vmState = $derived(status.loading ? 'warn' : status.error ? 'bad' : s?.running ? 'on' : 'off')
  const vmWord = $derived(status.loading ? 'Connecting' : status.error ? 'Offline' : s?.running ? 'Running' : 'Stopped')
  const running = $derived(containers.list.filter((c) => c.state === 'running'))
  const samples = $derived(running.map((c) => stats.byId[c.id]).filter(Boolean))
  const usedMem = $derived(samples.reduce((a, x) => a + x.mem, 0))
</script>

<div class="studio-root {dark ? 'dark' : ''}" style="--accent:{accent}">
  <!-- Sidebar -->
  <aside class="flex w-[232px] shrink-0 flex-col border-r border-[var(--border)] bg-[var(--sidebar)]">
    <div class="flex items-center gap-2.5 px-5 pb-3 pt-5">
      <div class="grid h-7 w-7 place-items-center rounded-lg bg-[var(--accent)] text-white shadow-[var(--shadow-sm)]">
        <Icon name="box" size={16} stroke={2} />
      </div>
      <span class="text-[15px] font-semibold tracking-tight">Oriel</span>
    </div>

    <nav class="flex flex-1 flex-col gap-0.5 px-3 py-2">
      {#each NAV as item}
        <button class="nav {active === item.name ? 'on' : ''}" onclick={() => (active = item.name)}>
          <Icon name={item.icon} size={17} class="nav-i" />
          {item.name}
        </button>
      {/each}
      <div class="my-1.5 mx-2 border-t border-[var(--border)]"></div>
      <button class="nav {active === 'Settings' ? 'on' : ''}" onclick={() => (active = 'Settings')}>
        <Icon name="settings" size={17} class="nav-i" />
        Settings
      </button>
    </nav>

    <OpTray />
    <Outages />

    <!-- VM status footer -->
    <div class="mx-3 mb-3 rounded-lg border border-[var(--border)] bg-[var(--panel)] p-3 shadow-[var(--shadow-sm)]">
      <div class="flex items-center gap-2">
        {#if vmState === 'on'}<span class="beacon"></span>{:else}<span class="h-2 w-2 rounded-full {vmState === 'bad' ? 'bg-[var(--red)]' : vmState === 'warn' ? 'bg-[var(--amber)]' : 'bg-[var(--slate-dot)]'}"></span>{/if}
        <span class="text-[13px] font-medium">{engine}</span>
        <span class="ml-auto text-xs {vmState === 'on' ? 'text-[var(--green)]' : vmState === 'bad' ? 'text-[var(--red)]' : 'text-[var(--text-3)]'}">{vmWord}</span>
      </div>
      {#if s?.running && !status.error}
        <div class="mono tnum mt-2 flex items-center justify-between text-[11px] text-[var(--text-3)]">
          <span>{running.length} running</span>
          <span>{fmt.bytes(usedMem)}</span>
        </div>
      {/if}
    </div>

    <!-- Brand + version footer -->
    <div class="flex items-center justify-between gap-2 border-t border-[var(--border)] px-5 py-3">
      <span class="text-[11px] font-medium tracking-wide text-[var(--text-3)]">Oriel</span>
      <div class="flex items-center gap-1.5">
        {#if update.available}
          <a href={update.url} target="_blank" rel="noopener" class="rounded-full bg-[var(--accent-tint-2)] px-2 py-0.5 font-mono text-[10px] font-medium text-[var(--accent)] hover:underline" title="Update available — v{update.latest}">update ↗</a>
        {/if}
        {#if verLabel}<span class="mono rounded-full border border-[var(--border)] bg-[var(--panel-2)] px-2 py-0.5 text-[10px] font-medium text-[var(--text-2)]">{verLabel}</span>{/if}
      </div>
    </div>
  </aside>

  <!-- Main -->
  <div class="flex min-w-0 flex-1 flex-col">
    <header class="flex h-[60px] shrink-0 items-center gap-4 border-b border-[var(--border)] bg-[var(--panel)] px-6">
      <div>
        <h1 class="text-[15px] font-semibold leading-none tracking-tight">{active}</h1>
        <p class="mt-1 text-xs text-[var(--text-3)]">{SUBTITLES[active]}</p>
      </div>
      <button class="btn btn-default ml-auto btn-sm" onclick={openPalette}>
        <Icon name="command" size={13} /> Run command
        <kbd class="mono ml-1 rounded border border-[var(--border)] bg-[var(--panel-2)] px-1.5 text-[10px] text-[var(--text-3)]">⌘K</kbd>
      </button>
    </header>

    <main class="min-h-0 flex-1 overflow-auto p-6">
      {#if active === 'Dashboard'}
        <Dashboard navigate={(v) => (active = v)} />
      {:else if active === 'Containers'}
        <Containers />
      {:else if active === 'Images'}
        <Resources kind="images" />
      {:else if active === 'Volumes'}
        <Resources kind="volumes" />
      {:else if active === 'Networks'}
        <Resources kind="networks" />
      {:else if active === 'Stacks'}
        <Stacks navigate={(v) => (active = v)} />
      {:else if active === 'Settings'}
        <Settings />
      {/if}
    </main>
  </div>
</div>
