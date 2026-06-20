<script>
  import {
    images,
    volumes,
    networks,
    refreshImages,
    refreshVolumes,
    refreshNetworks,
    invoke,
    confirm,
    toast,
    apiGet,
    fmt,
    createSort,
    sortRows,
  } from '../../../platform/index.js'
  import Icon from '../lib/Icon.svelte'
  import SortHeader from '../lib/SortHeader.svelte'
  import PrunePreview from '../lib/PrunePreview.svelte'
  import PullDialog from '../lib/PullDialog.svelte'

  let { kind } = $props()

  const BUILTIN_NET = new Set(['bridge', 'host', 'none'])
  const shortId = (id) => (id || '').replace(/^sha256:/, '').slice(0, 12)

  // Per-kind schema: columns (with sort keys + colgroup widths), default sort,
  // remove + prune behaviour. Mirrors Classic's resource views.
  const config = {
    images: {
      store: images,
      icon: 'harddrive',
      key: (i) => i.id,
      defaultSort: 'repo',
      pullable: true,
      cols: [
        { label: 'Repository', key: 'repo', get: (i) => i.tags[0], grow: true, strong: true, mono: true, tags: true },
        { label: 'Size', key: 'size', get: (i) => i.size, right: true, mono: true, w: '110px', render: (i) => fmt.bytes(i.size) },
        { label: 'In use', key: 'inuse', get: (i) => i.containers, right: true, w: '92px', render: (i) => (i.containers > 0 ? String(i.containers) : '—') },
        { label: 'Created', key: 'created', get: (i) => i.created, right: true, w: '140px', render: (i) => fmt.relativeTime(i.created) },
      ],
      removable: () => true,
      remove: async (i) => {
        const inUse = i.containers > 0
        const ok = await confirm({
          title: 'Remove image?',
          message: `${i.tags.length > 1 ? `All ${i.tags.length} tags` : `“${i.tags[0]}”`} will be deleted.${inUse ? ` Used by ${i.containers} container(s); it will be force-removed.` : ''}`,
          confirmLabel: 'Remove',
        })
        if (!ok) return
        await invoke('image.remove', { id: i.id, force: inUse }, { success: `Removed ${i.tags[0]}` })
        refreshImages()
      },
      untag: async (img, tag) => {
        const last = img.tags.length <= 1
        const ok = await confirm({
          title: 'Remove this tag?',
          message: `“${tag}” will be removed.${last ? ' It is the only tag, so the image itself will be deleted.' : ' The image and its other tags stay.'}`,
          confirmLabel: 'Remove tag',
        })
        if (!ok) return
        await invoke('image.remove', { id: tag, force: false }, { success: `Removed ${tag}` })
        refreshImages()
      },
      prune: {
        label: 'Prune dangling',
        title: 'Prune dangling images',
        note: 'Untagged image layers left behind by rebuilds and re-pulls.',
        collect: () =>
          images.list
            .filter((i) => i.tags.length === 1 && i.tags[0] === '<none>')
            .map((i) => ({ id: i.id, primary: shortId(i.id), secondary: `untagged · ${fmt.relativeTime(i.created)}`, size: i.size })),
        run: async (chosen) => {
          let removed = 0
          let reclaimed = 0
          for (const it of chosen) {
            if (await invoke('image.remove', { id: it.id, force: true })) {
              removed++
              reclaimed += it.size
            }
          }
          if (removed) toast(`Pruned ${removed} image(s), reclaimed ${fmt.bytes(reclaimed)}`, 'ok')
          refreshImages()
        },
      },
    },
    volumes: {
      store: volumes,
      icon: 'database',
      key: (v) => v.name,
      defaultSort: 'name',
      cols: [
        { label: 'Name', key: 'name', get: (v) => v.name, grow: true, strong: true, mono: true },
        { label: 'Driver', key: 'driver', get: (v) => v.driver, w: '130px' },
        { label: 'Mountpoint', key: 'mount', get: (v) => v.mountpoint, grow: true, mono: true },
      ],
      removable: () => true,
      remove: async (v) => {
        const ok = await confirm({
          title: 'Remove volume?',
          message: `“${v.name}” and all of its data will be permanently deleted. This can't be undone.`,
          confirmLabel: 'Remove',
        })
        if (!ok) return
        await invoke('volume.remove', { name: v.name, force: false }, { success: `Removed ${v.name}` })
        refreshVolumes()
      },
      prune: {
        label: 'Prune unused',
        title: 'Prune unused volumes',
        note: 'Volumes no container references. Their data is deleted permanently.',
        collect: async () => {
          const list = await apiGet('/api/volumes/prune/preview')
          return list.map((v) => ({ id: v.name, primary: v.name, secondary: 'unused · data will be deleted', size: v.size }))
        },
        run: async (chosen) => {
          let removed = 0
          let reclaimed = 0
          for (const it of chosen) {
            if (await invoke('volume.remove', { name: it.id, force: false })) {
              removed++
              reclaimed += it.size
            }
          }
          if (removed) toast(`Pruned ${removed} volume(s), reclaimed ${fmt.bytes(reclaimed)}`, 'ok')
          refreshVolumes()
        },
      },
    },
    networks: {
      store: networks,
      icon: 'network',
      key: (n) => n.id,
      defaultSort: 'name',
      cols: [
        { label: 'Name', key: 'name', get: (n) => n.name, grow: true, strong: true, mono: true, badge: (n) => (n.internal ? 'internal' : null) },
        { label: 'Driver', key: 'driver', get: (n) => n.driver, w: '150px' },
        { label: 'Scope', key: 'scope', get: (n) => n.scope, w: '130px' },
      ],
      removable: (n) => !BUILTIN_NET.has(n.name),
      remove: async (n) => {
        const ok = await confirm({
          title: 'Remove network?',
          message: `“${n.name}” will be removed. Containers attached to it will lose this network.`,
          confirmLabel: 'Remove',
        })
        if (!ok) return
        await invoke('network.remove', { id: n.id }, { success: `Removed ${n.name}` })
        refreshNetworks()
      },
    },
  }

  const c = $derived(config[kind])
  const sort = createSort('name')
  // Reset the active sort to each kind's default when switching resource.
  $effect(() => {
    sort.key = c.defaultSort
    sort.dir = 'asc'
  })
  const rows = $derived(sortRows(c.store.list, c.cols, sort))
  const titles = { images: 'images', volumes: 'volumes', networks: 'networks' }

  let pullRef = $state(null) // null = closed; string = open with that initial ref
  let pruneItems = $state(null) // null = closed
  async function openPrune() {
    try {
      const items = await c.prune.collect()
      if (!items.length) {
        toast(`No ${titles[kind]} to prune`, 'info')
        return
      }
      pruneItems = items
    } catch (e) {
      toast(e.message, 'error')
    }
  }
</script>

<div class="mx-auto flex max-w-5xl flex-col gap-4">
  <div class="rise flex items-center gap-3">
    <span class="text-[13px] text-[var(--text-2)]"><span class="font-semibold text-[var(--text)]">{c.store.list.length}</span> {titles[kind]}</span>
    <div class="ml-auto flex gap-2">
      {#if c.prune}
        <button class="btn btn-default btn-sm" onclick={openPrune}><Icon name="broom" size={14} /> {c.prune.label}</button>
      {/if}
      {#if c.pullable}
        <button class="btn btn-primary btn-sm" onclick={() => (pullRef = '')}><Icon name="download" size={14} /> Pull image</button>
      {/if}
    </div>
  </div>

  {#if c.store.error}
    <div class="card border-[color-mix(in_srgb,var(--red)_40%,var(--border))] p-4 text-sm text-[var(--red)]">{c.store.error}</div>
  {:else if c.store.list.length === 0}
    <div class="card grid place-items-center gap-2 py-20 text-center">
      <Icon name={c.icon} size={26} class="text-[var(--text-3)]" />
      <p class="text-sm text-[var(--text-2)]">No {titles[kind]} yet.</p>
    </div>
  {:else}
    <div class="rise card overflow-hidden" style="animation-delay:40ms">
      <table class="w-full table-fixed border-collapse">
        <colgroup>
          {#each c.cols as col}<col style={col.w ? `width:${col.w}` : ''} />{/each}
          <col style="width:{kind === 'images' ? '108px' : '64px'}" />
        </colgroup>
        <thead>
          <tr class="border-b border-[var(--border)]">
            {#each c.cols as col}<th class="th {col.right ? 'text-right' : ''}"><SortHeader {col} {sort} /></th>{/each}
            <th class="th"></th>
          </tr>
        </thead>
        <tbody>
          {#each rows as item (c.key(item))}
            <tr class="tr border-b border-[var(--border)] last:border-0">
              {#each c.cols as col}
                <td class="px-4 py-2.5 {col.right ? 'text-right' : ''}">
                  {#if col.tags}
                    <div class="flex flex-col gap-0.5">
                      {#each item.tags as t}
                        <div class="group/tag flex items-center gap-1.5">
                          <span class="mono truncate text-[13px] font-medium text-[var(--text)]">{t}</span>
                          {#if t !== '<none>' && item.tags.length > 1}
                            <button class="shrink-0 rounded p-0.5 text-[var(--text-3)] opacity-0 transition hover:text-[var(--red)] group-hover/tag:opacity-100" title="Remove this tag" aria-label="Remove tag {t}" onclick={() => c.untag(item, t)}>
                              <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M18 6 6 18M6 6l12 12" /></svg>
                            </button>
                          {/if}
                        </div>
                      {/each}
                    </div>
                  {:else}
                    <span class="{col.mono ? 'mono' : ''} {col.right ? 'tnum' : ''} {col.strong ? 'text-[13px] font-medium text-[var(--text)]' : 'text-[12.5px] text-[var(--text-2)]'} {col.grow ? 'block truncate' : ''}">{col.render ? col.render(item) : col.get(item)}</span>
                    {#if col.badge?.(item)}<span class="chip ml-2">{col.badge(item)}</span>{/if}
                  {/if}
                </td>
              {/each}
              <td class="px-4 py-2.5">
                <div class="flex items-center justify-end gap-1.5">
                  {#if kind === 'images' && item.tags?.[0] && item.tags[0] !== '<none>'}
                    <button class="btn btn-default btn-icon btn-sm" title="Re-pull from registry" aria-label="Re-pull" onclick={() => (pullRef = item.tags[0])}><Icon name="download" size={14} /></button>
                  {/if}
                  {#if c.removable(item)}
                    <button class="btn btn-danger btn-icon btn-sm" title="Remove" aria-label="Remove" onclick={() => c.remove(item)}><Icon name="trash" size={14} /></button>
                  {/if}
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

{#if pruneItems}
  <PrunePreview title={c.prune.title} note={c.prune.note} items={pruneItems} onClose={() => (pruneItems = null)} onPrune={c.prune.run} />
{/if}

{#if pullRef !== null}
  <PullDialog initial={pullRef} onClose={() => (pullRef = null)} />
{/if}
