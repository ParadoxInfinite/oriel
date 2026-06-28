<script>
  import './studio.css'

  import {
    status,
    containers,
    stats,
    fmt,
    openPalette,
    nav,
    navigate,
    refreshImages,
    refreshVolumes,
    refreshNetworks,
    refreshStacks,
    setOverlayTheme,
    self,
    update,
    canSelfUpdate,
    promptUpdate,
    OpTray,
    t,
    tn,
  } from '../../platform/index.js'

  // Version label: real builds show "vX.Y.Z"; local builds show "dev" as-is.
  const verLabel = $derived(self.version || '')

  // The update pill opens the confirm-update modal directly when this install can
  // self-update; otherwise it falls back to the Updates panel (e.g. Homebrew,
  // where the user updates via `brew upgrade`).
  function onUpdatePill() {
    if (canSelfUpdate()) promptUpdate()
    else navigate('Settings')
  }

  import { appearance, systemPref, initAppearance } from './theme.svelte.js'
  import Icon from './lib/Icon.svelte'
  import Outages from './lib/Outages.svelte'
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

  // Translated view labels, keyed by the canonical view id (which drives routing).
  const VIEW_LABELS = $derived({
    Dashboard: t('nav.dashboard'),
    Containers: t('nav.containers'),
    Images: t('nav.images'),
    Volumes: t('nav.volumes'),
    Networks: t('nav.networks'),
    Stacks: t('nav.stacks'),
    Settings: t('nav.settings'),
  })
  const NAV = $derived([
    { name: 'Dashboard', icon: 'dashboard', label: VIEW_LABELS.Dashboard },
    { name: 'Containers', icon: 'box', label: VIEW_LABELS.Containers },
    { name: 'Images', icon: 'harddrive', label: VIEW_LABELS.Images },
    { name: 'Volumes', icon: 'database', label: VIEW_LABELS.Volumes },
    { name: 'Networks', icon: 'network', label: VIEW_LABELS.Networks },
    { name: 'Stacks', icon: 'layers', label: VIEW_LABELS.Stacks },
  ])
  const SUBTITLES = $derived({
    Dashboard: t('nav.subtitle.dashboard'),
    Containers: t('nav.subtitle.containers'),
    Images: t('nav.subtitle.images'),
    Volumes: t('nav.subtitle.volumes'),
    Networks: t('nav.subtitle.networks'),
    Stacks: t('nav.subtitle.stacks'),
    Settings: t('nav.subtitle.settings'),
  })
  // Active view comes from the shared nav seam so the palette can move it; the
  // sidebar buttons drive it through navigate().
  const active = $derived(nav.view)

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
  const vmWord = $derived(status.loading ? t('nav.vm.connecting') : status.error ? t('nav.vm.offline') : s?.running ? t('nav.vm.running') : t('nav.vm.stopped'))
  const running = $derived(containers.list.filter((c) => c.state === 'running'))
  const samples = $derived(running.map((c) => stats.byId[c.id]).filter(Boolean))
  const usedMem = $derived(samples.reduce((a, x) => a + x.mem, 0))

  // On a phone the sidebar is an off-canvas drawer; on md+ it's always shown.
  // Navigating (or tapping the backdrop) closes it.
  let mobileNav = $state(false)
  function go(view) {
    navigate(view)
    mobileNav = false
  }
</script>

<div class="studio-root {dark ? 'dark' : ''}" style="--accent:{accent}">
  <!-- Mobile drawer backdrop -->
  {#if mobileNav}
    <button class="fixed inset-0 z-30 bg-black/50 md:hidden" aria-label={t('nav.closeMenu')} onclick={() => (mobileNav = false)}></button>
  {/if}

  <!-- Sidebar: static on md+, off-canvas drawer below -->
  <aside
    class="fixed inset-y-0 left-0 z-40 flex w-[264px] flex-col border-r border-[var(--border)] bg-[var(--sidebar)] transition-transform duration-200 ease-out
           md:static md:z-auto md:w-[232px] md:shrink-0 md:translate-x-0 md:transition-none
           {mobileNav ? 'translate-x-0 shadow-2xl' : '-translate-x-full'}"
  >
    <div class="flex items-center gap-2.5 px-5 pb-3 pt-5">
      <div class="grid h-7 w-7 place-items-center rounded-lg bg-[var(--accent)] text-white shadow-[var(--shadow-sm)]">
        <Icon name="box" size={16} stroke={2} />
      </div>
      <span class="text-[15px] font-semibold tracking-tight">Oriel</span>
      <button class="ml-auto grid h-8 w-8 place-items-center rounded-lg text-[var(--text-3)] hover:bg-[var(--hover)] md:hidden" aria-label={t('nav.closeMenu')} onclick={() => (mobileNav = false)}>
        <Icon name="x" size={18} />
      </button>
    </div>

    <nav class="flex flex-1 flex-col gap-0.5 px-3 py-2">
      {#each NAV as item}
        <button class="nav {active === item.name ? 'on' : ''}" onclick={() => go(item.name)}>
          <Icon name={item.icon} size={17} class="nav-i" />
          {item.label}
        </button>
      {/each}
      <div class="my-1.5 mx-2 border-t border-[var(--border)]"></div>
      <button class="nav {active === 'Settings' ? 'on' : ''}" onclick={() => go('Settings')}>
        <Icon name="settings" size={17} class="nav-i" />
        {VIEW_LABELS.Settings}
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
          <span>{tn('nav.runningCount', running.length)}</span>
          <span>{fmt.bytes(usedMem)}</span>
        </div>
      {/if}
    </div>

    <!-- Brand + version footer -->
    <div class="flex items-center justify-between gap-2 border-t border-[var(--border)] px-5 py-3">
      <span class="text-[11px] font-medium tracking-wide text-[var(--text-3)]">Oriel</span>
      <div class="flex items-center gap-1.5">
        {#if update.available}
          <button type="button" onclick={onUpdatePill} class="rounded-full bg-[var(--accent-tint-2)] px-2 py-0.5 font-mono text-[10px] font-medium text-[var(--accent)] hover:underline" title={t('nav.updateTitle', { version: update.latest, action: canSelfUpdate() ? t('nav.updateInstallNow') : t('nav.updateOpen') })}>{t('nav.updatePill')}</button>
        {/if}
        {#if verLabel}<span class="mono rounded-full border border-[var(--border)] bg-[var(--panel-2)] px-2 py-0.5 text-[10px] font-medium text-[var(--text-2)]">{verLabel}</span>{/if}
      </div>
    </div>
  </aside>

  <!-- Main -->
  <div class="flex min-w-0 flex-1 flex-col">
    <header class="flex h-[60px] shrink-0 items-center gap-3 border-b border-[var(--border)] bg-[var(--panel)] px-4 md:px-6">
      <button class="-ml-1 grid h-9 w-9 shrink-0 place-items-center rounded-lg text-[var(--text-2)] hover:bg-[var(--hover)] md:hidden" aria-label={t('nav.openMenu')} onclick={() => (mobileNav = true)}>
        <Icon name="menu" size={20} />
      </button>
      <div class="min-w-0">
        <h1 class="truncate text-[15px] font-semibold leading-none tracking-tight">{VIEW_LABELS[active]}</h1>
        <p class="mt-1 truncate text-xs text-[var(--text-3)]">{SUBTITLES[active]}</p>
      </div>
      <button class="btn btn-default ml-auto btn-sm" onclick={openPalette}>
        <Icon name="command" size={13} /> <span class="hidden sm:inline">{t('nav.runCommand')}</span>
        <kbd class="mono ml-1 hidden rounded border border-[var(--border)] bg-[var(--panel-2)] px-1.5 text-[10px] text-[var(--text-3)] sm:inline">⌘K</kbd>
      </button>
    </header>

    <main class="min-h-0 flex-1 overflow-auto p-4 md:p-6">
      {#if active === 'Dashboard'}
        <Dashboard {navigate} />
      {:else if active === 'Containers'}
        <Containers />
      {:else if active === 'Images'}
        <Resources kind="images" />
      {:else if active === 'Volumes'}
        <Resources kind="volumes" />
      {:else if active === 'Networks'}
        <Resources kind="networks" />
      {:else if active === 'Stacks'}
        <Stacks {navigate} />
      {:else if active === 'Settings'}
        <Settings />
      {/if}
    </main>
  </div>
</div>
