<script>
  import { containers, refreshContainers } from '../lib/containers.svelte.js'
  import { stats } from '../lib/live.svelte.js'
  import { invoke } from '../lib/invoke.js'
  import { confirm } from '../lib/confirm.svelte.js'
  import { bytes, relativeTime } from '../lib/format.js'
  import { createSort, sortRows } from '../lib/sort.svelte.js'
  import { stackColor } from '../lib/stackColor.js'
  import { action } from '../lib/ui.js'
  import StateBadge from '../components/StateBadge.svelte'
  import SortHeader from '../components/SortHeader.svelte'
  import LogsDrawer from '../components/LogsDrawer.svelte'

  let filter = $state('')
  let selected = $state(null)
  let groupByStack = $state(true)
  let collapsed = $state({})

  const groupKey = (name) => name || '__standalone__'
  function toggleGroup(name) {
    const k = groupKey(name)
    collapsed[k] = !collapsed[k]
  }

  // Sortable columns. CPU/memory read from the live stats stream; missing values
  // sink to the bottom. The trailing Ports/Actions columns aren't sortable.
  const sort = createSort('name')
  const sortCols = [
    { key: 'name', get: (c) => c.name },
    { key: 'state', get: (c) => c.state },
    { key: 'cpu', get: (c) => stats.byId[c.id]?.cpu ?? -1 },
    { key: 'memory', get: (c) => stats.byId[c.id]?.mem ?? -1 },
    { key: 'uptime', get: (c) => c.created ?? 0 },
  ]
  const headers = [
    { label: 'Name', key: 'name' },
    { label: 'State', key: 'state' },
    { label: 'CPU', key: 'cpu' },
    { label: 'Memory', key: 'memory' },
    { label: 'Uptime', key: 'uptime' },
    { label: 'Ports' },
    { label: 'Actions' },
  ]

  const filtered = $derived(
    containers.list.filter((c) => {
      const q = filter.toLowerCase()
      return !q || c.name.toLowerCase().includes(q) || c.image.toLowerCase().includes(q)
    })
  )

  // Sorted flat list, or sorted-then-grouped by compose project (Standalone last).
  const view = $derived.by(() => {
    const rows = sortRows(filtered, sortCols, sort)
    if (!groupByStack) return { grouped: false, rows }
    const map = new Map()
    for (const c of rows) {
      const k = c.project || ''
      if (!map.has(k)) map.set(k, [])
      map.get(k).push(c)
    }
    const groups = [...map.entries()]
      .map(([name, items]) => ({ name, items }))
      .sort((a, b) => (!a.name ? 1 : !b.name ? -1 : a.name.localeCompare(b.name)))
    return { grouped: true, groups }
  })

  async function act(e, c, verb, tool, args) {
    e.stopPropagation()
    if (tool === 'container.remove') {
      const running = c.state === 'running'
      const ok = await confirm({
        title: 'Remove container?',
        message: `“${c.name}” will be permanently removed.${running ? ' It is running and will be force-stopped first.' : ''}`,
        confirmLabel: 'Remove',
      })
      if (!ok) return
    }
    await invoke(tool, args, { success: `${verb} · ${c.name}` })
    refreshContainers()
  }

  function portLabel(p) {
    return p.public ? `${p.public}→${p.private}` : `${p.private}`
  }
  function portTitle(p) {
    return p.public
      ? `Host ${p.public}  →  Container ${p.private}  ·  ${p.type}`
      : `Container ${p.private} · ${p.type} (not published)`
  }
</script>

<div class="flex flex-col">
  <div class="mb-4 flex items-center gap-3">
    <input
      bind:value={filter}
      placeholder="Filter containers…"
      class="w-64 rounded-[--radius] border border-border bg-surface px-3 py-1.5 text-sm outline-none placeholder:text-muted focus:border-accent/50"
    />
    <span class="text-xs text-muted">{filtered.length} containers</span>
    <button
      class="pop ml-auto rounded-[--radius] border px-2.5 py-1.5 text-xs transition-colors {groupByStack
        ? 'border-accent/50 bg-accent/10 text-accent'
        : 'border-border text-muted hover:text-fg'}"
      onclick={() => (groupByStack = !groupByStack)}
    >
      Group by stack
    </button>
  </div>

  {#if containers.error}
    <div class="rounded-[--radius] border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">
      {containers.error}
    </div>
  {:else if filtered.length === 0}
    <div class="rounded-[--radius] border border-dashed border-border py-20 text-center text-sm text-muted">
      No containers
    </div>
  {:else}
    <div class="overflow-hidden rounded-[--radius] border border-border">
      <table class="w-full text-sm">
        <thead class="bg-surface text-xs uppercase tracking-wide text-muted">
          <tr>
            {#each headers as col, i}
              <th class="px-3 py-2.5 font-medium {i === 0 ? 'pl-4 text-left' : i === headers.length - 1 ? 'text-right' : 'text-left'}" title={col.label === 'Ports' ? 'host → container' : null}>
                <SortHeader {col} {sort} />
              </th>
            {/each}
          </tr>
        </thead>
        <tbody>
          {#if view.grouped}
            {#each view.groups as g (groupKey(g.name))}
              {@const open = !collapsed[groupKey(g.name)]}
              {@const sc = stackColor(g.name)}
              <tr class="border-t border-border bg-surface/70" style="box-shadow: inset 3px 0 0 0 {sc}">
                <td colspan={headers.length} class="p-0">
                  <button
                    class="flex w-full items-center gap-2 py-2 pl-4 pr-3 text-left transition-colors hover:bg-surface-2/50"
                    style="color:{sc}"
                    onclick={() => toggleGroup(g.name)}
                  >
                    <span class="font-mono text-xs font-semibold">{g.name || 'Standalone'}</span>
                    <span class="rounded-full bg-surface-2 px-1.5 py-0.5 text-[10px] text-muted">{g.items.length}</span>
                    <svg
                      class="ml-auto shrink-0 transition-transform duration-200 {open ? 'rotate-90' : ''}"
                      viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"
                    ><path d="m9 6 6 6-6 6" /></svg>
                  </button>
                </td>
              </tr>
              {#if open}
                {#each g.items as c (c.id)}
                  {@render row(c, stackColor(g.name))}
                {/each}
              {/if}
            {/each}
          {:else}
            {#each view.rows as c (c.id)}
              {@render row(c, null)}
            {/each}
          {/if}
        </tbody>
      </table>
    </div>
  {/if}
</div>

{#snippet row(c, accent)}
  {@const running = c.state === 'running'}
  {@const st = stats.byId[c.id]}
  <tr
    class="{accent ? 'rowg' : 'rowx'} cursor-pointer border-t border-border"
    style={accent ? `--bar: ${accent}` : undefined}
    onclick={() => (selected = c)}
  >
    <td class="max-w-[18rem] py-2.5 pl-4 pr-3">
      <div class="truncate font-mono text-[13px]">{c.name}</div>
      <div class="truncate text-xs text-muted">{c.image}</div>
    </td>
    <td class="px-3 py-2.5"><StateBadge state={c.state} /></td>
    <td class="px-3 py-2.5 font-mono text-xs text-muted">{running && st ? `${st.cpu.toFixed(1)}%` : '—'}</td>
    <td class="px-3 py-2.5 font-mono text-xs text-muted">{running && st ? bytes(st.mem) : '—'}</td>
    <td class="px-3 py-2.5 text-xs text-muted" title={`created ${relativeTime(c.created)}`}>
      <span class="whitespace-nowrap">{c.status || '—'}</span>
    </td>
    <td class="px-3 py-2.5">
      <div class="flex flex-wrap gap-1">
        {#each c.ports.filter((p) => p.public) as p}
          <span class="cursor-help rounded bg-surface-2 px-1.5 py-0.5 font-mono text-[11px] text-muted" title={portTitle(p)}>{portLabel(p)}</span>
        {/each}
      </div>
    </td>
    <td class="px-3 py-2.5">
      <div class="flex justify-end gap-1">
        {#if running}
          <button class={action('accent')} onclick={(e) => act(e, c, 'Restart', 'container.restart', { id: c.id })}>Restart</button>
          <button class={action('warn')} onclick={(e) => act(e, c, 'Stop', 'container.stop', { id: c.id })}>Stop</button>
        {:else}
          <button class={action('ok')} onclick={(e) => act(e, c, 'Start', 'container.start', { id: c.id })}>Start</button>
        {/if}
        <button class={action('danger')} onclick={(e) => act(e, c, 'Remove', 'container.remove', { id: c.id, force: running })}>Remove</button>
      </div>
    </td>
  </tr>
{/snippet}

{#if selected}
  <LogsDrawer container={selected} onClose={() => (selected = null)} />
{/if}
