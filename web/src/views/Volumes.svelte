<script>
  import { volumes, refreshVolumes } from '../lib/resources.svelte.js'
  import { startVolumePrune } from '../lib/op.svelte.js'
  import { invoke } from '../lib/invoke.js'
  import { apiGet } from '../lib/api.js'
  import { toast } from '../lib/toast.svelte.js'
  import { confirm } from '../lib/confirm.svelte.js'
  import { btn, btnDanger } from '../lib/ui.js'
  import { createSort, sortRows } from '../lib/sort.svelte.js'
  import ResourceTable from '../components/ResourceTable.svelte'
  import PrunePreview from '../components/PrunePreview.svelte'

  let pruneItems = $state(null) // null = closed

  const columns = [
    { label: 'Name', key: 'name', get: (v) => v.name },
    { label: 'Driver', key: 'driver', get: (v) => v.driver },
    { label: 'Mountpoint', key: 'mount', get: (v) => v.mountpoint },
    { label: 'Actions', right: true },
  ]
  const sort = createSort('name')
  const sorted = $derived(sortRows(volumes.list, columns, sort))

  async function remove(v) {
    const ok = await confirm({
      title: 'Remove volume?',
      message: `“${v.name}” and all data stored in it will be permanently deleted. This cannot be undone.`,
      confirmLabel: 'Remove',
    })
    if (!ok) return
    await invoke('volume.remove', { name: v.name, force: false }, { success: `Removed ${v.name}` })
    refreshVolumes()
  }
  // Open the prune preview: the backend lists unused volumes (with sizes) so
  // they can be reviewed and individually deselected before any data is deleted.
  async function openPrune() {
    let list
    try {
      list = await apiGet('/api/volumes/prune/preview')
    } catch (e) {
      toast(e.message, 'error')
      return
    }
    if (!list.length) {
      toast('No unused volumes to prune', 'info')
      return
    }
    pruneItems = list.map((v) => ({
      id: v.name,
      primary: v.name,
      secondary: 'unused · data will be deleted',
      size: v.size,
    }))
  }
  // Prune runs as a background job (survives refresh, cancellable); progress
  // shows in the op overlay.
  function doPrune(chosen) {
    startVolumePrune(chosen.map((it) => ({ id: it.id, size: it.size })))
  }
</script>

<div class="flex flex-col">
  <div class="mb-4 flex items-center justify-between">
    <span class="text-xs text-muted">{volumes.list.length} volumes</span>
    <button class={btn} onclick={openPrune}>Prune unused</button>
  </div>

  <ResourceTable {columns} {sort} store={volumes} empty="No volumes">
    {#each sorted as v (v.name)}
      <tr class="rowx border-t border-border">
        <td class="max-w-[20rem] px-4 py-2.5"><div class="truncate font-mono text-[13px]">{v.name}</div></td>
        <td class="px-4 py-2.5 text-xs text-muted">{v.driver}</td>
        <td class="max-w-[20rem] px-4 py-2.5"><div class="truncate font-mono text-xs text-muted">{v.mountpoint}</div></td>
        <td class="px-4 py-2.5 text-right">
          <button class={btnDanger} onclick={() => remove(v)}>Remove</button>
        </td>
      </tr>
    {/each}
  </ResourceTable>
</div>

{#if pruneItems}
  <PrunePreview
    title="Prune unused volumes"
    note="Volumes no container references. Their data is deleted permanently."
    items={pruneItems}
    onClose={() => (pruneItems = null)}
    onPrune={doPrune}
  />
{/if}
