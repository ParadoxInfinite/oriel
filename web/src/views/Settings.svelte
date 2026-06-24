<script>
  import { onMount } from 'svelte'
  import {
    self, update, checkNow, restartService, promptUpdate,
    remote, loadRemote, removeRemoteHost, RemoteHostForm,
    grant, loadGrant, requestGrant, lockGrant, fmtRemaining,
    discovery, ensureDiscovery, updateRoot, removeRoot, rootResult, setFilter, removePattern, FILTER_MODES, DiscoveryForm,
    THEMES_DOC_URL,
  } from '../platform/index.js'
  import { editions, edition, setEdition, diskThemes } from '../editions/registry.svelte.js'
  import { btn, btnPrimary } from '../lib/ui.js'
  import Icon from '../components/Icon.svelte'
  import PathInput from '../components/PathInput.svelte'

  ensureDiscovery()
  const df = new DiscoveryForm()
  const hostForm = new RemoteHostForm()
  onMount(() => {
    loadRemote()
    loadGrant()
  })

  const field =
    'w-full rounded-[--radius] border border-border bg-bg px-3 py-1.5 text-sm outline-none placeholder:text-muted focus:border-accent/50'

  const verLabel = $derived(self.version || '—')
</script>

<div class="mx-auto grid max-w-5xl grid-cols-1 gap-5 pb-4 lg:grid-cols-2 lg:items-start">
  <div class="flex flex-col gap-5">
  <!-- Editions & themes -->
  <section class="card rounded-[--radius] p-5">
    <h2 class="display text-sm font-semibold tracking-tight">Editions &amp; themes</h2>
    <p class="mt-0.5 text-xs text-muted">Switch the whole interface, or load an external theme bundle.</p>

    <div class="mt-4 grid gap-2 sm:grid-cols-2">
      {#each editions() as e (e.id)}
        {@const on = edition.active === e.id}
        <button
          class="pop flex items-center gap-3 rounded-[--radius] border p-3 text-left {on ? 'border-accent/60 bg-accent/10' : 'border-border bg-surface hover:border-border-light'}"
          onclick={() => setEdition(e.id)}
        >
          <span class="h-3 w-3 shrink-0 rounded-full" style="background:{e.accent};box-shadow:0 0 8px -1px {e.accent}"></span>
          <span class="min-w-0 flex-1">
            <span class="block text-[13px] font-medium text-fg">{e.name}{#if e.external}<span class="ml-1.5 text-[10px] font-normal text-faint">external</span>{/if}</span>
            <span class="block truncate text-xs text-muted">{e.tagline}</span>
          </span>
          {#if on}<span class="text-[10px] uppercase tracking-wider text-accent">active</span>{/if}
        </button>
      {/each}
    </div>

    <div class="mt-5 border-t border-border pt-4">
      <div class="flex items-center justify-between gap-2">
        <span class="text-[13px] font-medium text-fg">Installed themes</span>
        <a href={THEMES_DOC_URL} target="_blank" rel="noopener" class="text-[11.5px] font-medium text-accent hover:underline">Build your own ↗</a>
      </div>
      <p class="mt-0.5 text-xs text-faint">Drop a theme bundle (an ES module default-exporting <span class="font-mono">{'{ id, name, component }'}</span>) into:</p>
      {#if diskThemes.dir}<p class="mt-1 break-all rounded-[--radius] bg-surface px-2.5 py-1.5 font-mono text-[11.5px] text-muted">{diskThemes.dir}</p>{/if}
      {#if diskThemes.list.length}
        <div class="mt-3 flex flex-col gap-1.5">
          {#each diskThemes.list as t (t.id)}
            <div class="flex items-center gap-2 rounded-[--radius] border border-border bg-surface px-3 py-2">
              <span class="h-2 w-2 shrink-0 rounded-full bg-ok"></span>
              <span class="min-w-0 flex-1 truncate text-[13px] text-muted">{t.name}</span>
            </div>
          {/each}
        </div>
      {:else}
        <p class="mt-2 text-xs text-faint">No themes installed yet.</p>
      {/if}
      {#each Object.entries(diskThemes.errors) as [file, err] (file)}
        <p class="mt-2 text-xs text-danger"><span class="font-mono">{file}</span>: {err}</p>
      {/each}
    </div>
  </section>

  <!-- Compose discovery -->
  <section class="card rounded-[--radius] p-5">
    <h2 class="display text-sm font-semibold tracking-tight">Compose discovery</h2>
    <p class="mt-0.5 text-xs text-muted">Find Docker Compose projects on disk so you can deploy them from the Stacks tab.</p>

    {#if discovery.config.roots.length}
      <div class="mt-4 flex flex-col gap-2">
        {#each discovery.config.roots as root (root.id)}
          {@const rr = rootResult(root.id)}
          <div class="flex items-center gap-3 rounded-[--radius] border border-border bg-surface px-3 py-2.5">
            <input type="checkbox" checked={root.enabled} onchange={() => updateRoot(root.id, { enabled: !root.enabled })} class="h-4 w-4 shrink-0 accent-accent" title="Enabled" />
            <div class="min-w-0 flex-1">
              <div class="truncate font-mono text-[13px] {root.enabled ? 'text-fg' : 'text-faint line-through'}">{root.path}</div>
              <div class="mt-0.5 text-[11px]">
                {#if rr?.error}<span class="text-danger">{rr.error}</span>
                {:else if root.enabled}<span class="text-faint">{rr?.found ?? 0} project{(rr?.found ?? 0) === 1 ? '' : 's'}</span>
                {:else}<span class="text-faint">disabled</span>{/if}
              </div>
            </div>
            <label class="flex shrink-0 cursor-pointer items-center gap-1.5 text-xs text-muted" title="Walk subdirectories recursively">
              <input type="checkbox" checked={root.traverse} onchange={() => updateRoot(root.id, { traverse: !root.traverse })} class="h-3.5 w-3.5 accent-accent" /> Traverse
            </label>
            <button class="pop rounded p-1.5 text-faint hover:text-danger" title="Remove" aria-label="Remove directory" onclick={() => removeRoot(root.id)}><Icon name="trash" size={14} /></button>
          </div>
        {/each}
      </div>
    {/if}

    <div class="mt-3 flex gap-2">
      <PathInput field={df.pathField} onEnter={() => df.addDir()} placeholder="Add a directory…  /Users/you/projects" />
      <button class={btn} onclick={() => df.addDir()} disabled={!df.pathField.value.trim()}>Add</button>
    </div>
    <p class="mt-1.5 text-[11px] text-faint">Turn on <span class="font-medium text-muted">Traverse</span> to scan subdirectories; off treats the directory itself as one project.</p>

    <div class="mt-5 border-t border-border pt-4">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <span class="text-[13px] font-medium text-fg">Filter <span class="font-normal text-faint">· discovered stacks only</span></span>
        <div class="flex gap-1 rounded-lg bg-surface-2 p-1">
          {#each FILTER_MODES as [m, label]}
            <button class="pop rounded-md px-2.5 py-1 text-xs {discovery.config.filter.mode === m ? 'bg-surface text-fg shadow-sm' : 'text-muted hover:text-fg'}" onclick={() => setFilter({ mode: m })}>{label}</button>
          {/each}
        </div>
      </div>
      {#if discovery.config.filter.mode !== 'off'}
        <div class="mt-3 flex flex-wrap gap-1.5">
          {#each discovery.config.filter.patterns as p (p)}
            <span class="inline-flex items-center gap-1 rounded-md border border-border bg-surface px-2 py-1 font-mono text-[11.5px] text-muted">{p}<button class="text-faint hover:text-danger" aria-label="Remove pattern" onclick={() => removePattern(p)}>×</button></span>
          {/each}
        </div>
        <div class="mt-2 flex gap-2">
          <input bind:value={df.pattern} placeholder="web-*  ·  My App  ·  ~/lab/**" class="{field} font-mono" onkeydown={(e) => e.key === 'Enter' && df.addPattern()} />
          <button class={btn} onclick={() => df.addPattern()} disabled={!df.pattern.trim()}>Add</button>
        </div>
        <p class="mt-1.5 text-[11px] text-faint">Matches a project name, its Oriel name, or a directory path (globs &amp; <span class="font-mono">**</span> allowed). Running stacks are never hidden.</p>
      {/if}
    </div>
  </section>
  </div>

  <div class="flex flex-col gap-5">
  <!-- Automation access (destructive grant) -->
  <section class="card rounded-[--radius] p-5">
    <h2 class="display text-sm font-semibold tracking-tight">Automation access</h2>
    <p class="mt-1 text-sm text-muted">
      The MCP server (<span class="mono">oriel mcp</span>) can run read actions any time. Destructive ones (remove, prune) stay locked until you open a time-boxed window — your own UI clicks are never affected.
    </p>
    <div class="mt-4 flex flex-wrap items-center gap-2">
      {#if grant.active}
        <span class="rounded-full bg-accent/15 px-2.5 py-1 text-xs font-medium text-accent">Unlocked · {fmtRemaining(grant.remainingSeconds)} left</span>
        <button class={btn} onclick={() => requestGrant(6)} disabled={grant.busy}>Extend 6h</button>
        <button class={btn} onclick={() => lockGrant()} disabled={grant.busy}>Lock now</button>
      {:else}
        <span class="text-sm text-faint">Destructive actions are <span class="font-medium text-fg">locked</span> for automation.</span>
        <button class={btnPrimary} onclick={() => requestGrant(6)} disabled={grant.busy}>Allow 6h</button>
        <button class={btn} onclick={() => requestGrant(24 * 6)} disabled={grant.busy}>Allow 6d</button>
      {/if}
    </div>
    <p class="mt-2 text-xs text-faint">Same window the <span class="mono">oriel ai allow-destructive</span> CLI opens. Auto-relocks when it lapses.</p>
  </section>

  <!-- Updates -->
  <section class="card rounded-[--radius] p-5">
    <h2 class="display text-sm font-semibold tracking-tight">Updates</h2>
    <p class="mt-0.5 text-xs text-muted">Current version <span class="font-mono text-fg/85">{verLabel}</span>.</p>

    <div class="mt-4">
      {#if update.phase === 'restarting'}
        <p class="text-[13px] text-muted">Restarting Oriel — this page will reconnect…</p>
      {:else if update.phase === 'applying'}
        <p class="text-[13px] text-muted">Downloading &amp; verifying…</p>
      {:else if update.phase === 'done'}
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="text-[13px] text-muted">Installed <span class="font-mono">v{update.latest}</span> — restart to apply.</span>
          <button class={btnPrimary} onclick={() => restartService()}>Restart now</button>
        </div>
      {:else if !update.managed}
        <p class="text-[13px] text-faint">In-app updates need a service install (<span class="font-mono">oriel service install</span>).{#if update.available}{' '}A new version <span class="font-mono">v{update.latest}</span> is out — <a href={update.url} target="_blank" rel="noopener" class="text-accent hover:underline">see release ↗</a>.{/if}</p>
      {:else if update.available}
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="text-[13px] text-muted">Update available: <span class="font-mono font-medium text-fg">v{update.latest}</span></span>
          <button class={btnPrimary} onclick={promptUpdate}>Update now</button>
        </div>
      {:else}
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="text-[13px] text-faint">{update.checking ? 'Checking…' : "You're on the latest version."}</span>
          <div class="flex items-center gap-3">
            <button class={btn} onclick={() => checkNow()} disabled={update.checking}>{update.checking ? 'Checking…' : 'Check for updates'}</button>
            <a href={update.url || 'https://github.com/ParadoxInfinite/oriel/releases'} target="_blank" rel="noopener" class="text-xs text-accent hover:underline">Releases ↗</a>
          </div>
        </div>
      {/if}
      {#if update.error}<p class="mt-2 text-xs text-danger">{update.error}</p>{/if}
    </div>
  </section>

  <!-- Remote access -->
  <section class="card rounded-[--radius] p-5">
    <h2 class="display text-sm font-semibold tracking-tight">Remote access</h2>
    <p class="mt-0.5 text-xs text-muted">By default Oriel only answers on <span class="font-mono">localhost</span>. To reach it over a private network (Tailscale, a reverse proxy, a domain), add those hostnames.</p>
    <p class="mt-2 rounded-[--radius] bg-danger/10 px-3 py-2 text-xs text-danger">Oriel has no login and controls Docker. Only add hosts you reach over a trusted private network — never the public internet.</p>

    {#if remote.hosts.length}
      <div class="mt-3 flex flex-col gap-1.5">
        {#each remote.hosts as h (h)}
          <div class="flex items-center gap-2 rounded-[--radius] border border-border bg-surface px-3 py-2">
            <span class="min-w-0 flex-1 truncate font-mono text-[13px] text-muted">{h}</span>
            <button class="pop rounded p-1 text-faint hover:text-danger" aria-label="Remove host" onclick={() => removeRemoteHost(h)}><Icon name="trash" size={13} /></button>
          </div>
        {/each}
      </div>
    {/if}
    <div class="mt-3 flex gap-2">
      <input bind:value={hostForm.draft} placeholder="oriel.example.com" class={field} onkeydown={(e) => e.key === 'Enter' && hostForm.add()} />
      <button class={btn} onclick={() => hostForm.add()} disabled={!hostForm.draft.trim() || remote.saving}>Add</button>
    </div>
    {#if remote.error}<p class="mt-2 text-xs text-danger">{remote.error}</p>{/if}
  </section>
  </div>
</div>
