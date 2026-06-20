<script>
  import { containers, refreshContainers, stats, invoke, confirm, fmt, createSort, sortRows } from '../../../platform/index.js'
  import Icon from '../lib/Icon.svelte'
  import StatusPill from '../lib/StatusPill.svelte'
  import SortHeader from '../lib/SortHeader.svelte'
  import ContainerDrawer from '../lib/ContainerDrawer.svelte'

  let selected = $state(null)

  let filter = $state('')
  let groupByStack = $state(true)
  let collapsed = $state({})

  // Pull the exit code out of docker's status string ("Exited (137) 2 days ago").
  const exitCode = (status) => {
    const m = /Exited \((\d+)\)/.exec(status || '')
    return m ? Number(m[1]) : null
  }

  const sort = createSort('name')
  // `w` feeds the colgroup so numeric columns stay tight and Name absorbs the rest.
  const columns = [
    { label: 'Name', key: 'name', get: (c) => c.name, w: null },
    { label: 'Status', key: 'state', get: (c) => c.state, w: '150px' },
    { label: 'CPU', key: 'cpu', get: (c) => stats.byId[c.id]?.cpu ?? -1, right: true, w: '92px' },
    { label: 'Memory', key: 'memory', get: (c) => stats.byId[c.id]?.mem ?? -1, right: true, w: '104px' },
    { label: 'Ports', w: '150px' },
    { label: '', w: '188px' },
  ]

  const filtered = $derived(
    containers.list.filter((c) => {
      const q = filter.trim().toLowerCase()
      return !q || c.name.toLowerCase().includes(q) || c.image.toLowerCase().includes(q)
    })
  )
  const rows = $derived(sortRows(filtered, columns, sort))

  const groups = $derived.by(() => {
    if (!groupByStack) return null
    const map = new Map()
    for (const c of rows) {
      const k = c.project || ''
      if (!map.has(k)) map.set(k, [])
      map.get(k).push(c)
    }
    return [...map.entries()]
      .map(([name, items]) => ({ name, items }))
      .sort((a, b) => (!a.name ? 1 : !b.name ? -1 : a.name.localeCompare(b.name)))
  })

  async function act(e, c, tool, verb) {
    e.stopPropagation()
    const isRunning = c.state === 'running'
    if (tool === 'container.remove') {
      const ok = await confirm({
        title: 'Remove container?',
        message: `“${c.name}” will be permanently removed.${isRunning ? ' It is running and will be force-stopped first.' : ''}`,
        confirmLabel: 'Remove',
      })
      if (!ok) return
    }
    const args = tool === 'container.remove' ? { id: c.id, force: isRunning } : { id: c.id }
    await invoke(tool, args, { success: `${verb} · ${c.name}` })
    refreshContainers()
  }

  // ── Bulk selection ──────────────────────────────────────────────────────────
  let selectedIds = $state(new Set())
  const allSelected = $derived(rows.length > 0 && rows.every((c) => selectedIds.has(c.id)))
  function toggleOne(id, e) {
    e?.stopPropagation()
    const s = new Set(selectedIds)
    s.has(id) ? s.delete(id) : s.add(id)
    selectedIds = s
  }
  function toggleAll() {
    selectedIds = allSelected ? new Set() : new Set(rows.map((c) => c.id))
  }
  async function bulk(tool, verb, predicate) {
    const targets = rows.filter((c) => selectedIds.has(c.id) && predicate(c))
    if (!targets.length) return
    if (tool === 'container.remove') {
      const ok = await confirm({
        title: `Remove ${targets.length} container(s)?`,
        message: `${targets.length} selected container(s) will be permanently removed. Running ones are force-stopped first.`,
        confirmLabel: 'Remove',
      })
      if (!ok) return
    }
    for (const c of targets) {
      const args = tool === 'container.remove' ? { id: c.id, force: c.state === 'running' } : { id: c.id }
      await invoke(tool, args, { success: `${verb} · ${c.name}` })
    }
    selectedIds = new Set()
    refreshContainers()
  }
</script>

<div class="mx-auto flex max-w-5xl flex-col gap-4">
  <div class="rise flex flex-wrap items-center gap-3">
    <div class="relative">
      <Icon name="box" size={15} class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-3)]" />
      <input bind:value={filter} placeholder="Search containers…" class="input has-icon w-72" />
    </div>
    <span class="text-[13px] text-[var(--text-3)]">{filtered.length} of {containers.list.length}</span>
    <button class="btn {groupByStack ? 'btn-primary' : 'btn-default'} ml-auto btn-sm" onclick={() => (groupByStack = !groupByStack)}>
      <Icon name="layers" size={14} /> Group by stack
    </button>
  </div>

  {#if selectedIds.size}
    <div class="flex flex-wrap items-center gap-2 rounded-lg border border-[var(--accent)] bg-[var(--accent-tint)] px-3 py-2">
      <span class="text-[13px] font-medium text-[var(--accent)]">{selectedIds.size} selected</span>
      <div class="ml-auto flex gap-1.5">
        <button class="btn btn-default btn-sm" onclick={() => bulk('container.start', 'Started', (c) => c.state !== 'running')}><Icon name="play" size={13} /> Start</button>
        <button class="btn btn-default btn-sm" onclick={() => bulk('container.stop', 'Stopped', (c) => c.state === 'running')}><Icon name="stop" size={13} /> Stop</button>
        <button class="btn btn-default btn-sm" onclick={() => bulk('container.restart', 'Restarted', (c) => c.state === 'running')}><Icon name="restart" size={13} /> Restart</button>
        <button class="btn btn-danger btn-sm" onclick={() => bulk('container.remove', 'Removed', () => true)}><Icon name="trash" size={13} /> Remove</button>
        <button class="btn btn-ghost btn-sm" onclick={() => (selectedIds = new Set())}>Clear</button>
      </div>
    </div>
  {/if}

  {#if containers.error}
    <div class="card border-[color-mix(in_srgb,var(--red)_40%,var(--border))] p-4 text-sm text-[var(--red)]">{containers.error}</div>
  {:else if filtered.length === 0}
    <div class="card grid place-items-center gap-2 py-20 text-center">
      <Icon name="box" size={26} class="text-[var(--text-3)]" />
      <p class="text-sm text-[var(--text-2)]">{containers.list.length ? 'No containers match your search.' : 'No containers yet.'}</p>
    </div>
  {:else}
    <div class="rise card overflow-hidden" style="animation-delay:40ms">
      <table class="w-full table-fixed border-collapse">
        <colgroup>
          <col style="width:42px" />
          {#each columns as col}<col style={col.w ? `width:${col.w}` : ''} />{/each}
        </colgroup>
        <thead>
          <tr class="border-b border-[var(--border)]">
            <th class="th"><input type="checkbox" checked={allSelected} onchange={toggleAll} class="h-3.5 w-3.5 align-middle" style="accent-color:var(--accent)" aria-label="Select all" /></th>
            {#each columns as col}
              <th class="th {col.right ? 'text-right' : ''}"><SortHeader {col} {sort} /></th>
            {/each}
          </tr>
        </thead>
        <tbody>
          {#if groups}
            {#each groups as g (g.name || '__solo__')}
              {@const key = g.name || '__solo__'}
              {@const open = !collapsed[key]}
              <tr class="border-b border-[var(--border)] bg-[var(--panel-2)]">
                <td colspan="7" class="p-0">
                  <button class="flex w-full items-center gap-2 px-4 py-2 text-left" onclick={() => (collapsed[key] = open)}>
                    <svg class="text-[var(--text-3)] transition-transform {open ? 'rotate-90' : ''}" viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="m9 6 6 6-6 6" /></svg>
                    <Icon name="layers" size={13} class="text-[var(--text-3)]" />
                    <span class="text-[13px] font-semibold">{g.name || 'Standalone'}</span>
                    <span class="count">{g.items.length}</span>
                  </button>
                </td>
              </tr>
              {#if open}{#each g.items as c (c.id)}{@render row(c)}{/each}{/if}
            {/each}
          {:else}
            {#each rows as c (c.id)}{@render row(c)}{/each}
          {/if}
        </tbody>
      </table>
    </div>
  {/if}
</div>

{#snippet row(c)}
  {@const running = c.state === 'running'}
  {@const st = stats.byId[c.id]}
  {@const code = exitCode(c.status)}
  <tr class="tr cursor-pointer border-b border-[var(--border)] last:border-0 {selectedIds.has(c.id) ? 'bg-[var(--accent-tint)]' : ''}" onclick={() => (selected = c)}>
    <td class="px-4 py-2.5" onclick={(e) => e.stopPropagation()}>
      <input type="checkbox" checked={selectedIds.has(c.id)} onchange={(e) => toggleOne(c.id, e)} class="h-3.5 w-3.5 align-middle" style="accent-color:var(--accent)" aria-label="Select {c.name}" />
    </td>
    <td class="px-4 py-2.5">
      <div class="truncate text-[13px] font-medium">{c.name}</div>
      <div class="mono truncate text-[11.5px] text-[var(--text-3)]">{c.image}</div>
    </td>
    <td class="px-4 py-2.5">
      <div class="flex items-center gap-1.5">
        <StatusPill state={c.state} />
        {#if code != null}
          <span class="mono text-[11px] font-medium {code === 0 ? 'text-[var(--text-3)]' : 'text-[var(--red)]'}">({code})</span>
        {/if}
      </div>
      {#if !running && c.status}
        <div class="mono mt-0.5 truncate text-[10.5px] text-[var(--text-3)]" title={c.status}>{c.status}</div>
      {/if}
    </td>
    <td class="mono tnum px-4 py-2.5 text-right text-[12.5px] text-[var(--text-2)]">{running && st ? `${st.cpu.toFixed(1)}%` : '—'}</td>
    <td class="mono tnum px-4 py-2.5 text-right text-[12.5px] text-[var(--text-2)]">{running && st ? fmt.bytes(st.mem) : '—'}</td>
    <td class="px-4 py-2.5">
      <div class="flex flex-wrap gap-1">
        {#each c.ports.filter((p) => p.public) as p}
          <span class="chip" title={`host ${p.public} → container ${p.private} · ${p.type}`}>{p.public}→{p.private}</span>
        {/each}
      </div>
    </td>
    <td class="px-4 py-2.5">
      <div class="flex items-center justify-end gap-1.5">
        {#if running}
          <button class="btn btn-default btn-sm" onclick={(e) => act(e, c, 'container.restart', 'Restarted')}><Icon name="restart" size={13} /> Restart</button>
          <button class="btn btn-default btn-sm" onclick={(e) => act(e, c, 'container.stop', 'Stopped')}><Icon name="stop" size={13} /> Stop</button>
        {:else}
          <button class="btn btn-default btn-sm" onclick={(e) => act(e, c, 'container.start', 'Started')}><Icon name="play" size={13} /> Start</button>
        {/if}
        <button class="btn btn-danger btn-icon btn-sm" title="Remove" aria-label="Remove" onclick={(e) => act(e, c, 'container.remove', 'Removed')}><Icon name="trash" size={14} /></button>
      </div>
    </td>
  </tr>
{/snippet}

{#if selected}
  <ContainerDrawer container={selected} onClose={() => (selected = null)} />
{/if}
