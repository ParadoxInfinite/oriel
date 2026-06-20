<script>
  import { images, refreshImages } from '../lib/resources.svelte.js'
  import { invoke } from '../lib/invoke.js'
  import { confirm } from '../lib/confirm.svelte.js'
  import { toast } from '../lib/toast.svelte.js'
  import { bytes, relativeTime } from '../lib/format.js'
  import { btn, btnDanger, btnPrimary, action } from '../lib/ui.js'
  import { createSort, sortRows } from '../lib/sort.svelte.js'
  import ResourceTable from '../components/ResourceTable.svelte'
  import PullDialog from '../components/PullDialog.svelte'
  import PrunePreview from '../components/PrunePreview.svelte'

  let showPull = $state(false)
  let pullRef = $state('')
  let pruneItems = $state(null) // null = closed

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
  const shortId = (id) => (id || '').replace(/^sha256:/, '').slice(0, 12)
  const dangling = $derived(images.list.filter((i) => i.tags.length === 1 && i.tags[0] === '<none>'))

  function startPull(ref) {
    pullRef = ref
    showPull = true
  }

  async function untag(img, tag) {
    const last = img.tags.length <= 1
    const ok = await confirm({
      title: 'Remove this tag?',
      message: `“${tag}” will be removed.${last ? ' It is the only tag, so the image itself will be deleted.' : ' The image and its other tags stay.'}`,
      confirmLabel: 'Remove tag',
    })
    if (!ok) return
    await invoke('image.remove', { id: tag, force: false }, { success: `Removed ${tag}` })
    refreshImages()
  }

  async function removeImage(img) {
    const inUse = img.containers > 0
    const many = img.tags.length > 1
    const ok = await confirm({
      title: 'Remove image?',
      message: `${many ? `All ${img.tags.length} tags of this image` : `“${img.tags[0]}”`} will be deleted.${inUse ? ` It is used by ${img.containers} container(s) and will be force-removed.` : ''}`,
      confirmLabel: 'Remove',
    })
    if (!ok) return
    await invoke('image.remove', { id: img.id, force: inUse }, { success: `Removed ${img.tags[0]}` })
    refreshImages()
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
  async function doPrune(chosen, progress) {
    let removed = 0
    let reclaimed = 0
    let done = 0
    for (const it of chosen) {
      if (await invoke('image.remove', { id: it.id, force: true })) {
        removed++
        reclaimed += it.size
      }
      progress?.(++done)
    }
    if (removed) toast(`Pruned ${removed} image(s), reclaimed ${bytes(reclaimed)}`, 'ok')
    refreshImages()
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
          {#each img.tags as t}
            <div class="group/tag flex items-center gap-1.5">
              <span class="truncate font-mono text-[13px]">{t}</span>
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
        <td class="px-4 py-2.5 text-xs text-muted">{img.containers > 0 ? `${img.containers}` : '—'}</td>
        <td class="px-4 py-2.5 text-xs text-muted">{relativeTime(img.created)}</td>
        <td class="px-4 py-2.5 text-right">
          <div class="flex items-center justify-end gap-1.5">
            {#if img.tags[0] !== '<none>'}
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
