<script>
  import { networks, refreshNetworks } from '../lib/resources.svelte.js'
  import { invoke } from '../lib/invoke.js'
  import { confirm } from '../lib/confirm.svelte.js'
  import { btnDanger } from '../lib/ui.js'
  import { createSort, sortRows } from '../lib/sort.svelte.js'
  import ResourceTable from '../components/ResourceTable.svelte'

  // Built-in networks can't be removed; don't offer the action for them.
  const builtin = new Set(['bridge', 'host', 'none'])

  const columns = [
    { label: 'Name', key: 'name', get: (n) => n.name },
    { label: 'Driver', key: 'driver', get: (n) => n.driver },
    { label: 'Scope', key: 'scope', get: (n) => n.scope },
    { label: 'Actions', right: true },
  ]
  const sort = createSort('name')
  const sorted = $derived(sortRows(networks.list, columns, sort))

  async function remove(n) {
    const ok = await confirm({
      title: 'Remove network?',
      message: `“${n.name}” will be removed. Containers attached to it will lose this network.`,
      confirmLabel: 'Remove',
    })
    if (!ok) return
    await invoke('network.remove', { id: n.id }, { success: `Removed ${n.name}` })
    refreshNetworks()
  }
</script>

<div class="flex flex-col">
  <div class="mb-4">
    <span class="text-xs text-muted">{networks.list.length} networks</span>
  </div>

  <ResourceTable {columns} {sort} store={networks} empty="No networks">
    {#each sorted as n (n.id)}
      <tr class="rowx border-t border-border">
        <td class="px-4 py-2.5">
          <span class="font-mono text-[13px]">{n.name}</span>
          {#if n.internal}<span class="ml-2 rounded bg-surface-2 px-1.5 py-0.5 text-[11px] text-muted">internal</span>{/if}
        </td>
        <td class="px-4 py-2.5 text-xs text-muted">{n.driver}</td>
        <td class="px-4 py-2.5 text-xs text-muted">{n.scope}</td>
        <td class="px-4 py-2.5 text-right">
          {#if !builtin.has(n.name)}
            <button class={btnDanger} onclick={() => remove(n)}>Remove</button>
          {/if}
        </td>
      </tr>
    {/each}
  </ResourceTable>
</div>
