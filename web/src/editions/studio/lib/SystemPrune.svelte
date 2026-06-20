<script>
  import { onMount } from 'svelte'
  import { apiGet, startSystemPrune, fmt } from '../../../platform/index.js'
  import Icon from './Icon.svelte'

  let { onClose } = $props()

  let df = $state(null)
  let loadErr = $state('')
  let includeVolumes = $state(false)

  onMount(async () => {
    try {
      df = await apiGet('/api/system/df')
    } catch (e) {
      loadErr = e.message
    }
  })

  const total = $derived((df?.reclaimable ?? 0) + (includeVolumes ? (df?.volumesSize ?? 0) : 0))

  const items = $derived(
    df
      ? [
          { label: 'Stopped containers', count: df.stoppedContainers, size: df.containersSize },
          { label: 'Dangling images', count: df.danglingImages, size: df.imagesSize },
          { label: 'Build cache', count: null, size: df.buildCacheSize },
          { label: 'Unused networks', count: null, size: 0 },
        ]
      : []
  )

  function run() {
    const vol = includeVolumes
    onClose() // hand off to the background-job op overlay (streams per-step progress)
    startSystemPrune(vol)
  }
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && onClose()} />

<div class="fixed inset-0 z-[70] flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm" role="presentation" onclick={(e) => e.target === e.currentTarget && onClose()}>
  <div class="w-full max-w-md overflow-hidden rounded-xl border border-[var(--border)] bg-[var(--panel)] shadow-[var(--shadow-lg)]">
    <div class="flex items-center gap-2.5 border-b border-[var(--border)] px-5 py-3.5">
      <div class="grid h-7 w-7 place-items-center rounded-lg bg-[var(--amber-tint)] text-[var(--amber)]"><Icon name="broom" size={15} /></div>
      <h2 class="text-[14px] font-semibold tracking-tight">Reclaim disk space</h2>
    </div>

    <div class="p-5">
      <div class="flex items-start gap-2.5 rounded-lg border border-[color-mix(in_srgb,var(--amber)_35%,var(--border))] bg-[var(--amber-tint)] px-3 py-2.5 text-[12px] text-[var(--text-2)]">
        <Icon name="broom" size={14} class="mt-0.5 shrink-0 text-[var(--amber)]" />
        <span>This permanently removes <strong>all stopped containers</strong>, unused networks, dangling images and build cache. It can't be undone.</span>
      </div>

      {#if loadErr}
        <div class="mono mt-3 rounded-lg border border-[color-mix(in_srgb,var(--red)_35%,var(--border))] bg-[var(--red-tint)] px-3 py-2 text-[12px] text-[var(--red)]">{loadErr}</div>
      {:else if !df}
        <div class="mt-4 flex items-center gap-2 text-sm text-[var(--text-3)]"><span class="h-4 w-4 animate-spin rounded-full border-2 border-[var(--border-strong)] border-t-[var(--accent)]"></span> Calculating…</div>
      {:else}
        <div class="mt-4 card overflow-hidden">
          {#each items as it, i}
            <div class="flex items-center justify-between gap-3 px-4 py-2.5 {i ? 'border-t border-[var(--border)]' : ''}">
              <span class="text-[13px] text-[var(--text)]">{it.label}{#if it.count != null}<span class="ml-1.5 text-[var(--text-3)]">· {it.count}</span>{/if}</span>
              {#if it.size > 0}<span class="mono tnum text-[12.5px] text-[var(--text-2)]">{fmt.bytes(it.size)}</span>{/if}
            </div>
          {/each}
        </div>

        <label class="mt-3 flex cursor-pointer items-start gap-2.5 rounded-lg border border-[var(--border)] px-3 py-2.5">
          <input type="checkbox" bind:checked={includeVolumes} class="mt-0.5 h-4 w-4" style="accent-color:var(--red)" />
          <span class="min-w-0">
            <span class="block text-[13px] font-medium text-[var(--text)]">Also remove unused volumes{#if df.unusedVolumes}<span class="ml-1 text-[var(--text-3)]">· {df.unusedVolumes} · {fmt.bytes(df.volumesSize)}</span>{/if}</span>
            <span class="block text-[11.5px] text-[var(--red)]">Deletes data in volumes no container references. Irreversible.</span>
          </span>
        </label>
      {/if}
    </div>

    <div class="flex items-center justify-between gap-3 border-t border-[var(--border)] px-5 py-3">
      <span class="text-[12px] text-[var(--text-2)]">Reclaims ≈ <span class="mono font-medium text-[var(--green)]">{fmt.bytes(total)}</span></span>
      <div class="flex gap-2">
        <button class="btn btn-default btn-sm" onclick={onClose}>Cancel</button>
        <button class="btn btn-sm" style="background:var(--red);color:#fff" onclick={run} disabled={!df}>Prune</button>
      </div>
    </div>
  </div>
</div>
