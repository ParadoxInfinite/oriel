<script>
  import { onMount } from 'svelte'
  import { apiGet, startSystemPrune, fmt } from '../../../platform/index.js'
  import Icon from './Icon.svelte'

  let { onClose } = $props()

  let df = $state(null)
  let loadErr = $state('')

  // Each category is opted in. Volumes default off (irreversible); the rest on.
  // cacheall extends the build-cache prune from dangling-only to all unused cache.
  let sel = $state({ containers: true, images: true, networks: true, cache: true, cacheall: false, volumes: false })

  onMount(async () => {
    try {
      df = await apiGet('/api/system/df')
    } catch (e) {
      loadErr = e.message
    }
  })

  // Simple size-bearing rows. Build cache is handled separately (dangling vs all).
  const rows = $derived(
    df
      ? [
          { key: 'containers', label: 'Stopped containers', count: df.stoppedContainers, size: df.containersSize },
          { key: 'images', label: 'Dangling images', count: df.danglingImages, size: df.imagesSize },
          { key: 'networks', label: 'Unused networks', count: null, size: 0 },
          { key: 'volumes', label: 'Unused volumes', count: df.unusedVolumes, size: df.volumesSize, danger: true },
        ]
      : []
  )
  // Build-cache size is the all-unused figure; only count it when the override is on,
  // since dangling-only reclaim can't be predicted up front.
  const total = $derived(rows.reduce((a, r) => a + (sel[r.key] ? r.size : 0), 0) + (sel.cache && sel.cacheall ? (df?.buildCacheSize ?? 0) : 0))
  const anySelected = $derived(sel.containers || sel.images || sel.networks || sel.cache || sel.volumes)

  function run() {
    onClose() // hand off to the background-job op overlay (streams per-step progress)
    startSystemPrune({ ...sel })
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
      <p class="text-[12px] text-[var(--text-2)]">Choose what to remove. Everything selected is permanently deleted and can't be undone.</p>

      {#if loadErr}
        <div class="mono mt-3 rounded-lg border border-[color-mix(in_srgb,var(--red)_35%,var(--border))] bg-[var(--red-tint)] px-3 py-2 text-[12px] text-[var(--red)]">{loadErr}</div>
      {:else if !df}
        <div class="mt-4 flex items-center gap-2 text-sm text-[var(--text-3)]"><span class="h-4 w-4 animate-spin rounded-full border-2 border-[var(--border-strong)] border-t-[var(--accent)]"></span> Calculating…</div>
      {:else}
        <div class="mt-4 card overflow-hidden">
          <!-- Containers, images, networks -->
          {#each rows.slice(0, 3) as r, i (r.key)}
            <label class="flex cursor-pointer items-center gap-3 px-4 py-2.5 transition-colors hover:bg-[var(--panel-2)] {i ? 'border-t border-[var(--border)]' : ''}">
              <input type="checkbox" bind:checked={sel[r.key]} class="h-4 w-4" style="accent-color:var(--accent)" />
              <span class="flex flex-1 items-center justify-between gap-3">
                <span class="text-[13px] text-[var(--text)]">{r.label}{#if r.count != null}<span class="ml-1.5 text-[var(--text-3)]">· {r.count}</span>{/if}</span>
                {#if r.size > 0}<span class="mono tnum shrink-0 text-[12.5px] text-[var(--text-2)]">{fmt.bytes(r.size)}</span>{/if}
              </span>
            </label>
          {/each}

          <!-- Build cache: dangling by default, with an all-unused override -->
          <div class="border-t border-[var(--border)]">
            <label class="flex cursor-pointer items-center gap-3 px-4 py-2.5 transition-colors hover:bg-[var(--panel-2)]">
              <input type="checkbox" bind:checked={sel.cache} class="h-4 w-4" style="accent-color:var(--accent)" />
              <span class="flex-1 text-[13px] text-[var(--text)]">Build cache <span class="text-[var(--text-3)]">· dangling layers</span></span>
            </label>
            {#if sel.cache}
              <label class="flex cursor-pointer items-start gap-3 border-t border-dashed border-[var(--border)] bg-[var(--panel-2)] px-4 py-2.5 pl-9">
                <input type="checkbox" bind:checked={sel.cacheall} class="mt-0.5 h-3.5 w-3.5" style="accent-color:var(--amber)" />
                <span class="min-w-0 flex-1">
                  <span class="flex items-center justify-between gap-3">
                    <span class="text-[12.5px] text-[var(--text)]">Also remove cache for existing images</span>
                    <span class="mono tnum shrink-0 text-[12.5px] text-[var(--amber)]">{fmt.bytes(df.buildCacheSize)}</span>
                  </span>
                  <span class="mt-0.5 block text-[11px] text-[var(--text-3)]">Frees the most space; the next build of those images is slower.</span>
                </span>
              </label>
            {/if}
          </div>

          <!-- Volumes: irreversible, default off -->
          <label class="flex cursor-pointer items-start gap-3 border-t border-[var(--border)] px-4 py-2.5 transition-colors hover:bg-[var(--panel-2)]">
            <input type="checkbox" bind:checked={sel.volumes} class="mt-0.5 h-4 w-4" style="accent-color:var(--red)" />
            <span class="min-w-0 flex-1">
              <span class="flex items-center justify-between gap-3">
                <span class="text-[13px] font-medium text-[var(--text)]">Unused volumes{#if df.unusedVolumes}<span class="ml-1.5 text-[var(--text-3)]">· {df.unusedVolumes}</span>{/if}</span>
                {#if df.volumesSize > 0}<span class="mono tnum shrink-0 text-[12.5px] text-[var(--red)]">{fmt.bytes(df.volumesSize)}</span>{/if}
              </span>
              <span class="mt-0.5 block text-[11.5px] text-[var(--red)]">Deletes data in volumes no container references. Irreversible.</span>
            </span>
          </label>
        </div>
      {/if}
    </div>

    <div class="flex items-center justify-between gap-3 border-t border-[var(--border)] px-5 py-3">
      <span class="text-[12px] text-[var(--text-2)]">Reclaims ≈ <span class="mono font-medium text-[var(--green)]">{fmt.bytes(total)}</span></span>
      <div class="flex gap-2">
        <button class="btn btn-default btn-sm" onclick={onClose}>Cancel</button>
        <button class="btn btn-sm" style="background:var(--red);color:#fff" onclick={run} disabled={!df || !anySelected}>Prune</button>
      </div>
    </div>
  </div>
</div>
