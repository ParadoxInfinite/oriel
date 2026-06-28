<script>
  import { inspectNetwork, invoke, containers, refreshContainers, refreshNetworks, registerEscape, trapFocus, t } from '../../../platform/index.js'
  import Icon from './Icon.svelte'

  let { network, onClose } = $props()

  // The connect picker reads the container list; make sure it's fresh on open.
  $effect(() => {
    refreshContainers()
  })

  let detail = $state(null)
  let error = $state(null)
  let loading = $state(true)
  let busy = $state(false)
  let pick = $state('') // container to connect

  async function refresh() {
    loading = true
    try {
      detail = await inspectNetwork(network.id)
      error = null
    } catch (e) {
      error = e.message
    } finally {
      loading = false
    }
  }
  $effect(() => {
    refresh()
  })
  $effect(() => registerEscape(() => !busy && onClose()))

  const attached = $derived(new Set((detail?.containers ?? []).map((c) => c.name)))
  // Containers not already on this network, the candidates the picker offers.
  const candidates = $derived(containers.list.filter((c) => !attached.has(c.name)))

  async function connect() {
    if (!pick || busy) return
    busy = true
    const ok = await invoke('network.connect', { network: network.id, container: pick }, { success: t('network.detail.connected', { name: pick }) })
    busy = false
    if (ok !== false) {
      pick = ''
      await refresh()
      refreshNetworks()
    }
  }
  async function disconnect(c) {
    if (busy) return
    busy = true
    const ok = await invoke('network.disconnect', { network: network.id, container: c.containerId, force: true }, { success: t('network.detail.disconnected', { name: c.name }) })
    busy = false
    if (ok !== false) {
      await refresh()
      refreshNetworks()
    }
  }
</script>

<div class="fixed inset-0 z-[70] flex items-start justify-center bg-black/45 p-4 pt-[8vh] backdrop-blur-sm" role="presentation" onclick={(e) => e.target === e.currentTarget && !busy && onClose()}>
  <div class="flex max-h-[84vh] w-full max-w-lg flex-col overflow-hidden rounded-xl border border-[var(--border)] bg-[var(--panel)] shadow-[var(--shadow-lg)]" role="dialog" aria-modal="true" aria-label={t('network.detail.aria')} tabindex="-1" use:trapFocus>
    <div class="flex shrink-0 items-center gap-2.5 border-b border-[var(--border)] px-5 py-3.5">
      <Icon name="network" size={16} class="shrink-0 text-[var(--text-3)]" />
      <div class="min-w-0">
        <h2 class="mono truncate text-[14px] font-semibold tracking-tight">{network.name}</h2>
        <p class="text-[11px] text-[var(--text-3)]">{network.driver} · {network.scope}{network.internal ? t('network.detail.internalSuffix') : ''}</p>
      </div>
      <button class="btn btn-default btn-sm ml-auto" onclick={onClose} disabled={busy}>{t('common.close')}</button>
    </div>

    <div class="min-h-0 flex-1 overflow-auto p-5">
      {#if loading && !detail}
        <p class="text-[13px] text-[var(--text-3)]">{t('common.loading')}</p>
      {:else if error}
        <p class="text-[13px] text-[var(--red)]">{error}</p>
      {:else if detail}
        <!-- Addressing -->
        <div class="eyebrow mb-2">{t('network.detail.addressing')}</div>
        {#if detail.ipam?.length}
          <div class="flex flex-col gap-2">
            {#each detail.ipam as ip}
              <div class="mono flex flex-wrap gap-x-5 gap-y-0.5 rounded-lg border border-[var(--border)] bg-[var(--panel-2)] px-3 py-2 text-[12px]">
                <span><span class="text-[var(--text-3)]">{t('network.detail.subnet')}</span> {ip.subnet || 'n/a'}</span>
                <span><span class="text-[var(--text-3)]">{t('network.detail.gateway')}</span> {ip.gateway || 'n/a'}</span>
              </div>
            {/each}
          </div>
        {:else}
          <p class="text-[12px] text-[var(--text-3)]">{t('network.detail.noAddressing')}</p>
        {/if}

        <!-- Attached containers -->
        <div class="eyebrow mb-2 mt-5">{t('network.detail.attached')}</div>
        {#if detail.containers?.length}
          <div class="flex flex-col gap-1.5">
            {#each detail.containers as c (c.containerId)}
              <div class="flex items-center gap-3 rounded-lg border border-[var(--border)] px-3 py-2">
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-[13px] font-medium">{c.name}</span>
                  {#if c.ipv4}<span class="mono block truncate text-[11px] text-[var(--text-3)]">{c.ipv4}</span>{/if}
                </span>
                <button class="btn btn-danger btn-sm shrink-0" onclick={() => disconnect(c)} disabled={busy}>{t('network.detail.disconnect')}</button>
              </div>
            {/each}
          </div>
        {:else}
          <p class="text-[12px] text-[var(--text-3)]">{t('network.detail.noContainers')}</p>
        {/if}

        <!-- Connect a container -->
        <div class="eyebrow mb-2 mt-5">{t('network.detail.connectTitle')}</div>
        {#if candidates.length}
          <div class="flex gap-2">
            <select bind:value={pick} class="input flex-1 cursor-pointer" disabled={busy}>
              <option value="" disabled>{t('network.detail.pickPlaceholder')}</option>
              {#each candidates as c (c.id)}<option value={c.name}>{c.name}</option>{/each}
            </select>
            <button class="btn btn-primary btn-sm shrink-0" onclick={connect} disabled={!pick || busy}>{t('network.detail.connect')}</button>
          </div>
        {:else}
          <p class="text-[12px] text-[var(--text-3)]">{t('network.detail.allAttached')}</p>
        {/if}
      {/if}
    </div>
  </div>
</div>
