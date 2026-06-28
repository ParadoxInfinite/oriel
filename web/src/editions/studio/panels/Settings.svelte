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
  import { t, tn, locale, AVAILABLE, setLocale } from '../../../platform/index.js'
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
      toast(e?.message || t('settings.toast.saveFailed'), 'error')
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
      toast(e?.message || t('settings.toast.saveFailed'), 'error')
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
        title: t('settings.updates.edgeConfirmTitle'),
        message: t('settings.updates.edgeConfirmMessage'),
        confirmLabel: t('settings.updates.edgeConfirmLabel'),
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
      toast(e?.message || t('settings.updates.channelFailed'), 'error')
    }
  }

  // Accent is per-theme; reflect the base that's actually showing.
  const base = $derived(appearance.mode === 'system' ? (systemPref.dark ? 'dark' : 'light') : appearance.mode)
  const activeAccent = $derived(appearance.accents[base])

  const MODES = $derived([
    { id: 'light', label: t('settings.appearance.themeLight') },
    { id: 'dark', label: t('settings.appearance.themeDark') },
    { id: 'system', label: t('settings.appearance.themeSystem') },
  ])
  const swatches = $derived([...ACCENTS, ...appearance.custom])

  const verLabel = $derived(self.version || t('settings.updates.versionUnknown'))

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
    <h2 class="text-[14px] font-semibold tracking-tight">{t('settings.appearance.title')}</h2>
    <p class="mt-0.5 text-[13px] text-[var(--text-2)]">{t('settings.appearance.desc')}</p>

    <div class="mt-4 flex flex-wrap items-center justify-between gap-3">
      <span class="text-[13px] font-medium">{t('settings.appearance.theme')}</span>
      <div class="seg">
        {#each MODES as m}
          <button class="seg-btn {appearance.mode === m.id ? 'on' : ''}" onclick={() => setMode(m.id)}>{m.label}</button>
        {/each}
      </div>
    </div>

    <div class="mt-5 border-t border-[var(--border)] pt-4">
      <div class="flex items-center justify-between">
        <span class="text-[13px] font-medium">{t('settings.appearance.accent')} <span class="font-normal text-[var(--text-3)]">· {t('settings.appearance.accentBase', { base })}</span></span>
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
              <button class="absolute -right-1.5 -top-1.5 grid h-4 w-4 place-items-center rounded-full border border-[var(--border)] bg-[var(--panel)] text-[var(--text-3)] opacity-0 shadow-[var(--shadow-sm)] transition-opacity hover:text-[var(--red)] group-hover:opacity-100" title={t('action.remove')} aria-label={t('settings.appearance.removeAccent')} onclick={() => removeCustomAccent(sw.id)}>
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
        <input bind:value={newName} placeholder={t('settings.appearance.namePlaceholder')} class="input w-44" />
        <button class="btn btn-default btn-sm" onclick={addAccent}><Icon name="sparkles" size={14} /> {t('settings.appearance.addAccent')}</button>
      </div>
    </div>
  </section>

  <!-- Language -->
  <section class="rise card mb-4 break-inside-avoid p-5" style="animation-delay:20ms">
    <h2 class="text-[14px] font-semibold tracking-tight">{t('settings.language.title')}</h2>
    <p class="mt-0.5 text-[13px] text-[var(--text-2)]">{t('settings.language.desc')}</p>
    <div class="mt-4 flex flex-wrap items-center justify-between gap-3">
      <span class="text-[13px] font-medium">{t('settings.language.label')}</span>
      <select class="input w-44" value={locale.tag} onchange={(e) => setLocale(e.currentTarget.value)}>
        {#each AVAILABLE as l (l.tag)}<option value={l.tag}>{l.name}</option>{/each}
      </select>
    </div>
  </section>

  <!-- Editions & themes -->
  <section class="rise card mb-4 break-inside-avoid p-5" style="animation-delay:40ms">
    <h2 class="text-[14px] font-semibold tracking-tight">{t('settings.editions.title')}</h2>
    <p class="mt-0.5 text-[13px] text-[var(--text-2)]">{t('settings.editions.desc')}</p>

    <div class="mt-4 grid gap-2 sm:grid-cols-2">
      {#each editions() as e (e.id)}
        {@const on = edition.active === e.id}
        <button class="flex items-center gap-3 rounded-lg border p-3 text-left transition-colors {on ? 'border-[var(--accent)] bg-[var(--accent-tint)]' : 'border-[var(--border)] hover:border-[var(--border-strong)] hover:bg-[var(--panel-2)]'}" onclick={() => setEdition(e.id)}>
          <span class="h-3 w-3 shrink-0 rounded-full" style="background:{e.accent};box-shadow:0 0 8px -1px {e.accent}"></span>
          <span class="min-w-0 flex-1">
            <span class="block text-[13px] font-medium">{e.name}{#if e.external}<span class="ml-1.5 text-[10px] font-normal text-[var(--text-3)]">{t('settings.editions.installed')}</span>{/if}</span>
            <span class="block truncate text-[12px] text-[var(--text-3)]">{e.tagline}</span>
          </span>
          {#if on}<span class="text-[var(--accent)]"><Icon name="play" size={13} /></span>{/if}
        </button>
      {/each}
    </div>

    <div class="mt-5 border-t border-[var(--border)] pt-4">
      <div class="flex items-center justify-between gap-2">
        <span class="text-[13px] font-medium">{t('settings.editions.installedThemes')}</span>
        <a href={THEMES_DOC_URL} target="_blank" rel="noopener" class="text-[11.5px] font-medium text-[var(--accent)] hover:underline">{t('settings.editions.buildYourOwn')}</a>
      </div>
      <p class="mt-0.5 text-[12px] text-[var(--text-3)]">{t('settings.editions.dropBundlePre')}<span class="mono">{'{ id, name, component }'}</span>{t('settings.editions.dropBundlePost')}</p>
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
        <p class="mt-2 text-[12px] text-[var(--text-3)]">{t('settings.editions.noThemes')}</p>
      {/if}
      {#each Object.entries(diskThemes.errors) as [file, err] (file)}
        <p class="mt-2 text-[12px] text-[var(--red)]"><span class="mono">{file}</span>: {err}</p>
      {/each}
    </div>
  </section>

  <!-- Automation access (destructive grant) -->
  <section class="rise card mb-4 break-inside-avoid p-5" style="animation-delay:60ms">
    <h2 class="text-[14px] font-semibold tracking-tight">{t('settings.automation.title')}</h2>
    <p class="mt-1 text-[13px] text-[var(--text-2)]">
      {t('settings.automation.descPre')}<span class="mono">oriel mcp</span>{t('settings.automation.descPost')}
    </p>
    <div class="mt-4 flex flex-wrap items-center gap-2">
      {#if grant.active}
        <span class="inline-flex items-center gap-1.5 rounded-full bg-[var(--accent-tint-2)] px-2.5 py-1 text-[12px] font-medium text-[var(--accent)]">
          {t('settings.automation.unlocked', { remaining: fmtRemaining(grant.remainingSeconds) })}
        </span>
        <button class="btn btn-sm btn-default" onclick={() => requestGrant(6)} disabled={grant.busy}>{t('settings.automation.extend6h')}</button>
        <button class="btn btn-sm btn-default" onclick={() => lockGrant()} disabled={grant.busy}>{t('settings.automation.lockNow')}</button>
      {:else}
        <span class="text-[13px] text-[var(--text-3)]">{t('settings.automation.lockedPre')}<span class="font-medium text-[var(--text-2)]">{t('settings.automation.lockedWord')}</span>{t('settings.automation.lockedPost')}</span>
        <button class="btn btn-sm btn-primary" onclick={() => requestGrant(0.25)} disabled={grant.busy}>{t('settings.automation.allow15m')}</button>
        <button class="btn btn-sm btn-default" onclick={() => requestGrant(6)} disabled={grant.busy}>{t('settings.automation.allow6h')}</button>
        <button class="btn btn-sm btn-default" onclick={() => requestGrant(24 * 6)} disabled={grant.busy}>{t('settings.automation.allow6d')}</button>
      {/if}
    </div>
    <p class="mt-2 text-[12px] text-[var(--text-3)]">
      {t('settings.automation.cliNotePre')}<span class="mono">oriel ai allow-destructive</span>{t('settings.automation.cliNotePost')}
    </p>
  </section>

  <!-- Secrets -->
  <section class="rise card mb-4 break-inside-avoid p-5" style="animation-delay:80ms">
    <h2 class="text-[14px] font-semibold tracking-tight">{t('settings.secrets.title')}</h2>
    <p class="mt-1 text-[13px] text-[var(--text-2)]">
      {t('settings.secrets.desc')}
    </p>
    <div class="mt-4 grid gap-4 sm:grid-cols-2">
      <label class="block">
        <span class="text-[13px] font-medium">{t('settings.secrets.maskEnv')}</span>
        <select class="input mt-2 w-full" value={self.maskEnv} onchange={(e) => saveSecret({ maskEnv: e.currentTarget.value })}>
          <option value="all">{t('settings.secrets.maskEnvAll')}</option>
          <option value="sensitive">{t('settings.secrets.maskEnvSensitive')}</option>
          <option value="off">{t('settings.secrets.maskEnvOff')}</option>
        </select>
      </label>
      <label class="block">
        <span class="text-[13px] font-medium">{t('settings.secrets.allowReveal')}</span>
        <select class="input mt-2 w-full" value={self.envReveal} onchange={(e) => saveSecret({ envReveal: e.currentTarget.value })}>
          <option value="local">{t('settings.secrets.revealLocal')}</option>
          <option value="remote">{t('settings.secrets.revealRemote')}</option>
          <option value="off">{t('settings.secrets.revealOff')}</option>
        </select>
      </label>
      <label class="block">
        <span class="text-[13px] font-medium">{t('settings.secrets.maskLogs')}</span>
        <select class="input mt-2 w-full" value={self.maskLogs} onchange={(e) => saveSecret({ maskLogs: e.currentTarget.value })}>
          <option value="sensitive">{t('settings.secrets.maskLogsRedact')}</option>
          <option value="off">{t('settings.secrets.maskLogsRaw')}</option>
        </select>
      </label>
    </div>
    <p class="mt-2 text-[12px] text-[var(--text-3)]">
      {t('settings.secrets.notePre')}<span class="mono">{t('settings.secrets.revealLocal')}</span>{t('settings.secrets.noteMid1')}<span class="mono">127.0.0.1</span>{t('settings.secrets.noteMid2')}<span class="mono">{t('settings.secrets.revealRemote')}</span>{t('settings.secrets.noteMid3')}<span class="mono">{t('settings.secrets.revealNeverWord')}</span>{t('settings.secrets.notePost')}
    </p>
  </section>

  <!-- Authentication -->
  <section class="rise card mb-4 break-inside-avoid p-5" style="animation-delay:100ms">
    <h2 class="text-[14px] font-semibold tracking-tight">{t('settings.auth.title')}</h2>
    <p class="mt-1 text-[13px] text-[var(--text-2)]">
      {t('settings.auth.desc')}
    </p>

    <div class="mt-4 flex flex-wrap items-center gap-2">
      {#if auth.enabled}
        <span class="inline-flex items-center gap-1.5 rounded-full bg-[var(--accent-tint-2)] px-2.5 py-1 text-[12px] font-medium text-[var(--accent)]">{t('settings.auth.tokenSet')}</span>
        {#if auth.localAdmin}
          <button class="btn btn-sm btn-default" onclick={() => setToken({ generate: true }, t('settings.auth.regenerateFailed'))} disabled={tokenBusy}>{t('settings.auth.regenerate')}</button>
          <button class="btn btn-sm btn-default" onclick={() => setToken({ clear: true }, t('settings.auth.clearFailed'))} disabled={tokenBusy}>{t('action.clear')}</button>
        {/if}
      {:else}
        <span class="text-[13px] text-[var(--text-3)]">{t('settings.auth.noToken')}</span>
        {#if auth.localAdmin}
          <button class="btn btn-sm btn-primary" onclick={() => setToken({ generate: true }, t('settings.auth.setFailed'))} disabled={tokenBusy}>{t('settings.auth.generateToken')}</button>
        {/if}
      {/if}
    </div>

    {#if !auth.localAdmin}
      <p class="mt-2 text-[12px] text-[var(--text-3)]">{t('settings.auth.setOnMachinePre')}<span class="mono">oriel config auth-token</span>{t('settings.auth.setOnMachinePost')}</p>
    {/if}

    {#if revealedToken}
      <div class="mt-3 rounded-lg border border-[var(--border)] bg-[var(--panel-2)] p-3">
        <p class="text-[12px] text-[var(--text-2)]">{t('settings.auth.copyNow')}</p>
        <code class="mono mt-1 block break-all text-[12px] text-[var(--text)]">{revealedToken}</code>
      </div>
    {/if}

    {#if auth.enabled}
      <div class="mt-4 grid gap-4 sm:grid-cols-2">
        <label class="block">
          <span class="text-[13px] font-medium">{t('settings.auth.sessionTimeout')} <span class="font-normal text-[var(--text-3)]">{t('settings.auth.minutes')}</span></span>
          <input type="number" min="1" class="input mt-2 w-full" value={self.sessionTTLMinutes || ''} placeholder={t('settings.auth.sessionTimeoutPlaceholder')} onchange={(e) => saveAuthKnob({ sessionTTLMinutes: Number(e.currentTarget.value) || 0 })} />
        </label>
        <label class="block">
          <span class="text-[13px] font-medium">{t('settings.auth.loginAttempts')}</span>
          <input type="number" min="1" class="input mt-2 w-full" value={self.loginFreeAttempts || ''} placeholder="5" onchange={(e) => saveAuthKnob({ loginFreeAttempts: Number(e.currentTarget.value) || 0 })} />
        </label>
      </div>
      <p class="mt-2 text-[12px] text-[var(--text-3)]">{t('settings.auth.knobsHelp')}</p>
      {#if auth.authenticated}
        <button class="btn btn-sm btn-default mt-4" onclick={() => logout()}>{t('settings.auth.signOut')}</button>
      {/if}
    {/if}
  </section>

  <!-- Remote access -->
  <section class="rise card mb-4 break-inside-avoid p-5" style="animation-delay:120ms">
    <h2 class="text-[14px] font-semibold tracking-tight">{t('settings.remote.title')}</h2>
    <p class="mt-0.5 text-[12px] text-[var(--text-3)]">{t('settings.remote.descPre')}<span class="mono">localhost</span>{t('settings.remote.descPost')}</p>
    <p class="mt-2 rounded-lg bg-[var(--red-tint)] px-3 py-2 text-[12px] text-[var(--red)]">{t('settings.remote.warning')}</p>

    {#if remote.hosts.length}
      <div class="mt-3 flex flex-col gap-1.5">
        {#each remote.hosts as h (h)}
          <div class="flex items-center gap-2 rounded-lg border border-[var(--border)] bg-[var(--panel-2)] px-3 py-2">
            <span class="mono min-w-0 flex-1 truncate text-[13px] text-[var(--text-2)]">{h}</span>
            <button class="text-[var(--text-3)] hover:text-[var(--red)]" aria-label={t('settings.remote.removeHost')} onclick={() => removeRemoteHost(h)}><Icon name="trash" size={13} /></button>
          </div>
        {/each}
      </div>
    {/if}
    <div class="mt-3 flex gap-2">
      <input bind:value={hostForm.draft} placeholder="oriel.example.com" class="input flex-1 text-[13px]" onkeydown={(e) => e.key === 'Enter' && hostForm.add()} />
      <button class="btn btn-default btn-sm" onclick={() => hostForm.add()} disabled={!hostForm.draft.trim() || remote.saving}>{t('settings.remote.add')}</button>
    </div>
    {#if remote.error}<p class="mt-2 text-[12px] text-[var(--red)]">{remote.error}</p>{/if}
  </section>

  <!-- Updates -->
  <section class="rise card mb-4 break-inside-avoid p-5" style="animation-delay:140ms">
    <h2 class="text-[14px] font-semibold tracking-tight">{t('settings.updates.title')}</h2>
    <p class="mt-0.5 text-[12px] text-[var(--text-3)]">{t('settings.updates.currentVersionPre')}<span class="mono text-[var(--text-2)]">{verLabel}</span>{t('settings.updates.currentVersionPost')}</p>

    <div class="mt-4">
      {#if update.phase === 'restarting'}
        <div class="flex items-center gap-2 text-[13px] text-[var(--text-2)]"><span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-[var(--accent)] border-t-transparent"></span> {t('settings.updates.restarting')}</div>
      {:else if update.phase === 'applying'}
        <div class="flex items-center gap-2 text-[13px] text-[var(--text-2)]"><span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-[var(--accent)] border-t-transparent"></span> {t('settings.updates.applying')}</div>
      {:else if update.phase === 'done'}
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="text-[13px] text-[var(--text-2)]">{t('settings.updates.installedPre')}<span class="mono">v{update.latest}</span>{t('settings.updates.installedPost')}</span>
          <button class="btn btn-primary btn-sm" onclick={() => restartService()}>{t('settings.updates.restartNow')}</button>
        </div>
      {:else if update.packageManager === 'homebrew'}
        <p class="text-[13px] text-[var(--text-3)]">{t('settings.updates.homebrewPre')}<span class="mono">brew upgrade oriel</span>{t('settings.updates.homebrewPost')}{#if update.available}{' '}{t('settings.updates.newVersionPre')}<span class="mono">v{update.latest}</span>{t('settings.updates.newVersionMid')}<a href={update.url} target="_blank" rel="noopener" class="text-[var(--accent)] hover:underline">{t('settings.updates.seeRelease')}</a>{t('settings.updates.newVersionPost')}{/if}</p>
      {:else if update.packageManager === 'container'}
        <p class="text-[13px] text-[var(--text-3)]">{t('settings.updates.containerPre')}<span class="mono">docker pull ghcr.io/paradoxinfinite/oriel</span>{t('settings.updates.containerPost')}{#if update.available}{' '}{t('settings.updates.newVersionPre')}<span class="mono">v{update.latest}</span>{t('settings.updates.newVersionMid')}<a href={update.url} target="_blank" rel="noopener" class="text-[var(--accent)] hover:underline">{t('settings.updates.seeRelease')}</a>{t('settings.updates.newVersionPost')}{/if}</p>
      {:else if !update.managed}
        <p class="text-[13px] text-[var(--text-3)]">{t('settings.updates.unmanagedPre')}<span class="mono">oriel service install</span>{t('settings.updates.unmanagedPost')}{#if update.available}{' '}{t('settings.updates.newVersionPre')}<span class="mono">v{update.latest}</span>{t('settings.updates.newVersionMid')}<a href={update.url} target="_blank" rel="noopener" class="text-[var(--accent)] hover:underline">{t('settings.updates.seeRelease')}</a>{t('settings.updates.newVersionPost')}{/if}</p>
      {:else if update.available}
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="text-[13px] text-[var(--text-2)]">{t('settings.updates.updateAvailablePre')}<span class="mono font-medium text-[var(--text)]">v{update.latest}</span></span>
          <button class="btn btn-primary btn-sm" onclick={promptUpdate}>{t('settings.updates.updateNow')}</button>
        </div>
      {:else}
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="text-[13px] text-[var(--text-3)]">{update.checking ? t('settings.updates.checking') : t('settings.updates.onLatest')}</span>
          <div class="flex items-center gap-3">
            <button class="btn btn-default btn-sm" onclick={() => checkNow()} disabled={update.checking}>{update.checking ? t('settings.updates.checking') : t('settings.updates.checkForUpdates')}</button>
            <a href={update.url || 'https://github.com/ParadoxInfinite/oriel/releases'} target="_blank" rel="noopener" class="text-[12px] text-[var(--accent)] hover:underline">{t('settings.updates.releases')}</a>
          </div>
        </div>
      {/if}
      {#if update.error}<p class="mt-2 text-[12px] text-[var(--red)]">{update.error}</p>{/if}
    </div>

    <div class="mt-5 border-t border-[var(--border)] pt-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <span class="text-[13px] font-medium">{t('settings.updates.releaseChannel')}</span>
        <div class="seg">
          <button class="seg-btn {self.updateChannel === 'stable' ? 'on' : ''}" onclick={() => setChannel('stable')}>{t('settings.updates.stable')}</button>
          <button class="seg-btn {self.updateChannel === 'edge' ? 'on warn' : ''}" onclick={() => setChannel('edge')}>{t('settings.updates.edge')}</button>
        </div>
      </div>
      {#if self.updateChannel === 'edge'}
        <p class="mt-3 flex items-start gap-2 rounded-lg bg-[var(--amber-tint)] px-3 py-2 text-[12px] text-[var(--amber)]">
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" class="mt-0.5 shrink-0"><path d="M12 9v4" /><path d="M12 17h.01" /><path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z" /></svg>
          <span>{t('settings.updates.edgeWarn')}</span>
        </p>
      {:else}
        <p class="mt-2 text-[12px] text-[var(--text-3)]">
          <span class="mono">{t('settings.updates.stable')}</span> {t('settings.updates.channelHelpStable')} <span class="mono">{t('settings.updates.edge')}</span> {t('settings.updates.channelHelpEdge')}
        </p>
      {/if}
    </div>
  </section>

  <!-- Compose discovery (summary; full UI in a dialog) -->
  <section class="rise card mb-4 break-inside-avoid p-5" style="animation-delay:160ms">
    <div class="flex items-start justify-between gap-2">
      <h2 class="text-[14px] font-semibold tracking-tight">{t('settings.discovery.title')}</h2>
      <button class="btn btn-sm btn-default" onclick={() => (showDiscovery = true)}>{t('settings.discovery.manage')}</button>
    </div>
    <p class="mt-1 text-[13px] text-[var(--text-2)]">{t('settings.discovery.desc')}</p>
    <p class="mt-2 text-[12px] text-[var(--text-3)]">
      {#if discovery.config.roots.length}
        {t('settings.discovery.summary', { dirs: tn('settings.discovery.directories', discovery.config.roots.length), enabled: enabledRoots, projects: tn('settings.discovery.projects', totalProjects) })}
      {:else}
        {t('settings.discovery.empty')}
      {/if}
    </p>
  </section>

  <!-- AI activity (summary; full log in a dialog) -->
  <section class="rise card mb-4 break-inside-avoid p-5" style="animation-delay:180ms">
    <div class="flex items-start justify-between gap-2">
      <h2 class="text-[14px] font-semibold tracking-tight">{t('settings.ai.title')}</h2>
      <button class="btn btn-sm btn-default" onclick={() => (showAudit = true)}>{t('settings.ai.viewActivity')}</button>
    </div>
    <p class="mt-1 text-[13px] text-[var(--text-2)]">{t('settings.ai.desc')}</p>
    <p class="mt-2 text-[12px] text-[var(--text-3)]">
      {#if audit.error}
        <span class="text-[var(--red)]">{audit.error}</span>
      {:else if audit.entries.length}
        {tn('settings.ai.recentCalls', audit.entries.length, { time: fmtTime(audit.entries[0].time) })}
      {:else}
        {t('settings.ai.emptyPre')}<span class="mono">oriel mcp</span>{t('settings.ai.emptyPost')}
      {/if}
    </p>
  </section>
</div>

{#if showDiscovery}<ComposeDiscoveryDialog onClose={() => (showDiscovery = false)} />{/if}
{#if showAudit}<AuditLogDialog onClose={() => (showAudit = false)} />{/if}
