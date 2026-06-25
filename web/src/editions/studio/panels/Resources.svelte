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
    startImagePrune,
    startVolumePrune,
    isPinnedImage,
    ImageActions,
    trapFocus,
  } from '../../../platform/index.js'
  import Icon from '../lib/Icon.svelte'
  import SortHeader from '../lib/SortHeader.svelte'
  import PrunePreview from '../lib/PrunePreview.svelte'
  import PullDialog from '../lib/PullDialog.svelte'
  import StatusPill from '../lib/StatusPill.svelte'
  import ContainerDrawer from '../lib/ContainerDrawer.svelte'

  let { kind } = $props()

  // Image tag / used-by / remove behaviour lives in the shared controller.
  const ia = new ImageActions()
  const openTag = (img) => ia.openTag(img)
  const applyTag = () => ia.applyTag()

  const BUILTIN_NET = new Set(['bridge', 'host', 'none'])
  const shortId = (id) => (id || '').replace(/^sha256:/, '').slice(0, 12)

  // Per-kind schema: columns (with sort keys + colgroup widths), default sort,
  // remove + prune behaviour.
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
        { label: 'In use', key: 'inuse', get: (i) => i.containers, right: true, w: '92px', render: (i) => (i.containers > 0 ? String(i.containers) : ', ') },
        { label: 'Created', key: 'created', get: (i) => i.created, right: true, w: '140px', render: (i) => fmt.relativeTime(i.created) },
      ],
      removable: () => true,
      remove: (i) => ia.removeImage(i),
      untag: (img, tag) => ia.untag(img, tag),
      prune: {
        label: 'Prune dangling',
        title: 'Prune dangling images',
        note: 'Removes only untagged (<none>) layers left by rebuilds and re-pulls. Tagged images you are not running are not "dangling" and are kept.',
        empty: 'No dangling (untagged) images to prune',
        collect: () =>
          images.list
            .filter((i) => i.tags.length === 1 && i.tags[0] === '<none>')
            .map((i) => ({ id: i.id, primary: shortId(i.id), secondary: `untagged · ${fmt.relativeTime(i.created)}`, size: i.size })),
        // Background job: survives refresh, cancellable, progress in the op overlay.
        run: (chosen) => startImagePrune(chosen.map((it) => ({ id: it.id, size: it.size }))),
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
        empty: 'No unused volumes to prune',
        collect: async () => {
          const list = await apiGet('/api/volumes/prune/preview')
          return list.map((v) => ({ id: v.name, primary: v.name, secondary: 'unused · data will be deleted', size: v.size }))
        },
        run: (chosen) => startVolumePrune(chosen.map((it) => ({ id: it.id, size: it.size }))),
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
  let drawerContainer = $state(null) // container shown in the drawer

  async function openPrune() {
    try {
      const items = await c.prune.collect()
      if (!items.length) {
        toast(c.prune.empty ?? `No ${titles[kind]} to prune`, 'info')
        return
      }
      pruneItems = items
    } catch (e) {
      toast(e.message, 'error')
    }
  }
</script>

<div class="mx-auto flex max-w-5xl flex-col gap-4">
  <div class="rise flex flex-wrap items-center gap-3">
    <span class="text-[13px] text-[var(--text-2)]"><span class="font-semibold text-[var(--text)]">{c.store.list.length}</span> {titles[kind]}</span>
    <div class="flex flex-wrap gap-2 sm:ml-auto">
      {#if c.prune}
        <button class="btn btn-default btn-sm" onclick={openPrune} title={c.prune.note}><Icon name="broom" size={14} /> {c.prune.label}</button>
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
    <!-- Table layout: sm and up -->
    <div class="rise card hidden overflow-x-auto sm:block" style="animation-delay:40ms">
      <table class="w-full min-w-[600px] table-fixed border-collapse">
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
                      {#each item.tags as t (t)}
                        <div class="group/tag flex items-center gap-1.5">
                          <span class="mono truncate text-[13px] font-medium text-[var(--text)]" title={t}>{fmt.shortRef(t)}</span>
                          {#if t !== '<none>' && item.tags.length > 1}
                            <button class="shrink-0 rounded p-0.5 text-[var(--text-3)] opacity-0 transition hover:text-[var(--red)] group-hover/tag:opacity-100" title="Remove this tag" aria-label="Remove tag {t}" onclick={() => c.untag(item, t)}>
                              <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M18 6 6 18M6 6l12 12" /></svg>
                            </button>
                          {/if}
                        </div>
                      {/each}
                    </div>
                  {:else if col.key === 'inuse' && item.containers > 0}
                    <button class="mono tnum text-[12.5px] font-medium text-[var(--accent)] hover:underline" title="See which containers use this image" onclick={() => (ia.usedByImage = item)}>{item.containers} →</button>
                  {:else}
                    <span class="{col.mono ? 'mono' : ''} {col.right ? 'tnum' : ''} {col.strong ? 'text-[13px] font-medium text-[var(--text)]' : 'text-[12.5px] text-[var(--text-2)]'} {col.grow ? 'block truncate' : ''}">{col.render ? col.render(item) : col.get(item)}</span>
                    {#if col.badge?.(item)}<span class="chip ml-2">{col.badge(item)}</span>{/if}
                  {/if}
                </td>
              {/each}
              <td class="px-4 py-2.5">
                <div class="flex items-center justify-end gap-1.5">
                  {#if kind === 'images' && isPinnedImage(item)}
                    <button class="btn btn-default btn-icon btn-sm" title="Tag this digest-pinned image" aria-label="Tag" onclick={() => openTag(item)}><Icon name="tag" size={14} /></button>
                  {:else if kind === 'images' && item.tags?.[0] && item.tags[0] !== '<none>'}
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

    <!-- Card layout: below sm (phones). Same rows, reflowed so nothing scrolls sideways. -->
    <div class="rise flex flex-col gap-2.5 sm:hidden" style="animation-delay:40ms">
      {#each rows as item (c.key(item))}{@render rcard(item)}{/each}
    </div>
  {/if}
</div>

{#snippet rcard(item)}
  {@const head = c.cols[0]}
  {@const showActions = kind === 'images' || c.removable(item)}
  <div class="card p-3">
    <div class="flex items-start gap-2">
      <Icon name={c.icon} size={15} class="mt-0.5 shrink-0 text-[var(--text-3)]" />
      <div class="min-w-0 flex-1">
        {#if head.tags}
          <div class="flex flex-col gap-0.5">
            {#each item.tags as t (t)}
              <span class="mono block truncate text-[13px] font-medium text-[var(--text)]" title={t}>{fmt.shortRef(t)}</span>
            {/each}
          </div>
        {:else}
          <div class="flex items-center gap-2">
            <span class="mono truncate text-[13px] font-medium text-[var(--text)]" title={head.get(item)}>{head.render ? head.render(item) : head.get(item)}</span>
            {#if head.badge?.(item)}<span class="chip shrink-0">{head.badge(item)}</span>{/if}
          </div>
        {/if}
      </div>
    </div>
    <dl class="mt-2.5 flex flex-col gap-1 text-[12px]">
      {#each c.cols.slice(1) as col}
        <div class="flex items-baseline justify-between gap-3">
          <dt class="shrink-0 text-[var(--text-3)]">{col.label}</dt>
          <dd class="mono tnum min-w-0 truncate text-right text-[var(--text-2)]" title={col.get(item)}>
            {#if col.key === 'inuse' && item.containers > 0}
              <button class="font-medium text-[var(--accent)] hover:underline" title="See which containers use this image" onclick={() => (ia.usedByImage = item)}>{item.containers} →</button>
            {:else}
              {col.render ? col.render(item) : col.get(item)}
            {/if}
          </dd>
        </div>
      {/each}
    </dl>
    {#if showActions}
      <div class="mt-3 flex justify-end gap-2">
        {#if kind === 'images' && isPinnedImage(item)}
          <button class="btn btn-default btn-sm" onclick={() => openTag(item)}><Icon name="tag" size={14} /> Tag</button>
        {:else if kind === 'images' && item.tags?.[0] && item.tags[0] !== '<none>'}
          <button class="btn btn-default btn-sm" onclick={() => (pullRef = item.tags[0])}><Icon name="download" size={14} /> Re-pull</button>
        {/if}
        {#if c.removable(item)}
          <button class="btn btn-danger btn-sm" onclick={() => c.remove(item)}><Icon name="trash" size={14} /> Remove</button>
        {/if}
      </div>
    {/if}
  </div>
{/snippet}

{#if pruneItems}
  <PrunePreview title={c.prune.title} note={c.prune.note} items={pruneItems} onClose={() => (pruneItems = null)} onPrune={c.prune.run} />
{/if}

{#if pullRef !== null}
  <PullDialog initial={pullRef} onClose={() => (pullRef = null)} />
{/if}

<svelte:window
  onkeydown={(e) => {
    if (e.key !== 'Escape') return
    if (ia.tagImage) ia.tagImage = null
    else if (ia.usedByImage) ia.usedByImage = null
  }}
/>

{#if ia.usedByImage}
  <div class="fixed inset-0 z-[70] flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm" role="presentation" onclick={(e) => e.target === e.currentTarget && (ia.usedByImage = null)}>
    <div class="flex max-h-[80vh] w-full max-w-md flex-col overflow-hidden rounded-xl border border-[var(--border)] bg-[var(--panel)] shadow-[var(--shadow-lg)]" role="dialog" aria-modal="true" aria-label="Image used by" tabindex="-1" use:trapFocus>
      <div class="border-b border-[var(--border)] px-5 py-3.5">
        <h2 class="text-[14px] font-semibold tracking-tight">Used by {ia.usingContainers.length} container{ia.usingContainers.length === 1 ? '' : 's'}</h2>
        <p class="mono mt-0.5 truncate text-[11px] text-[var(--text-3)]">{ia.usedByImage.tags?.[0] && ia.usedByImage.tags[0] !== '<none>' ? ia.usedByImage.tags[0] : shortId(ia.usedByImage.id)}</p>
      </div>
      <div class="min-h-0 flex-1 overflow-auto">
        {#each ia.usingContainers as ct (ct.id)}
          <button class="flex w-full items-center gap-3 border-b border-[var(--border)] px-5 py-3 text-left transition-colors last:border-0 hover:bg-[var(--panel-2)]" onclick={() => { drawerContainer = ct; ia.usedByImage = null }}>
            <StatusPill state={ct.state} />
            <span class="min-w-0 flex-1">
              <span class="block truncate text-[13px] font-medium text-[var(--text)]">{ct.name}</span>
              <span class="block truncate text-[11px] text-[var(--text-3)]">{ct.status}{ct.project ? ` · ${ct.project}` : ''}</span>
            </span>
            <svg class="shrink-0 text-[var(--text-3)]" viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6" /></svg>
          </button>
        {:else}
          <div class="px-5 py-10 text-center text-sm text-[var(--text-2)]">No matching containers found, the list may be refreshing.</div>
        {/each}
      </div>
    </div>
  </div>
{/if}

{#if drawerContainer}
  <ContainerDrawer container={drawerContainer} onClose={() => (drawerContainer = null)} />
{/if}

{#if ia.tagImage}
  <div class="fixed inset-0 z-[70] flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm" role="presentation" onclick={(e) => e.target === e.currentTarget && (ia.tagImage = null)}>
    <div class="w-full max-w-md overflow-hidden rounded-xl border border-[var(--border)] bg-[var(--panel)] shadow-[var(--shadow-lg)]" role="dialog" aria-modal="true" aria-label="Tag image" tabindex="-1" use:trapFocus>
      <div class="border-b border-[var(--border)] px-5 py-3.5">
        <h2 class="text-[14px] font-semibold tracking-tight">Tag image</h2>
        <p class="mono mt-0.5 truncate text-[11px] text-[var(--text-3)]" title={ia.tagImage.tags?.[0]}>{shortId(ia.tagImage.id)}</p>
      </div>
      <div class="p-5">
        <p class="mb-2 text-[12px] text-[var(--text-2)]">This image is pinned by digest, so it shows no name. Give it a <span class="mono">repository:tag</span> to label it locally.</p>
        <!-- svelte-ignore a11y_autofocus -->
        <input
          class="mono w-full rounded-lg border border-[var(--border)] bg-[var(--panel-2)] px-3 py-2 text-[13px] text-[var(--text)] outline-none focus:border-[var(--accent)]"
          bind:value={ia.tagRef}
          autofocus
          spellcheck="false"
          placeholder="repo/name:tag"
          onkeydown={(e) => e.key === 'Enter' && applyTag()}
        />
      </div>
      <div class="flex justify-end gap-2 border-t border-[var(--border)] px-5 py-3">
        <button class="btn btn-default btn-sm" onclick={() => (ia.tagImage = null)}>Cancel</button>
        <button class="btn btn-primary btn-sm" onclick={applyTag} disabled={!ia.tagRef.trim() || ia.tagging}>{ia.tagging ? 'Tagging…' : 'Tag'}</button>
      </div>
    </div>
  </div>
{/if}
