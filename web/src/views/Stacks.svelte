<script>
  import {
    stacks, refreshStacks, stackOp, confirm,
    discovery, ensureDiscovery, rescan, deployStack, confirmHide, revealLabel, revealOrCopy, AliasEditor,
  } from '../platform/index.js'
  import { action } from '../lib/ui.js'
  import StateBadge from '../components/StateBadge.svelte'
  import Icon from '../components/Icon.svelte'

  let { navigate } = $props()
  ensureDiscovery()
  const sorted = $derived([...stacks.list].sort((a, b) => a.name.localeCompare(b.name)))
  const hasRoots = $derived(discovery.config.roots.some((r) => r.enabled))
  const aliases = new AliasEditor()

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
</script>

<div class="flex flex-col">
  <div class="mb-4 flex items-center gap-2.5">
    <span class="text-xs text-muted">{stacks.list.length} stacks</span>
    <button class="pop ml-auto flex items-center gap-1.5 rounded-lg border border-border bg-surface/40 px-2.5 py-1.5 text-xs text-muted hover:border-accent/50 hover:text-fg" onclick={() => navigate?.('Settings')}><Icon name="settings" size={13} /> Compose discovery</button>
  </div>

  {#if stacks.error}
    <div class="rounded-[--radius] border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">{stacks.error}</div>
  {:else if sorted.length === 0}
    <div class="rounded-[--radius] border border-dashed border-border py-20 text-center text-sm text-muted">
      No compose stacks
    </div>
  {:else}
    <div class="flex flex-col gap-3">
      {#each sorted as s (s.name)}
        {@const allRunning = s.running === s.total}
        <div class="lift rounded-[--radius] border border-border bg-surface">
          <div class="flex items-center justify-between border-b border-border px-4 py-3">
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                {#if aliases.editing === s.name}
                  <input bind:value={aliases.draft} placeholder={s.name} class="rounded-[--radius] border border-border bg-bg px-2 py-0.5 font-mono text-sm outline-none focus:border-accent/50" onkeydown={(e) => { if (e.key === 'Enter') aliases.save(); if (e.key === 'Escape') aliases.cancel() }} />
                  <button class={action('accent')} onclick={() => aliases.save()}>Save</button>
                  <button class="text-xs text-muted hover:text-fg" onclick={() => aliases.cancel()}>Cancel</button>
                {:else}
                  <span class="font-mono text-sm">{aliases.display(s.name)}</span>
                  {#if aliases.display(s.name) !== s.name}<span class="font-mono text-[11px] text-faint">({s.name})</span>{/if}
                  <button class="text-faint hover:text-accent" title="Rename in Oriel" aria-label="Rename" onclick={() => aliases.start(s.name)}>
                    <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9" /><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z" /></svg>
                  </button>
                  <span class="rounded-full px-2 py-0.5 text-[11px] {allRunning ? 'bg-ok/15 text-ok' : s.running === 0 ? 'bg-surface-2 text-muted' : 'bg-warn/15 text-warn'}">
                    {s.running}/{s.total} running
                  </span>
                {/if}
              </div>
              <div class="mt-0.5 truncate font-mono text-xs text-muted">{s.workingDir}</div>
            </div>
            <div class="flex shrink-0 gap-1">
              {#if s.running < s.total}<button class={action('ok')} onclick={() => act(s, 'start')}>Start</button>{/if}
              {#if s.running > 0}<button class={action('warn')} onclick={() => act(s, 'stop')}>Stop</button>{/if}
              <button class={action('accent')} onclick={() => act(s, 'restart')}>Restart</button>
              <button class={action('danger')} onclick={() => act(s, 'down')}>Down</button>
            </div>
          </div>
          <div class="grid grid-cols-1 gap-x-6 gap-y-1.5 px-4 py-3 sm:grid-cols-2">
            {#each s.containers as c (c.id)}
              <div class="flex items-center justify-between gap-2">
                <span class="truncate font-mono text-xs">{c.name}</span>
                <StateBadge state={c.state} />
              </div>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  {/if}

  <!-- Available (discovered, not deployed) -->
  <div class="mb-2 mt-6 flex items-center gap-2.5">
    <span class="text-[10px] font-medium uppercase tracking-[0.15em] text-faint">Available</span>
    {#if discovery.stacks.length}<span class="rounded-full bg-surface-2 px-1.5 text-[10px] text-muted">{discovery.stacks.length}</span>{/if}
    <button class="pop ml-auto flex items-center gap-1.5 rounded-md px-2 py-1 text-xs text-muted hover:text-fg" onclick={rescan} disabled={discovery.loading}><Icon name="restart" size={13} /> {discovery.loading ? 'Scanning…' : 'Rescan'}</button>
  </div>

  {#if !hasRoots}
    <div class="flex items-center gap-3 rounded-[--radius] border border-border bg-surface px-4 py-3 text-[13px] text-muted">
      <Icon name="settings" size={16} class="shrink-0 text-faint" />
      Add directories in <span class="font-medium text-fg">Settings → Compose discovery</span> to find projects you can deploy.
    </div>
  {:else if discovery.stacks.length === 0}
    <div class="rounded-[--radius] border border-border bg-surface px-4 py-3 text-[13px] text-faint">
      Nothing new to deploy — everything discovered is already running.{#if discovery.hidden}<span> {discovery.hidden} hidden by filter.</span>{/if}
    </div>
  {:else}
    <div class="flex flex-col gap-2">
      {#each discovery.stacks as d (d.file)}
        <div class="lift flex flex-wrap items-center gap-3 rounded-[--radius] border border-border bg-surface px-4 py-3">
          <div class="min-w-0 flex-1">
            {#if aliases.editing === d.name}
              <div class="flex items-center gap-2">
                <input bind:value={aliases.draft} placeholder={d.name} class="rounded-[--radius] border border-border bg-bg px-2 py-1 text-sm outline-none focus:border-accent/50" onkeydown={(e) => { if (e.key === 'Enter') aliases.save(); if (e.key === 'Escape') aliases.cancel() }} />
                <button class={action('accent')} onclick={() => aliases.save()}>Save</button>
                <button class="text-xs text-muted hover:text-fg" onclick={() => aliases.cancel()}>Cancel</button>
              </div>
            {:else}
              <div class="flex items-center gap-1.5">
                <span class="font-mono text-sm text-fg">{aliases.display(d.name)}</span>
                {#if aliases.display(d.name) !== d.name}<span class="font-mono text-[11px] text-faint">({d.name})</span>{/if}
                <button class="text-faint hover:text-accent" title="Rename in Oriel" aria-label="Rename" onclick={() => aliases.start(d.name)}>
                  <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9" /><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z" /></svg>
                </button>
              </div>
            {/if}
            <div class="mt-0.5 truncate font-mono text-xs text-faint">{d.dir} · {d.services >= 0 ? `${d.services} service${d.services === 1 ? '' : 's'}` : 'compose'}</div>
          </div>
          <div class="flex shrink-0 items-center gap-1.5">
            <button class={action('ok')} onclick={() => deployStack(d)}>Up</button>
            <button class="pop rounded p-1.5 text-faint hover:text-fg" title={revealLabel()} aria-label={revealLabel()} onclick={() => revealOrCopy(d.dir)}>
              <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z" /></svg>
            </button>
            {#if discovery.config.filter.mode !== 'allow'}
              <button class="pop rounded p-1.5 text-faint hover:text-fg" title="Hide from this list" aria-label="Hide" onclick={() => confirmHide(d)}>
                <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M9.9 4.24A9.1 9.1 0 0 1 12 4c7 0 10 8 10 8a18 18 0 0 1-2.16 3.19M6.6 6.6A18 18 0 0 0 2 12s3 8 10 8a9.3 9.3 0 0 0 5.4-1.6" /><path d="m2 2 20 20" /></svg>
              </button>
            {/if}
          </div>
        </div>
      {/each}
    </div>
    {#if discovery.hidden}<p class="mt-2 text-center text-[11px] text-faint">{discovery.hidden} hidden by filter · manage in Settings</p>{/if}
  {/if}
</div>
