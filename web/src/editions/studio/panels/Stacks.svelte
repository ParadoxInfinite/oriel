<script>
  import { stacks, refreshStacks, stackOp, confirm } from '../../../platform/index.js'
  import { discovery, ensureDiscovery, rescan, deployStack, confirmHide, AliasEditor, revealLabel, revealOrCopy } from '../../../platform/index.js'
  import Icon from '../lib/Icon.svelte'
  import StatusPill from '../lib/StatusPill.svelte'

  let { navigate } = $props()
  ensureDiscovery()
  const sorted = $derived([...stacks.list].sort((a, b) => a.name.localeCompare(b.name)))
  const hasRoots = $derived(discovery.config.roots.some((r) => r.enabled))

  // Per-stack collapse state, keyed by name (deep $state proxy → reactive).
  let collapsed = $state({})
  const toggle = (name) => (collapsed[name] = !collapsed[name])

  // Search box over the discovered (not-yet-deployed) list.
  let query = $state('')
  const discovered = $derived(
    discovery.stacks.filter((d) => {
      const q = query.trim().toLowerCase()
      if (!q) return true
      return [d.alias, d.name, d.dir].some((v) => (v || '').toLowerCase().includes(q))
    })
  )

  async function act(stack, action) {
    if (action === 'down') {
      const ok = await confirm({
        title: 'Take stack down?',
        message: `All ${stack.total} container(s) in “${stack.name}” will be stopped and removed.`,
        confirmLabel: 'Down',
      })
      if (!ok) return
    }
    stackOp(stack.name, action, refreshStacks)
  }

  const aliases = new AliasEditor()
</script>

<div class="mx-auto flex max-w-4xl flex-col gap-4">
  <div class="rise flex items-center gap-2.5">
    <span class="text-[13px] text-[var(--text-2)]"><span class="font-semibold text-[var(--text)]">{stacks.list.length}</span> compose stacks</span>
    <button class="btn btn-default ml-auto btn-sm" onclick={() => navigate?.('Settings', { kind: 'discovery' })}><Icon name="settings" size={13} /> Compose discovery</button>
  </div>

  {#if stacks.error}
    <div class="card p-4 text-sm text-[var(--red)]" style="border-color:color-mix(in srgb,var(--red) 40%,var(--border))">{stacks.error}</div>
  {:else if sorted.length === 0}
    <div class="card grid place-items-center gap-2 py-16 text-center">
      <Icon name="layers" size={26} class="text-[var(--text-3)]" />
      <p class="text-sm text-[var(--text-2)]">No running compose stacks.</p>
    </div>
  {:else}
    {#each sorted as s, i (s.name)}
      {@const allUp = s.running === s.total}
      {@const someUp = s.running > 0}
      {@const isOpen = !collapsed[s.name]}
      <div class="rise card overflow-hidden" style={`animation-delay:${i * 40}ms`}>
        <div class="flex flex-wrap items-center gap-3 px-5 py-3.5 {isOpen ? 'border-b border-[var(--border)]' : ''}">
          <button class="flex min-w-0 items-center gap-2.5 text-left" onclick={() => toggle(s.name)} aria-expanded={isOpen} title={isOpen ? 'Collapse' : 'Expand'}>
            <Icon name="chevron" size={16} class="shrink-0 text-[var(--text-3)] transition-transform {isOpen ? '' : '-rotate-90'}" />
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="text-[14px] font-semibold tracking-tight">{aliases.display(s.name)}</span>
                {#if aliases.display(s.name) !== s.name}<span class="mono text-[11px] text-[var(--text-3)]">({s.name})</span>{/if}
                <span class="pill {allUp ? 'on' : someUp ? 'warn' : 'off'}"><span class="dot"></span>{s.running}/{s.total} up</span>
              </div>
              <div class="mono mt-0.5 truncate text-xs text-[var(--text-3)]">{s.workingDir}</div>
            </div>
          </button>
          {#if aliases.editing === s.name}
            <div class="flex items-center gap-1.5">
              <input bind:value={aliases.draft} placeholder={s.name} class="input py-1 text-[13px]" onkeydown={(e) => { if (e.key === 'Enter') aliases.save(); if (e.key === 'Escape') aliases.cancel() }} />
              <button class="btn btn-primary btn-sm" onclick={() => aliases.save()}>Save</button>
              <button class="btn btn-ghost btn-sm" onclick={() => aliases.cancel()}>Cancel</button>
            </div>
          {:else}
            <button class="text-[var(--text-3)] hover:text-[var(--accent)]" title="Rename in Oriel" aria-label="Rename" onclick={() => aliases.start(s.name)}>
              <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9" /><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z" /></svg>
            </button>
          {/if}
          <div class="ml-auto flex gap-1.5">
            {#if s.running < s.total}<button class="btn btn-default btn-sm" onclick={() => act(s, 'start')}><Icon name="play" size={13} /> Start</button>{/if}
            {#if someUp}<button class="btn btn-default btn-sm" onclick={() => act(s, 'stop')}><Icon name="stop" size={13} /> Stop</button>{/if}
            <button class="btn btn-default btn-sm" onclick={() => act(s, 'restart')}><Icon name="restart" size={13} /> Restart</button>
            <button class="btn btn-danger btn-sm" onclick={() => act(s, 'down')}><Icon name="trash" size={13} /> Down</button>
          </div>
        </div>
        {#if isOpen}
          <div class="grid gap-x-8 gap-y-2 px-5 py-3.5 sm:grid-cols-2">
            {#each s.containers as c (c.id)}
              <div class="flex items-center justify-between gap-2">
                <span class="mono truncate text-[12.5px] text-[var(--text-2)]">{c.name}</span>
                <StatusPill state={c.state} />
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/each}
  {/if}

  <!-- Available (discovered, not deployed) -->
  <div class="rise mt-2 flex flex-wrap items-center gap-2.5" style="animation-delay:60ms">
    <span class="eyebrow">Available</span>
    {#if discovery.stacks.length}<span class="count">{discovery.stacks.length}</span>{/if}
    {#if discovery.stacks.length > 1}
      <div class="relative ml-auto">
        <Icon name="search" size={14} class="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-[var(--text-3)]" />
        <input bind:value={query} placeholder="Search available…" class="input has-icon btn-sm w-56" />
      </div>
    {/if}
    <button class="btn btn-ghost btn-sm {discovery.stacks.length > 1 ? '' : 'ml-auto'}" onclick={rescan} disabled={discovery.loading}><Icon name="restart" size={13} /> {discovery.loading ? 'Scanning…' : 'Rescan'}</button>
  </div>

  {#if !hasRoots}
    <div class="card flex items-center gap-3 px-5 py-4 text-[13px] text-[var(--text-2)]">
      <Icon name="settings" size={16} class="shrink-0 text-[var(--text-3)]" />
      Add directories in <span class="font-medium text-[var(--text)]">Settings → Compose discovery</span> to find projects you can deploy.
    </div>
  {:else if discovery.stacks.length === 0}
    <div class="card px-5 py-4 text-[13px] text-[var(--text-3)]">
      Nothing new to deploy, everything discovered is already running.{#if discovery.hidden}<span> {discovery.hidden} hidden by filter.</span>{/if}
    </div>
  {:else if discovered.length === 0}
    <div class="card px-5 py-4 text-[13px] text-[var(--text-3)]">No available stacks match “{query}”.</div>
  {:else}
    {#each discovered as d (d.file)}
      <div class="card flex flex-wrap items-center gap-3 px-5 py-3.5">
        <div class="min-w-0 flex-1">
          {#if aliases.editing === d.name}
            <div class="flex items-center gap-2">
              <input bind:value={aliases.draft} placeholder={d.name} class="input py-1 text-[13px]" onkeydown={(e) => { if (e.key === 'Enter') aliases.save(); if (e.key === 'Escape') aliases.cancel() }} />
              <button class="btn btn-default btn-sm" onclick={() => aliases.save()}>Save</button>
              <button class="btn btn-ghost btn-sm" onclick={() => aliases.cancel()}>Cancel</button>
            </div>
          {:else}
            <div class="flex items-center gap-1.5">
              <span class="text-[14px] font-semibold tracking-tight">{aliases.display(d.name)}</span>
              {#if aliases.display(d.name) !== d.name}<span class="mono text-[11px] text-[var(--text-3)]">({d.name})</span>{/if}
              <button class="text-[var(--text-3)] hover:text-[var(--accent)]" title="Rename in Oriel" aria-label="Rename" onclick={() => aliases.start(d.name)}>
                <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9" /><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z" /></svg>
              </button>
            </div>
          {/if}
          <div class="mono mt-0.5 truncate text-xs text-[var(--text-3)]">{d.dir} · {d.services >= 0 ? `${d.services} service${d.services === 1 ? '' : 's'}` : 'compose'}</div>
        </div>
        <div class="flex shrink-0 gap-1.5">
          <button class="btn btn-primary btn-sm" onclick={() => deployStack(d)}><Icon name="play" size={13} /> Up</button>
          <button class="btn btn-default btn-icon btn-sm" title={revealLabel()} aria-label={revealLabel()} onclick={() => revealOrCopy(d.dir)}>
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z" /></svg>
          </button>
          {#if discovery.config.filter.mode !== 'allow'}
            <button class="btn btn-default btn-icon btn-sm" title="Hide from this list" aria-label="Hide" onclick={() => confirmHide(d)}>
              <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M9.9 4.24A9.1 9.1 0 0 1 12 4c7 0 10 8 10 8a18 18 0 0 1-2.16 3.19M6.6 6.6A18 18 0 0 0 2 12s3 8 10 8a9.3 9.3 0 0 0 5.4-1.6" /><path d="m2 2 20 20" /></svg>
            </button>
          {/if}
        </div>
      </div>
    {/each}
    {#if discovery.hidden}<p class="text-center text-[11px] text-[var(--text-3)]">{discovery.hidden} hidden by filter · manage in Settings</p>{/if}
  {/if}
</div>
