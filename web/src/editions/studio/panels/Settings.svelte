<script>
  import { onMount } from 'svelte'
  import { toast, confirm } from '../../../platform/index.js'
  import { discovery, ensureDiscovery, rootResult, THEMES_DOC_URL } from '../../../platform/index.js'
  import { self, update, checkNow, restartService, promptUpdate, apiPut } from '../../../platform/index.js'
  import { remote, loadRemote, removeRemoteHost, RemoteHostForm } from '../../../platform/index.js'
  import { grant, loadGrant, requestGrant, lockGrant, fmtRemaining } from '../../../platform/index.js'
  import { auth, logout } from '../../../platform/index.js'
  import { audit, loadAudit } from '../../../platform/index.js'
  import { takeTarget } from '../../../platform/index.js'
  import { editions, edition, setEdition, diskThemes } from '../../../editions/registry.svelte.js'
  import { appearance, systemPref, ACCENTS, setMode, setAccent, addCustomAccent, removeCustomAccent } from '../theme.svelte.js'
  import Icon from '../lib/Icon.svelte'
  import ComposeDiscoveryDialog from '../lib/ComposeDiscoveryDialog.svelte'
  import AuditLogDialog from '../lib/AuditLogDialog.svelte'

  ensureDiscovery()
  const hostForm = new RemoteHostForm()
  onMount(loadRemote)
  onMount(loadGrant)
  onMount(loadAudit)

  // Compose discovery + AI activity moved to their own dialogs; these cards show
  // a compact summary and open the full view.
  let showDiscovery = $state(false)
  let showAudit = $state(false)
  // Deep link: Stacks → "Compose discovery" opens the dialog straight away.
  $effect(() => {
    const t = takeTarget('Settings')
    if (t?.kind === 'discovery') showDiscovery = true
  })
  const enabledRoots = $derived(discovery.config.roots.filter((r) => r.enabled).length)
  const totalProjects = $derived(discovery.config.roots.reduce((n, r) => n + (rootResult(r.id)?.found ?? 0), 0))
  function fmtTime(iso) {
    try {
      return new Date(iso).toLocaleString()
    } catch {
      return iso
    }
  }

  // Secret-masking + reveal policy (Settings → Secrets), saved via /api/config.
  async function saveSecret(patch) {
    const prev = { maskEnv: self.maskEnv, maskLogs: self.maskLogs, envReveal: self.envReveal }
    Object.assign(self, patch) // optimistic
    try {
      const d = await apiPut('/api/config', patch)
      if (d?.maskEnv) self.maskEnv = d.maskEnv
      if (d?.maskLogs) self.maskLogs = d.maskLogs
      if (d?.envReveal) self.envReveal = d.envReveal
    } catch (e) {
      Object.assign(self, prev)
      toast(e?.message || 'Could not save', 'error')
    }
  }

  // Authentication: the token is local-machine-only (PUT /api/auth); the session
  // knobs are settable by any authenticated session (PUT /api/config).
  let tokenBusy = $state(false)
  let revealedToken = $state('') // shown once, right after generating
  async function setToken(body, msg) {
    tokenBusy = true
    try {
      const d = await apiPut('/api/auth', body)
      auth.enabled = !!d?.enabled
      revealedToken = d?.token || ''
    } catch (e) {
      toast(e?.message || msg, 'error')
    }
    tokenBusy = false
  }
  async function saveAuthKnob(patch) {
    const prev = { sessionTTLMinutes: self.sessionTTLMinutes, loginFreeAttempts: self.loginFreeAttempts }
    Object.assign(self, patch) // optimistic
    try {
      const d = await apiPut('/api/config', patch)
      if (d?.sessionTTLMinutes != null) self.sessionTTLMinutes = d.sessionTTLMinutes
      if (d?.loginFreeAttempts != null) self.loginFreeAttempts = d.loginFreeAttempts
    } catch (e) {
      Object.assign(self, prev)
      toast(e?.message || 'Could not save', 'error')
    }
  }

  // Self-update channel (Settings → Updates). Opting into edge is gated by a
  // warn-toned confirm so the "you stay on a build until the next stable catches
  // up" behavior is explicit; switching back to stable is the safe direction and
  // isn't gated.
  async function setChannel(channel) {
    if (channel === self.updateChannel) return
    if (channel === 'edge') {
      const ok = await confirm({
        title: 'Switch to the Edge channel?',
        message:
          "Edge gets the newest builds first, including pre-releases that are less tested. You stay on a build until the next stable release catches up; switching back to Stable doesn't downgrade you, it just waits for stable to reach you. Use Edge to help test; keep Stable for everyday use.",
        confirmLabel: 'Switch to Edge',
        tone: 'warn',
      })
      if (!ok) return
    }
    const prev = self.updateChannel
    self.updateChannel = channel
    try {
      await apiPut('/api/config', { updateChannel: channel })
      checkNow() // re-check against the new channel
    } catch (e) {
      self.updateChannel = prev
      toast(e?.message || 'Could not change channel', 'error')
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

  const verLabel = $derived(self.version || 'unknown')

  // Custom accent form.
  let newColor = $state('#22c55e')
  let newName = $state('')
  function addAccent() {
    addCustomAccent(newName, newColor)
    newName = ''
  }
</script>

<!-- Cards auto-balance across two columns (CSS multi-column); each is kept whole
     with break-inside-avoid. Order top-to-bottom is priority order. -->
<div class="mx-auto max-w-5xl columns-1 gap-4 pb-4 [column-fill:balance] md:columns-2">
  <!-- Appearance -->
  <section class="rise card mb-4 break-inside-avoid p-5">
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
  <section class="rise card mb-4 break-inside-avoid p-5" style="animation-delay:40ms">
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

  <!-- Automation access (destructive grant) -->
  <section class="rise card mb-4 break-inside-avoid p-5" style="animation-delay:60ms">
    <h2 class="text-[14px] font-semibold tracking-tight">Automation access</h2>
    <p class="mt-1 text-[13px] text-[var(--text-2)]">
      The MCP server (<span class="mono">oriel mcp</span>) can run <em>read</em> actions any time. <em>Destructive</em> ones (remove, prune) stay locked until you open a time-boxed window, your own clicks here in the UI are never affected.
    </p>
    <div class="mt-4 flex flex-wrap items-center gap-2">
      {#if grant.active}
        <span class="inline-flex items-center gap-1.5 rounded-full bg-[var(--accent-tint-2)] px-2.5 py-1 text-[12px] font-medium text-[var(--accent)]">
          Unlocked · {fmtRemaining(grant.remainingSeconds)} left
        </span>
        <button class="btn btn-sm btn-default" onclick={() => requestGrant(6)} disabled={grant.busy}>Extend 6h</button>
        <button class="btn btn-sm btn-default" onclick={() => lockGrant()} disabled={grant.busy}>Lock now</button>
      {:else}
        <span class="text-[13px] text-[var(--text-3)]">Destructive actions are <span class="font-medium text-[var(--text-2)]">locked</span> for automation.</span>
        <button class="btn btn-sm btn-primary" onclick={() => requestGrant(0.25)} disabled={grant.busy}>Allow 15m</button>
        <button class="btn btn-sm btn-default" onclick={() => requestGrant(6)} disabled={grant.busy}>Allow 6h</button>
        <button class="btn btn-sm btn-default" onclick={() => requestGrant(24 * 6)} disabled={grant.busy}>Allow 6d</button>
      {/if}
    </div>
    <p class="mt-2 text-[12px] text-[var(--text-3)]">
      Same window the <span class="mono">oriel ai allow-destructive</span> CLI opens. It auto-relocks when it lapses.
    </p>
  </section>

  <!-- Secrets -->
  <section class="rise card mb-4 break-inside-avoid p-5" style="animation-delay:80ms">
    <h2 class="text-[14px] font-semibold tracking-tight">Secrets</h2>
    <p class="mt-1 text-[13px] text-[var(--text-2)]">
      Mask environment-variable values in the container inspect panel, and redact secret-shaped tokens from container logs, so API keys don't leak from screenshots or screen-shares. Masking is enforced server-side.
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
      <label class="block">
        <span class="text-[13px] font-medium">Mask log secrets</span>
        <select class="input mt-2 w-full" value={self.maskLogs} onchange={(e) => saveSecret({ maskLogs: e.currentTarget.value })}>
          <option value="sensitive">Redact secrets</option>
          <option value="off">Show raw</option>
        </select>
      </label>
    </div>
    <p class="mt-2 text-[12px] text-[var(--text-3)]">
      Reveal is gated server-side: <span class="mono">Local only</span> unmasks just on <span class="mono">127.0.0.1</span>; <span class="mono">Local &amp; remote</span> also allows it from allowed hosts; <span class="mono">Never</span> is a kill-switch. Log redaction is best-effort (free-form text); an AI client over MCP always gets at least secret redaction, regardless of this setting.
    </p>
  </section>

  <!-- Authentication -->
  <section class="rise card mb-4 break-inside-avoid p-5" style="animation-delay:100ms">
    <h2 class="text-[14px] font-semibold tracking-tight">Authentication</h2>
    <p class="mt-1 text-[13px] text-[var(--text-2)]">
      An optional access token gates non-loopback and MCP-over-HTTP access. Local use never needs it; with it on, the browser logs in once over your private network.
    </p>

    <div class="mt-4 flex flex-wrap items-center gap-2">
      {#if auth.enabled}
        <span class="inline-flex items-center gap-1.5 rounded-full bg-[var(--accent-tint-2)] px-2.5 py-1 text-[12px] font-medium text-[var(--accent)]">Token set · remote login required</span>
        {#if auth.localAdmin}
          <button class="btn btn-sm btn-default" onclick={() => setToken({ generate: true }, 'Could not regenerate token')} disabled={tokenBusy}>Regenerate</button>
          <button class="btn btn-sm btn-default" onclick={() => setToken({ clear: true }, 'Could not clear token')} disabled={tokenBusy}>Clear</button>
        {/if}
      {:else}
        <span class="text-[13px] text-[var(--text-3)]">No token, loopback-only access.</span>
        {#if auth.localAdmin}
          <button class="btn btn-sm btn-primary" onclick={() => setToken({ generate: true }, 'Could not set token')} disabled={tokenBusy}>Generate token</button>
        {/if}
      {/if}
    </div>

    {#if !auth.localAdmin}
      <p class="mt-2 text-[12px] text-[var(--text-3)]">The token is set on the machine running Oriel: <span class="mono">oriel config auth-token</span>.</p>
    {/if}

    {#if revealedToken}
      <div class="mt-3 rounded-lg border border-[var(--border)] bg-[var(--panel-2)] p-3">
        <p class="text-[12px] text-[var(--text-2)]">Copy this now, it won't be shown again:</p>
        <code class="mono mt-1 block break-all text-[12px] text-[var(--text)]">{revealedToken}</code>
      </div>
    {/if}

    {#if auth.enabled}
      <div class="mt-4 grid gap-4 sm:grid-cols-2">
        <label class="block">
          <span class="text-[13px] font-medium">Session timeout <span class="font-normal text-[var(--text-3)]">(minutes)</span></span>
          <input type="number" min="1" class="input mt-2 w-full" value={self.sessionTTLMinutes || ''} placeholder="10080 · 7 days" onchange={(e) => saveAuthKnob({ sessionTTLMinutes: Number(e.currentTarget.value) || 0 })} />
        </label>
        <label class="block">
          <span class="text-[13px] font-medium">Login attempts before backoff</span>
          <input type="number" min="1" class="input mt-2 w-full" value={self.loginFreeAttempts || ''} placeholder="5" onchange={(e) => saveAuthKnob({ loginFreeAttempts: Number(e.currentTarget.value) || 0 })} />
        </label>
      </div>
      <p class="mt-2 text-[12px] text-[var(--text-3)]">Blank = default. The session timeout is a sliding idle window (capped at 30 days); after the free attempts, failed logins back off exponentially.</p>
      {#if auth.authenticated}
        <button class="btn btn-sm btn-default mt-4" onclick={() => logout()}>Sign out</button>
      {/if}
    {/if}
  </section>

  <!-- Remote access -->
  <section class="rise card mb-4 break-inside-avoid p-5" style="animation-delay:120ms">
    <h2 class="text-[14px] font-semibold tracking-tight">Remote access</h2>
    <p class="mt-0.5 text-[12px] text-[var(--text-3)]">By default Oriel only answers on <span class="mono">localhost</span>. To reach it over a private network (Tailscale, a reverse proxy, a domain), add those hostnames.</p>
    <p class="mt-2 rounded-lg bg-[var(--red-tint)] px-3 py-2 text-[12px] text-[var(--red)]">Oriel controls Docker. Only add hosts you reach over a trusted private network, never expose it to the public internet. Set an access token above to require a login for remote access.</p>

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
      <input bind:value={hostForm.draft} placeholder="oriel.example.com" class="input flex-1 text-[13px]" onkeydown={(e) => e.key === 'Enter' && hostForm.add()} />
      <button class="btn btn-default btn-sm" onclick={() => hostForm.add()} disabled={!hostForm.draft.trim() || remote.saving}>Add</button>
    </div>
    {#if remote.error}<p class="mt-2 text-[12px] text-[var(--red)]">{remote.error}</p>{/if}
  </section>

  <!-- Updates -->
  <section class="rise card mb-4 break-inside-avoid p-5" style="animation-delay:140ms">
    <h2 class="text-[14px] font-semibold tracking-tight">Updates</h2>
    <p class="mt-0.5 text-[12px] text-[var(--text-3)]">Current version <span class="mono text-[var(--text-2)]">{verLabel}</span>.</p>

    <div class="mt-4">
      {#if update.phase === 'restarting'}
        <div class="flex items-center gap-2 text-[13px] text-[var(--text-2)]"><span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-[var(--accent)] border-t-transparent"></span> Restarting Oriel, this page will reconnect…</div>
      {:else if update.phase === 'applying'}
        <div class="flex items-center gap-2 text-[13px] text-[var(--text-2)]"><span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-[var(--accent)] border-t-transparent"></span> Downloading &amp; verifying…</div>
      {:else if update.phase === 'done'}
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="text-[13px] text-[var(--text-2)]">Installed <span class="mono">v{update.latest}</span>, restart to apply.</span>
          <button class="btn btn-primary btn-sm" onclick={() => restartService()}>Restart now</button>
        </div>
      {:else if update.packageManager === 'homebrew'}
        <p class="text-[13px] text-[var(--text-3)]">Installed via Homebrew, update with <span class="mono">brew upgrade oriel</span>.{#if update.available}{' '}A new version <span class="mono">v{update.latest}</span> is out, <a href={update.url} target="_blank" rel="noopener" class="text-[var(--accent)] hover:underline">see release ↗</a>.{/if}</p>
      {:else if update.packageManager === 'container'}
        <p class="text-[13px] text-[var(--text-3)]">Running in a container, update with <span class="mono">docker pull ghcr.io/paradoxinfinite/oriel</span> and recreate it.{#if update.available}{' '}A new version <span class="mono">v{update.latest}</span> is out, <a href={update.url} target="_blank" rel="noopener" class="text-[var(--accent)] hover:underline">see release ↗</a>.{/if}</p>
      {:else if !update.managed}
        <p class="text-[13px] text-[var(--text-3)]">In-app updates need a service install (<span class="mono">oriel service install</span>).{#if update.available}{' '}A new version <span class="mono">v{update.latest}</span> is out, <a href={update.url} target="_blank" rel="noopener" class="text-[var(--accent)] hover:underline">see release ↗</a>.{/if}</p>
      {:else if update.available}
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="text-[13px] text-[var(--text-2)]">Update available: <span class="mono font-medium text-[var(--text)]">v{update.latest}</span></span>
          <button class="btn btn-primary btn-sm" onclick={promptUpdate}>Update now</button>
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

    <div class="mt-5 border-t border-[var(--border)] pt-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <span class="text-[13px] font-medium">Release channel</span>
        <div class="seg">
          <button class="seg-btn {self.updateChannel === 'stable' ? 'on' : ''}" onclick={() => setChannel('stable')}>Stable</button>
          <button class="seg-btn {self.updateChannel === 'edge' ? 'on warn' : ''}" onclick={() => setChannel('edge')}>Edge</button>
        </div>
      </div>
      {#if self.updateChannel === 'edge'}
        <p class="mt-3 flex items-start gap-2 rounded-lg bg-[var(--amber-tint)] px-3 py-2 text-[12px] text-[var(--amber)]">
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" class="mt-0.5 shrink-0"><path d="M12 9v4" /><path d="M12 17h.01" /><path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z" /></svg>
          <span>On Edge you get pre-releases that are less tested. You stay on a build until the next stable catches up; switching back to Stable never downgrades you.</span>
        </p>
      {:else}
        <p class="mt-2 text-[12px] text-[var(--text-3)]">
          <span class="mono">Stable</span> tracks confirmed releases. <span class="mono">Edge</span> gets the newest builds first, including pre-releases.
        </p>
      {/if}
    </div>
  </section>

  <!-- Compose discovery (summary; full UI in a dialog) -->
  <section class="rise card mb-4 break-inside-avoid p-5" style="animation-delay:160ms">
    <div class="flex items-start justify-between gap-2">
      <h2 class="text-[14px] font-semibold tracking-tight">Compose discovery</h2>
      <button class="btn btn-sm btn-default" onclick={() => (showDiscovery = true)}>Manage</button>
    </div>
    <p class="mt-1 text-[13px] text-[var(--text-2)]">Find Docker Compose projects on disk so you can deploy them from the Stacks tab.</p>
    <p class="mt-2 text-[12px] text-[var(--text-3)]">
      {#if discovery.config.roots.length}
        {discovery.config.roots.length} {discovery.config.roots.length === 1 ? 'directory' : 'directories'} ({enabledRoots} enabled) · {totalProjects} project{totalProjects === 1 ? '' : 's'} found
      {:else}
        No directories added yet. Add one to deploy stacks from disk.
      {/if}
    </p>
  </section>

  <!-- AI activity (summary; full log in a dialog) -->
  <section class="rise card mb-4 break-inside-avoid p-5" style="animation-delay:180ms">
    <div class="flex items-start justify-between gap-2">
      <h2 class="text-[14px] font-semibold tracking-tight">AI activity</h2>
      <button class="btn btn-sm btn-default" onclick={() => (showAudit = true)}>View activity</button>
    </div>
    <p class="mt-1 text-[13px] text-[var(--text-2)]">Tool calls an MCP client or assistant made. Your own clicks here aren't recorded.</p>
    <p class="mt-2 text-[12px] text-[var(--text-3)]">
      {#if audit.error}
        <span class="text-[var(--red)]">{audit.error}</span>
      {:else if audit.entries.length}
        {audit.entries.length} recent call{audit.entries.length === 1 ? '' : 's'} · last {fmtTime(audit.entries[0].time)}
      {:else}
        No AI activity yet. Point an MCP client at <span class="mono">oriel mcp</span>.
      {/if}
    </p>
  </section>
</div>

{#if showDiscovery}<ComposeDiscoveryDialog onClose={() => (showDiscovery = false)} />{/if}
{#if showAudit}<AuditLogDialog onClose={() => (showAudit = false)} />{/if}
