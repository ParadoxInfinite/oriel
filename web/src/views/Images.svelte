<script>
  import {
    images, startImagePrune, isPinnedImage,
    toast, createSort, sortRows, fmt, ImageActions,
  } from '../platform/index.js'
  import { btn, btnDanger, btnPrimary, action } from '../lib/ui.js'
  import ResourceTable from '../components/ResourceTable.svelte'
  import PullDialog from '../components/PullDialog.svelte'
  import PrunePreview from '../components/PrunePreview.svelte'
  import LogsDrawer from '../components/LogsDrawer.svelte'
  import StateBadge from '../components/StateBadge.svelte'

  const { bytes, relativeTime, shortRef, shortId } = fmt

  let showPull = $state(false)
  let pullRef = $state('')
  let pruneItems = $state(null) // null = closed
  let drawerContainer = $state(null) // container shown in the logs drawer

  // Tag / used-by / remove behaviour lives in the shared controller; markup binds to it.
  const ia = new ImageActions()
  const openTag = (img) => ia.openTag(img)
  const applyTag = () => ia.applyTag()
  const untag = (img, tag) => ia.untag(img, tag)
  const removeImage = (img) => ia.removeImage(img)

  const columns = [
    { label: 'Repository', key: 'repo', get: (i) => i.tags[0] },
    { label: 'Size', key: 'size', get: (i) => i.size },
    { label: 'In use', key: 'inuse', get: (i) => i.containers },
    { label: 'Created', key: 'created', get: (i) => i.created },
    { label: 'Actions', right: true },
  ]
  const sort = createSort('repo')
  const sorted = $derived(sortRows(images.list, columns, sort))

  // Untagged dangling layers, identified by short id in the prune list.
  const dangling = $derived(images.list.filter((i) => i.tags.length === 1 && i.tags[0] === '<none>'))

  function startPull(ref) {
    pullRef = ref
    showPull = true
  }

  // Open the prune preview: list the dangling (untagged) images so they can be
  // reviewed and individually deselected before anything is deleted.
  function openPrune() {
    if (!dangling.length) {
      toast('No dangling images to prune', 'info')
      return
    }
    pruneItems = dangling.map((i) => ({
      id: i.id,
      primary: shortId(i.id),
      secondary: `untagged · ${relativeTime(i.created)}`,
      size: i.size,
    }))
  }
  // Prune runs as a background job (survives refresh, cancellable); progress
  // shows in the op overlay.
  function doPrune(chosen) {
    startImagePrune(chosen.map((it) => ({ id: it.id, size: it.size })))
  }
</script>

<div class="flex flex-col">
  <div class="mb-4 flex items-center justify-between">
    <span class="text-xs text-muted">{images.list.length} images</span>
    <div class="flex gap-2">
      <button class={btn} onclick={openPrune}>Prune dangling</button>
      <button class={btnPrimary} onclick={() => startPull('')}>Pull image</button>
    </div>
  </div>

  <ResourceTable {columns} {sort} store={images} empty="No images">
    {#each sorted as img (img.id)}
      <tr class="rowx border-t border-border">
        <td class="max-w-[22rem] px-4 py-2.5">
          {#each img.tags as t (t)}
            <div class="group/tag flex items-center gap-1.5">
              <span class="truncate font-mono text-[13px]" title={t}>{shortRef(t)}</span>
              {#if t !== '<none>' && img.tags.length > 1}
                <button
                  class="shrink-0 rounded p-0.5 text-faint opacity-0 transition hover:text-danger group-hover/tag:opacity-100"
                  title="Remove just this tag"
                  aria-label="Remove tag {t}"
                  onclick={() => untag(img, t)}
                >
                  <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M18 6 6 18M6 6l12 12" /></svg>
                </button>
              {/if}
            </div>
          {/each}
        </td>
        <td class="px-4 py-2.5 font-mono text-xs text-muted">{bytes(img.size)}</td>
        <td class="px-4 py-2.5 text-xs text-muted">
          {#if img.containers > 0}
            <button class="font-mono font-medium text-accent hover:underline" title="See which containers use this image" onclick={() => (ia.usedByImage = img)}>{img.containers} →</button>
          {:else}—{/if}
        </td>
        <td class="px-4 py-2.5 text-xs text-muted">{relativeTime(img.created)}</td>
        <td class="px-4 py-2.5 text-right">
          <div class="flex items-center justify-end gap-1.5">
            {#if isPinnedImage(img)}
              <button class={action('accent')} title="Tag this digest-pinned image so it shows a name" onclick={() => openTag(img)}>Tag</button>
            {:else if img.tags[0] !== '<none>'}
              <button class={action('accent')} title="Re-pull this tag from its registry (fails clearly if there's no such repo — e.g. a locally-built image)" onclick={() => startPull(img.tags[0])}>Pull</button>
            {/if}
            <button class={btnDanger} onclick={() => removeImage(img)}>{img.tags.length > 1 ? 'Remove all' : 'Remove'}</button>
          </div>
        </td>
      </tr>
    {/each}
  </ResourceTable>
</div>

{#if showPull}
  <PullDialog initial={pullRef} onClose={() => (showPull = false)} />
{/if}

{#if pruneItems}
  <PrunePreview
    title="Prune dangling images"
    note="Untagged image layers left behind by rebuilds and re-pulls."
    items={pruneItems}
    onClose={() => (pruneItems = null)}
    onPrune={doPrune}
  />
{/if}

<svelte:window
  onkeydown={(e) => {
    if (e.key !== 'Escape') return
    if (ia.tagImage) ia.tagImage = null
    else if (ia.usedByImage) ia.usedByImage = null
  }}
/>

{#if ia.usedByImage}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/55 p-4" role="presentation" onclick={(e) => e.target === e.currentTarget && (ia.usedByImage = null)}>
    <div class="flex max-h-[80vh] w-full max-w-md flex-col overflow-hidden rounded-[--radius] border border-border bg-surface shadow-2xl">
      <div class="border-b border-border px-5 py-3">
        <h2 class="display text-sm font-semibold tracking-tight">Used by {ia.usingContainers.length} container{ia.usingContainers.length === 1 ? '' : 's'}</h2>
        <p class="mt-0.5 truncate font-mono text-[11px] text-faint">{ia.usedByImage.tags?.[0] && ia.usedByImage.tags[0] !== '<none>' ? ia.usedByImage.tags[0] : shortId(ia.usedByImage.id)}</p>
      </div>
      <div class="min-h-0 flex-1 overflow-auto">
        {#each ia.usingContainers as ct (ct.id)}
          <button class="flex w-full items-center gap-3 border-b border-border/60 px-5 py-3 text-left transition-colors last:border-0 hover:bg-surface-2/40" onclick={() => { drawerContainer = ct; ia.usedByImage = null }}>
            <StateBadge state={ct.state} />
            <span class="min-w-0 flex-1">
              <span class="block truncate text-[13px] font-medium text-fg">{ct.name}</span>
              <span class="block truncate text-[11px] text-faint">{ct.status}{ct.project ? ` · ${ct.project}` : ''}</span>
            </span>
            <svg class="shrink-0 text-faint" viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6" /></svg>
          </button>
        {:else}
          <div class="px-5 py-10 text-center text-sm text-muted">No matching containers found — the list may be refreshing.</div>
        {/each}
      </div>
    </div>
  </div>
{/if}

{#if ia.tagImage}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/55 p-4" role="presentation" onclick={(e) => e.target === e.currentTarget && (ia.tagImage = null)}>
    <div class="w-full max-w-md overflow-hidden rounded-[--radius] border border-border bg-surface shadow-2xl">
      <div class="border-b border-border px-5 py-3">
        <h2 class="display text-sm font-semibold tracking-tight">Tag image</h2>
        <p class="mt-0.5 truncate font-mono text-[11px] text-faint" title={ia.tagImage.tags?.[0]}>{shortId(ia.tagImage.id)}</p>
      </div>
      <div class="p-5">
        <p class="mb-2 text-xs text-muted">This image is pinned by digest, so it shows no name. Give it a <span class="font-mono">repository:tag</span> to label it locally.</p>
        <!-- svelte-ignore a11y_autofocus -->
        <input
          class="w-full rounded-md border border-border bg-surface-2 px-3 py-2 font-mono text-[13px] text-fg outline-none focus:border-accent"
          bind:value={ia.tagRef}
          autofocus
          spellcheck="false"
          placeholder="repo/name:tag"
          onkeydown={(e) => e.key === 'Enter' && applyTag()}
        />
      </div>
      <div class="flex justify-end gap-2 border-t border-border px-5 py-3">
        <button class="rounded-md border border-border px-3 py-1.5 text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg" onclick={() => (ia.tagImage = null)}>Cancel</button>
        <button class={btnPrimary} onclick={applyTag} disabled={!ia.tagRef.trim() || ia.tagging}>{ia.tagging ? 'Tagging…' : 'Tag'}</button>
      </div>
    </div>
  </div>
{/if}

{#if drawerContainer}
  <LogsDrawer container={drawerContainer} onClose={() => (drawerContainer = null)} />
{/if}
