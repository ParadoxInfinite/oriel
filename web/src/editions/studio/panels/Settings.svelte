<script>
  import { onMount } from 'svelte'
  import { provider, checkProvider, setProvider, resolveText, toast } from '../../../platform/index.js'
  import { discovery, ensureDiscovery, addRoot, updateRoot, removeRoot, rootResult, setFilter, addPattern, removePattern, PathField, THEMES_DOC_URL } from '../../../platform/index.js'
  import { self, update, checkNow, applyUpdate, restartService, confirm, apiPut } from '../../../platform/index.js'
  import { remote, loadRemote, addRemoteHost, removeRemoteHost } from '../../../platform/index.js'
  import { editions, edition, setEdition, diskThemes } from '../../../editions/registry.svelte.js'
  import { appearance, systemPref, ACCENTS, setMode, setAccent, addCustomAccent, removeCustomAccent } from '../theme.svelte.js'
  import Icon from '../lib/Icon.svelte'
  import PathInput from '../lib/PathInput.svelte'

  ensureDiscovery()
  const pf = new PathField()
  // Secret-masking + reveal policy (Settings → Secrets), saved via /api/config.
  async function saveSecret(patch) {
    const prev = { maskEnv: self.maskEnv, envReveal: self.envReveal }
    Object.assign(self, patch) // optimistic
    try {
      const d = await apiPut('/api/config', patch)
      if (d?.maskEnv) self.maskEnv = d.maskEnv
      if (d?.envReveal) self.envReveal = d.envReveal
    } catch (e) {
      Object.assign(self, prev)
      toast(e?.message || 'Could not save', 'error')
    }
  }
  function addDir() {
    if (pf.value.trim()) {
      addRoot(pf.value)
      pf.reset()
    }
  }
  let newPattern = $state('')
  function addPat() {
    addPattern(newPattern)
    newPattern = ''
  }
  const FILTER_MODES = [
    ['off', 'Off'],
    ['allow', 'Allow-list'],
    ['deny', 'Deny-list'],
  ]

  let hostDraft = $state('')
  onMount(loadRemote)
  function addHost() {
    const h = hostDraft.trim()
    if (h) {
      addRemoteHost(h)
      hostDraft = ''
    }
  }

  // Accent is per-theme; reflect the base that's actually showing.
  const base = $derived(appearance.mode === 'system' ? (systemPref.dark ? 'dark' : 'light') : appearance.mode)
  const activeAccent = $derived(appearance.accents[base])

  const MODES = [
    { id: 'light', label: 'Light' },
    { id: 'dark', label: 'Dark' },
    { id: 'system', label: 'System' },
  ]
  const swatches = $derived([...ACCENTS, ...appearance.custom])

  const verLabel = $derived(self.version || '—')
  async function doUpdate() {
    const res = await confirm({
      title: 'Update Oriel?',
      message: `Download v${update.latest}, verify its checksum, and replace the binary. Oriel must restart to apply.`,
      confirmLabel: 'Update',
      danger: false,
      checkbox: 'Restart automatically when done',
      checked: true,
    })
    if (!res || !res.ok) return
    const ok = await applyUpdate()
    if (ok && res.checked) await restartService()
  }

  // Custom accent form.
  let newColor = $state('#22c55e')
  let newName = $state('')
  function addAccent() {
    addCustomAccent(newName, newColor)
    newName = ''
  }


  // AI provider.
  let urlDraft = $state('')
  let testText = $state('')
  let testBusy = $state(false)
  let testErr = $state('')
  let testResult = $state(null)
  onMount(async () => {
    await checkProvider()
    urlDraft = provider.url
  })
  async function saveProvider() {
    try {
      await setProvider(urlDraft.trim())
      toast(urlDraft.trim() ? 'Provider connected' : 'Provider disabled', 'ok')
    } catch (e) {
      toast(e.message, 'error')
    }
  }
  async function runTest() {
    if (!testText.trim()) return
    testBusy = true
    testErr = ''
    testResult = null
    try {
      testResult = await resolveText(testText.trim())
    } catch (e) {
      testErr = e.message
    } finally {
      testBusy = false
    }
  }
</script>

<div class="mx-auto grid max-w-5xl grid-cols-1 gap-4 pb-4 md:grid-cols-2 md:items-start">
  <div class="flex flex-col gap-4">
  <!-- Appearance -->
  <section class="rise card p-5">
    <h2 class="text-[14px] font-semibold tracking-tight">Appearance</h2>
    <p class="mt-0.5 text-[13px] text-[var(--text-2)]">How Studio looks. Saved to this browser.</p>

    <div class="mt-4 flex flex-wrap items-center justify-between gap-3">
      <span class="text-[13px] font-medium">Theme</span>
      <div class="seg">
        {#each MODES as m}
          <button class="seg-btn {appearance.mode === m.id ? 'on' : ''}" onclick={() => setMode(m.id)}>{m.label}</button>
        {/each}
      </div>
    </div>

    <div class="mt-5 border-t border-[var(--border)] pt-4">
      <div class="flex items-center justify-between">
        <span class="text-[13px] font-medium">Accent <span class="font-normal text-[var(--text-3)]">· {base} theme</span></span>
        <span class="mono text-[11px] text-[var(--text-3)]">{activeAccent}</span>
      </div>
      <div class="mt-3 flex flex-wrap items-center gap-2.5">
        {#each swatches as sw (sw.id)}
          <div class="group relative">
            <button
              class="swatch {activeAccent === sw.value ? 'on' : ''}"
              style="background:{sw.value};color:{sw.value}"
              title={sw.name}
              aria-label={sw.name}
              onclick={() => setAccent(sw.value)}
            ></button>
            {#if sw.id?.startsWith('custom-')}
              <button class="absolute -right-1.5 -top-1.5 grid h-4 w-4 place-items-center rounded-full border border-[var(--border)] bg-[var(--panel)] text-[var(--text-3)] opacity-0 shadow-[var(--shadow-sm)] transition-opacity hover:text-[var(--red)] group-hover:opacity-100" title="Remove" aria-label="Remove accent" onclick={() => removeCustomAccent(sw.id)}>
                <svg viewBox="0 0 24 24" width="9" height="9" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><path d="M18 6 6 18M6 6l12 12" /></svg>
              </button>
            {/if}
          </div>
        {/each}
      </div>

      <div class="mt-4 flex flex-wrap items-end gap-2.5">
        <label class="flex items-center gap-2">
          <input type="color" bind:value={newColor} class="h-9 w-9 cursor-pointer rounded-lg border border-[var(--border-strong)] bg-transparent p-0.5" />
        </label>
        <input bind:value={newName} placeholder="Name (optional)" class="input w-44" />
        <button class="btn btn-default btn-sm" onclick={addAccent}><Icon name="sparkles" size={14} /> Add accent</button>
      </div>
    </div>
  </section>

  <!-- Editions & themes -->
  <section class="rise card p-5" style="animation-delay:40ms">
    <h2 class="text-[14px] font-semibold tracking-tight">Editions &amp; themes</h2>
    <p class="mt-0.5 text-[13px] text-[var(--text-2)]">Switch the whole interface, or drop a theme bundle on disk.</p>

    <div class="mt-4 grid gap-2 sm:grid-cols-2">
      {#each editions() as e (e.id)}
        {@const on = edition.active === e.id}
        <button class="flex items-center gap-3 rounded-lg border p-3 text-left transition-colors {on ? 'border-[var(--accent)] bg-[var(--accent-tint)]' : 'border-[var(--border)] hover:border-[var(--border-strong)] hover:bg-[var(--panel-2)]'}" onclick={() => setEdition(e.id)}>
          <span class="h-3 w-3 shrink-0 rounded-full" style="background:{e.accent};box-shadow:0 0 8px -1px {e.accent}"></span>
          <span class="min-w-0 flex-1">
            <span class="block text-[13px] font-medium">{e.name}{#if e.external}<span class="ml-1.5 text-[10px] font-normal text-[var(--text-3)]">installed</span>{/if}</span>
            <span class="block truncate text-[12px] text-[var(--text-3)]">{e.tagline}</span>
          </span>
          {#if on}<span class="text-[var(--accent)]"><Icon name="play" size={13} /></span>{/if}
        </button>
      {/each}
    </div>

    <div class="mt-5 border-t border-[var(--border)] pt-4">
      <div class="flex items-center justify-between gap-2">
        <span class="text-[13px] font-medium">Installed themes</span>
        <a href={THEMES_DOC_URL} target="_blank" rel="noopener" class="text-[11.5px] font-medium text-[var(--accent)] hover:underline">Build your own ↗</a>
      </div>
      <p class="mt-0.5 text-[12px] text-[var(--text-3)]">Drop a theme bundle (an ES module default-exporting <span class="mono">{'{ id, name, component }'}</span>) into:</p>
      {#if diskThemes.dir}<p class="mono mt-1 break-all rounded-lg bg-[var(--panel-2)] px-2.5 py-1.5 text-[11.5px] text-[var(--text-2)]">{diskThemes.dir}</p>{/if}
      {#if diskThemes.list.length}
        <div class="mt-3 flex flex-col gap-1.5">
          {#each diskThemes.list as t (t.id)}
            <div class="flex items-center gap-2 rounded-lg border border-[var(--border)] bg-[var(--panel-2)] px-3 py-2">
              <span class="h-2 w-2 shrink-0 rounded-full" style="background:var(--green)"></span>
              <span class="min-w-0 flex-1 truncate text-[12.5px] text-[var(--text-2)]">{t.name}</span>
            </div>
          {/each}
        </div>
      {:else}
        <p class="mt-2 text-[12px] text-[var(--text-3)]">No themes installed yet.</p>
      {/if}
      {#each Object.entries(diskThemes.errors) as [file, err] (file)}
        <p class="mt-2 text-[12px] text-[var(--red)]"><span class="mono">{file}</span>: {err}</p>
      {/each}
    </div>
  </section>

  <!-- Compose discovery -->
  <section class="rise card p-5" style="animation-delay:60ms">
    <h2 class="text-[14px] font-semibold tracking-tight">Compose discovery</h2>
    <p class="mt-0.5 text-[13px] text-[var(--text-2)]">Find Docker Compose projects on disk so you can deploy them from the Stacks tab.</p>

    {#if discovery.config.roots.length}
      <div class="mt-4 flex flex-col gap-2">
        {#each discovery.config.roots as root (root.id)}
          {@const rr = rootResult(root.id)}
          <div class="flex items-center gap-3 rounded-lg border border-[var(--border)] bg-[var(--panel-2)] px-3 py-2.5">
            <input type="checkbox" checked={root.enabled} onchange={() => updateRoot(root.id, { enabled: !root.enabled })} class="h-4 w-4 shrink-0" style="accent-color:var(--accent)" title="Enabled" />
            <div class="min-w-0 flex-1">
              <div class="mono truncate text-[13px] {root.enabled ? 'text-[var(--text)]' : 'text-[var(--text-3)] line-through'}">{root.path}</div>
              <div class="mt-0.5 text-[11px]">
                {#if rr?.error}<span class="text-[var(--red)]">{rr.error}</span>
                {:else if root.enabled}<span class="text-[var(--text-3)]">{rr?.found ?? 0} project{(rr?.found ?? 0) === 1 ? '' : 's'}</span>
                {:else}<span class="text-[var(--text-3)]">disabled</span>{/if}
              </div>
            </div>
            <label class="flex shrink-0 cursor-pointer items-center gap-1.5 text-[12px] text-[var(--text-2)]" title="Walk subdirectories recursively">
              <input type="checkbox" checked={root.traverse} onchange={() => updateRoot(root.id, { traverse: !root.traverse })} class="h-3.5 w-3.5" style="accent-color:var(--accent)" /> Traverse
            </label>
            <button class="btn btn-ghost btn-icon btn-sm" title="Remove" aria-label="Remove directory" onclick={() => removeRoot(root.id)}><Icon name="trash" size={14} /></button>
          </div>
        {/each}
      </div>
    {/if}

    <div class="mt-3 flex gap-2">
      <PathInput field={pf} onEnter={addDir} placeholder="Add a directory…  /Users/you/projects" />
      <button class="btn btn-default btn-sm" onclick={addDir} disabled={!pf.value.trim()}>Add</button>
    </div>
    <p class="mt-1.5 text-[11px] text-[var(--text-3)]">Turn on <span class="font-medium text-[var(--text-2)]">Traverse</span> to scan subdirectories; off treats the directory itself as one project.</p>

    <div class="mt-5 border-t border-[var(--border)] pt-4">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <span class="text-[13px] font-medium">Filter <span class="font-normal text-[var(--text-3)]">· discovered stacks only</span></span>
        <div class="seg">
          {#each FILTER_MODES as [m, label]}
            <button class="seg-btn {discovery.config.filter.mode === m ? 'on' : ''}" onclick={() => setFilter({ mode: m })}>{label}</button>
          {/each}
        </div>
      </div>
      {#if discovery.config.filter.mode !== 'off'}
        <div class="mt-3 flex flex-wrap gap-1.5">
          {#each discovery.config.filter.patterns as p (p)}
            <span class="mono inline-flex items-center gap-1 rounded-md border border-[var(--border-strong)] bg-[var(--panel-2)] px-2 py-1 text-[11.5px] text-[var(--text-2)]">{p}<button class="text-[var(--text-3)] hover:text-[var(--red)]" aria-label="Remove pattern" onclick={() => removePattern(p)}>×</button></span>
          {/each}
        </div>
        <div class="mt-2 flex gap-2">
          <input bind:value={newPattern} placeholder="web-*  ·  My App  ·  ~/lab/**" class="input mono min-w-0 flex-1" onkeydown={(e) => e.key === 'Enter' && addPat()} />
          <button class="btn btn-default btn-sm" onclick={addPat} disabled={!newPattern.trim()}>Add</button>
        </div>
        <p class="mt-1.5 text-[11px] text-[var(--text-3)]">Matches a project name, its Oriel name, or a directory path (globs &amp; <span class="mono">**</span> allowed). Running stacks are never hidden.</p>
      {/if}
    </div>
  </section>
  </div>

  <div class="flex flex-col gap-4">
  <!-- AI / Natural language -->
  <section class="rise card p-5" style="animation-delay:80ms">
    <div class="flex items-center gap-2.5">
      <h2 class="text-[14px] font-semibold tracking-tight">AI · natural language</h2>
      <span class="pill {provider.enabled ? 'on' : 'off'}"><span class="dot"></span>{provider.enabled ? 'Connected' : 'Not configured'}</span>
    </div>
    <p class="mt-1 text-[13px] text-[var(--text-2)]">
      The base ships no model. Point at an external resolver and the command palette (⌘K) gains a free-text mode — every suggestion still runs through the same validated tool path.
    </p>

    <div class="mt-4">
      <span class="text-[13px] font-medium">Provider URL</span>
      <div class="mt-2 flex flex-wrap gap-2">
        <input bind:value={urlDraft} placeholder="http://127.0.0.1:8899" class="input min-w-0 flex-1" onkeydown={(e) => e.key === 'Enter' && saveProvider()} />
        <button class="btn btn-primary btn-sm" onclick={saveProvider}>Save</button>
        {#if provider.enabled}<button class="btn btn-default btn-sm" onclick={() => { urlDraft = ''; saveProvider() }}>Disable</button>{/if}
      </div>
      <p class="mt-2 text-[12px] text-[var(--text-3)]">
        Tier 1 is a ~40-line rules server; tier 2 swaps in embeddings or a local LLM behind the same <span class="mono">/resolve</span> contract. Or set <span class="mono">ORIEL_PROVIDER_URL</span> at launch.
      </p>
    </div>

    {#if provider.enabled}
      <div class="mt-5 border-t border-[var(--border)] pt-4">
        <span class="text-[13px] font-medium">Test resolver</span>
        <div class="mt-2 flex flex-wrap gap-2">
          <input bind:value={testText} placeholder="e.g. restart postgres" class="input min-w-0 flex-1" onkeydown={(e) => e.key === 'Enter' && runTest()} />
          <button class="btn btn-default btn-sm" onclick={runTest} disabled={testBusy}>{testBusy ? 'Resolving…' : 'Run'}</button>
        </div>
        {#if testErr}<p class="mt-2 text-[12px] text-[var(--red)]">{testErr}</p>{/if}
        {#if testResult?.call}
          <div class="mono mt-3 rounded-lg border border-[var(--border)] bg-[var(--panel-2)] p-3 text-[12px]">
            <div><span class="text-[var(--text-3)]">tool</span> <span class="text-[var(--accent)]">{testResult.call.tool}</span></div>
            <div class="mt-1 break-all"><span class="text-[var(--text-3)]">args</span> {JSON.stringify(testResult.call.args)}</div>
          </div>
        {/if}
      </div>
    {/if}
  </section>

  <!-- Secrets -->
  <section class="rise card p-5" style="animation-delay:90ms">
    <h2 class="text-[14px] font-semibold tracking-tight">Secrets</h2>
    <p class="mt-1 text-[13px] text-[var(--text-2)]">
      Mask environment-variable values in the container inspect panel so API keys don't leak from screenshots or screen-shares. Masking is enforced server-side.
    </p>
    <div class="mt-4 grid gap-4 sm:grid-cols-2">
      <label class="block">
        <span class="text-[13px] font-medium">Mask env values</span>
        <select class="input mt-2 w-full" value={self.maskEnv} onchange={(e) => saveSecret({ maskEnv: e.currentTarget.value })}>
          <option value="all">All values</option>
          <option value="sensitive">Sensitive only</option>
          <option value="off">Off</option>
        </select>
      </label>
      <label class="block">
        <span class="text-[13px] font-medium">Allow “Reveal values”</span>
        <select class="input mt-2 w-full" value={self.envReveal} onchange={(e) => saveSecret({ envReveal: e.currentTarget.value })}>
          <option value="local">Local only</option>
          <option value="remote">Local &amp; remote</option>
          <option value="off">Never (locked)</option>
        </select>
      </label>
    </div>
    <p class="mt-2 text-[12px] text-[var(--text-3)]">
      Reveal is gated server-side: <span class="mono">Local only</span> unmasks just on <span class="mono">127.0.0.1</span>; <span class="mono">Local &amp; remote</span> also allows it from allowed hosts; <span class="mono">Never</span> is a kill-switch.
    </p>
  </section>

  <!-- Updates -->
  <section class="rise card p-5" style="animation-delay:100ms">
    <h2 class="text-[14px] font-semibold tracking-tight">Updates</h2>
    <p class="mt-0.5 text-[12px] text-[var(--text-3)]">Current version <span class="mono text-[var(--text-2)]">{verLabel}</span>.</p>

    <div class="mt-4">
      {#if update.phase === 'restarting'}
        <div class="flex items-center gap-2 text-[13px] text-[var(--text-2)]"><span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-[var(--accent)] border-t-transparent"></span> Restarting Oriel — this page will reconnect…</div>
      {:else if update.phase === 'applying'}
        <div class="flex items-center gap-2 text-[13px] text-[var(--text-2)]"><span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-[var(--accent)] border-t-transparent"></span> Downloading &amp; verifying…</div>
      {:else if update.phase === 'done'}
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="text-[13px] text-[var(--text-2)]">Installed <span class="mono">v{update.latest}</span> — restart to apply.</span>
          <button class="btn btn-primary btn-sm" onclick={() => restartService()}>Restart now</button>
        </div>
      {:else if update.packageManager === 'homebrew'}
        <p class="text-[13px] text-[var(--text-3)]">Installed via Homebrew — update with <span class="mono">brew upgrade oriel</span>.{#if update.available}{' '}A new version <span class="mono">v{update.latest}</span> is out — <a href={update.url} target="_blank" rel="noopener" class="text-[var(--accent)] hover:underline">see release ↗</a>.{/if}</p>
      {:else if !update.managed}
        <p class="text-[13px] text-[var(--text-3)]">In-app updates need a service install (<span class="mono">oriel service install</span>).{#if update.available}{' '}A new version <span class="mono">v{update.latest}</span> is out — <a href={update.url} target="_blank" rel="noopener" class="text-[var(--accent)] hover:underline">see release ↗</a>.{/if}</p>
      {:else if update.available}
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="text-[13px] text-[var(--text-2)]">Update available: <span class="mono font-medium text-[var(--text)]">v{update.latest}</span></span>
          <button class="btn btn-primary btn-sm" onclick={doUpdate}>Update now</button>
        </div>
      {:else}
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="text-[13px] text-[var(--text-3)]">{update.checking ? 'Checking…' : "You're on the latest version."}</span>
          <div class="flex items-center gap-3">
            <button class="btn btn-default btn-sm" onclick={() => checkNow()} disabled={update.checking}>{update.checking ? 'Checking…' : 'Check for updates'}</button>
            <a href={update.url || 'https://github.com/ParadoxInfinite/oriel/releases'} target="_blank" rel="noopener" class="text-[12px] text-[var(--accent)] hover:underline">Releases ↗</a>
          </div>
        </div>
      {/if}
      {#if update.error}<p class="mt-2 text-[12px] text-[var(--red)]">{update.error}</p>{/if}
    </div>
  </section>

  <!-- Remote access -->
  <section class="rise card p-5" style="animation-delay:120ms">
    <h2 class="text-[14px] font-semibold tracking-tight">Remote access</h2>
    <p class="mt-0.5 text-[12px] text-[var(--text-3)]">By default Oriel only answers on <span class="mono">localhost</span>. To reach it over a private network (Tailscale, a reverse proxy, a domain), add those hostnames.</p>
    <p class="mt-2 rounded-lg bg-[var(--red-tint)] px-3 py-2 text-[12px] text-[var(--red)]">Oriel has no login and controls Docker. Only add hosts you reach over a trusted private network — never expose it to the public internet.</p>

    {#if remote.hosts.length}
      <div class="mt-3 flex flex-col gap-1.5">
        {#each remote.hosts as h (h)}
          <div class="flex items-center gap-2 rounded-lg border border-[var(--border)] bg-[var(--panel-2)] px-3 py-2">
            <span class="mono min-w-0 flex-1 truncate text-[13px] text-[var(--text-2)]">{h}</span>
            <button class="text-[var(--text-3)] hover:text-[var(--red)]" aria-label="Remove host" onclick={() => removeRemoteHost(h)}><Icon name="trash" size={13} /></button>
          </div>
        {/each}
      </div>
    {/if}
    <div class="mt-3 flex gap-2">
      <input bind:value={hostDraft} placeholder="oriel.example.com" class="input flex-1 text-[13px]" onkeydown={(e) => e.key === 'Enter' && addHost()} />
      <button class="btn btn-default btn-sm" onclick={addHost} disabled={!hostDraft.trim() || remote.saving}>Add</button>
    </div>
    {#if remote.error}<p class="mt-2 text-[12px] text-[var(--red)]">{remote.error}</p>{/if}
  </section>
  </div>
</div>
