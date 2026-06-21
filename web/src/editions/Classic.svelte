<script>
  import {
    status,
    self,
    update,
    openPalette,
    refreshImages,
    refreshVolumes,
    refreshNetworks,
    refreshStacks,
    setOverlayTheme,
  } from '../platform/index.js'
  import Dashboard from '../views/Dashboard.svelte'
  import Containers from '../views/Containers.svelte'
  import Images from '../views/Images.svelte'
  import Volumes from '../views/Volumes.svelte'
  import Networks from '../views/Networks.svelte'
  import Stacks from '../views/Stacks.svelte'
  import Settings from '../views/Settings.svelte'
  import Icon from '../components/Icon.svelte'
  import RecentOutages from '../components/RecentOutages.svelte'
  import OpTray from '../components/OpTray.svelte'

  // Classic's own @theme tokens already style the global overlays — no override.
  $effect(() => setOverlayTheme('classic'))

  const nav = [
    { name: 'Dashboard', icon: 'dashboard' },
    { name: 'Containers', icon: 'box' },
    { name: 'Images', icon: 'harddrive' },
    { name: 'Volumes', icon: 'database' },
    { name: 'Networks', icon: 'network' },
    { name: 'Stacks', icon: 'layers' },
  ]
  let active = $state('Dashboard')

  // Lazy-load a view's data the first time it's opened.
  const loaders = {
    Images: refreshImages,
    Volumes: refreshVolumes,
    Networks: refreshNetworks,
    Stacks: refreshStacks,
  }
  const loaded = new Set()
  $effect(() => {
    if (loaders[active] && !loaded.has(active)) {
      loaded.add(active)
      loaders[active]()
    }
  })

  // Two distinct truths, kept separate so the chrome never conflates them:
  //   Oriel (this app's backend) — reachable? status.error means it is not.
  //   colima (the VM) — running? only knowable when the backend is reachable.
  const gui = $derived(status.loading ? 'connecting' : status.error ? 'offline' : 'connected')
  const vm = $derived(
    status.loading ? 'checking' : status.error ? 'unknown' : status.running ? 'running' : 'stopped'
  )
  const guiMeta = {
    connecting: { word: 'connecting…', dot: 'bg-warn', text: 'text-warn' },
    connected: { word: 'connected', dot: 'bg-accent', text: 'text-accent' },
    offline: { word: 'offline', dot: 'bg-danger', text: 'text-danger' },
  }
  const vmMeta = {
    checking: { word: 'checking…', dot: 'bg-faint', text: 'text-faint' },
    unknown: { word: 'unknown', dot: 'bg-faint', text: 'text-faint' },
    running: { word: 'running', dot: 'bg-ok', text: 'text-ok' },
    stopped: { word: 'stopped', dot: 'bg-faint', text: 'text-muted' },
  }
  // The engine being driven: a colima VM, or a generic Docker daemon.
  const engineName = $derived(status.data?.engine === 'docker' ? 'docker' : 'colima')

  // Version label: real builds show "vX.Y.Z"; local builds show "dev" as-is.
  const verLabel = $derived(self.version || '')
</script>

<div class="flex h-screen w-screen overflow-hidden">
  <aside class="flex w-56 shrink-0 flex-col border-r border-border bg-surface">
    <div class="flex flex-col gap-2.5 px-5 pb-4 pt-5">
      <!-- Oriel: this app and its link to the backend -->
      <div class="flex items-center gap-2">
        <span class="h-2 w-2 shrink-0 rounded-full transition-colors {guiMeta[gui].dot}"></span>
        <span class="display text-sm font-semibold leading-none tracking-tight">Oriel</span>
        <span class="ml-auto text-[10px] font-medium uppercase tracking-[0.12em] transition-colors {guiMeta[gui].text}">{guiMeta[gui].word}</span>
      </div>
      <!-- colima: the virtual machine -->
      <div class="flex items-center gap-2">
        <span class="relative flex h-2 w-2 shrink-0">
          {#if vm === 'running'}
            <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-ok opacity-50"></span>
          {/if}
          <span class="relative inline-flex h-2 w-2 rounded-full transition-colors {vmMeta[vm].dot}"></span>
        </span>
        <span class="text-xs text-muted">{engineName}</span>
        <span class="ml-auto text-[10px] font-medium uppercase tracking-[0.12em] transition-colors {vmMeta[vm].text}">{vmMeta[vm].word}</span>
      </div>
    </div>
    <nav class="flex flex-1 flex-col gap-0.5 px-3 py-3">
      {#each nav as item}
        {@const on = active === item.name}
        <button
          class="navx group relative flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-left text-sm
            {on ? 'bg-surface-2 text-fg' : 'text-muted hover:bg-surface-2/50 hover:text-fg'}"
          onclick={() => (active = item.name)}
        >
          <span
            class="absolute left-0 top-1/2 h-4 w-[2.5px] -translate-y-1/2 rounded-full bg-accent transition-all duration-300
              {on
              ? 'opacity-100 shadow-[0_0_8px_0_var(--color-accent)]'
              : 'scale-y-50 opacity-0 group-hover:scale-y-100 group-hover:opacity-60'}"
          ></span>
          <Icon name={item.icon} size={17} class={on ? 'text-accent' : 'text-faint group-hover:text-fg'} />
          {item.name}
        </button>
      {/each}
      <div class="mx-2.5 my-1.5 border-t border-border"></div>
      <button
        class="navx group relative flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-left text-sm
          {active === 'Settings' ? 'bg-surface-2 text-fg' : 'text-muted hover:bg-surface-2/50 hover:text-fg'}"
        onclick={() => (active = 'Settings')}
      >
        <span
          class="absolute left-0 top-1/2 h-4 w-[2.5px] -translate-y-1/2 rounded-full bg-accent transition-all duration-300
            {active === 'Settings' ? 'opacity-100 shadow-[0_0_8px_0_var(--color-accent)]' : 'scale-y-50 opacity-0 group-hover:scale-y-100 group-hover:opacity-60'}"
        ></span>
        <Icon name="settings" size={17} class={active === 'Settings' ? 'text-accent' : 'text-faint group-hover:text-fg'} />
        Settings
      </button>
    </nav>
    <OpTray />
    <RecentOutages />

    <div class="flex items-center justify-between gap-2 border-t border-border px-4 py-3">
      <span class="text-[11px] font-medium tracking-wide text-muted">Oriel</span>
      <div class="flex items-center gap-1.5">
        {#if update.available}
          <button type="button" onclick={() => (active = 'Settings')} class="rounded-full bg-accent/15 px-2 py-0.5 font-mono text-[10px] font-medium text-accent hover:underline" title="Update available — v{update.latest} · open updates">update</button>
        {/if}
        {#if verLabel}<span class="rounded-full border border-border bg-surface-2 px-2 py-0.5 font-mono text-[10px] font-medium text-fg/85">{verLabel}</span>{/if}
      </div>
    </div>
  </aside>

  <div class="flex flex-1 flex-col overflow-hidden">
    <header class="flex h-14 shrink-0 items-center justify-between border-b border-border bg-surface px-6">
      <h1 class="display text-sm font-medium tracking-tight">{active}</h1>
      <div class="flex items-center gap-2.5 text-xs">
        <button
          class="pop group flex items-center gap-2 rounded-lg border border-border bg-surface/40 px-2.5 py-1.5 text-muted hover:border-accent/50 hover:text-fg"
          onclick={openPalette}
        >
          <Icon name="command" size={13} class="text-faint transition-colors group-hover:text-accent" />
          <span>Run command</span>
          <kbd class="rounded bg-surface-2 px-1.5 py-0.5 font-mono text-[10px] text-muted">⌘K</kbd>
        </button>
      </div>
    </header>

    <main class="min-h-0 flex-1 overflow-auto p-6">
      {#if active === 'Dashboard'}
        <Dashboard navigate={(v) => (active = v)} />
      {:else if active === 'Containers'}
        <Containers />
      {:else if active === 'Images'}
        <Images />
      {:else if active === 'Volumes'}
        <Volumes />
      {:else if active === 'Networks'}
        <Networks />
      {:else if active === 'Stacks'}
        <Stacks navigate={(v) => (active = v)} />
      {:else if active === 'Settings'}
        <Settings />
      {/if}
    </main>
  </div>
</div>
