<script>
  import { palette, closePalette } from '../lib/palette.svelte.js'
  import { registerEscape } from '../lib/modalStack.svelte.js'
  import { trapFocus } from '../lib/focustrap.js'
  import { containers, refreshContainers } from '../lib/containers.svelte.js'
  import {
    images,
    volumes,
    networks,
    refreshImages,
    refreshVolumes,
    refreshNetworks,
  } from '../lib/resources.svelte.js'
  import { navigate, VIEWS } from '../lib/nav.svelte.js'
  import { stacks, refreshStacks } from '../lib/stacks.svelte.js'
  import { stackOp, t, tn } from '../platform/index.js'
  import { fuzzyScore } from '../lib/fuzzy.js'
  import { invoke } from '../lib/invoke.js'
  import { confirm } from '../lib/confirm.svelte.js'
  import { toast } from '../lib/toast.svelte.js'

  let query = $state('')
  let selected = $state(0)
  let inputEl = $state(null)

  // Escape via the shared modal stack so it closes only the top overlay.
  $effect(() => {
    if (palette.open) return registerEscape(closePalette)
  })

  // The palette is a thin client over the tool registry: each entry maps to a
  // {tool, args} call, the same shape /api/invoke runs for UI buttons. We
  // enumerate live entities × the actions valid for them, so
  // the palette only ever offers something that can actually run. Reads (logs,
  // inspect, df) are deliberately absent, those navigate to a view, not here.
  const DEFAULT_NETWORKS = new Set(['bridge', 'host', 'none'])
  const shortId = (id) => (id || '').replace(/^sha256:/, '').slice(0, 12)
  const imageName = (img) => {
    const tag = img.tags?.[0]
    return tag && !tag.includes('<none>') ? tag : shortId(img.id)
  }

  function containerActions(c) {
    const running = c.state === 'running'
    const acts = []
    if (!running) acts.push({ verb: t('action.start'), label: t('palette.startContainer'), tool: 'container.start', args: { id: c.id } })
    if (running) acts.push({ verb: t('action.stop'), label: t('palette.stopContainer'), tool: 'container.stop', args: { id: c.id } })
    if (running) acts.push({ verb: t('action.restart'), label: t('palette.restartContainer'), tool: 'container.restart', args: { id: c.id } })
    acts.push({ verb: t('action.remove'), label: t('palette.removeContainer'), tool: 'container.remove', args: { id: c.id, force: running }, danger: true })
    return acts.map((a) => ({
      ...a,
      target: c.name,
      key: `${a.tool}:${c.id}`,
    }))
  }

  function imageActions(img) {
    const tag = img.tags?.[0]
    const tagged = tag && !tag.includes('<none>')
    // Remove the exact tag shown (untags it, and deletes the image when it's the
    // last tag). Removing by bare id errors on a multi-tagged image.
    return [
      {
        verb: t('action.remove'),
        label: t('palette.removeImage'),
        target: imageName(img),
        tool: 'image.remove',
        args: { id: tagged ? tag : img.id, force: img.containers > 0 },
        danger: true,
        key: `image.remove:${img.id}`,
      },
    ]
  }

  function volumeActions(v) {
    return [
      {
        verb: t('action.remove'),
        label: t('palette.removeVolume'),
        target: v.name,
        tool: 'volume.remove',
        args: { name: v.name, force: false },
        danger: true,
        key: `volume.remove:${v.name}`,
      },
    ]
  }

  function networkActions(n) {
    // The predefined bridge/host/none networks can't be removed, don't offer it.
    if (DEFAULT_NETWORKS.has(n.name)) return []
    return [
      {
        verb: t('action.remove'),
        label: t('palette.removeNetwork'),
        target: n.name,
        tool: 'network.remove',
        args: { id: n.id },
        danger: true,
        key: `network.remove:${n.id}`,
      },
    ]
  }

  // Compose stacks run through the op tray (stackOp), not invoke, so ⌘K gets the
  // same live progress + cancel as the Stacks view buttons. Marked with `op`.
  function stackActions(s) {
    const acts = []
    if (s.running < s.total) acts.push({ verb: t('action.start'), label: t('palette.startStack'), op: 'start' })
    if (s.running > 0) acts.push({ verb: t('action.stop'), label: t('palette.stopStack'), op: 'stop' })
    acts.push({ verb: t('action.restart'), label: t('palette.restartStack'), op: 'restart' })
    acts.push({ verb: t('palette.down'), label: t('palette.downStack'), op: 'down', danger: true })
    return acts.map((a) => ({
      ...a,
      target: s.name,
      key: `stack.${a.op}:${s.name}`,
    }))
  }

  // Entity-free maintenance actions, always available.
  const globalActions = $derived([
    { verb: t('palette.prune'), label: t('palette.pruneImages'), target: t('palette.danglingImages'), tool: 'image.prune', args: {}, danger: true, key: 'image.prune' },
    { verb: t('palette.prune'), label: t('palette.pruneVolumes'), target: t('palette.unusedVolumes'), tool: 'volume.prune', args: {}, danger: true, key: 'volume.prune' },
  ])

  // Navigation entries carry a `nav` payload instead of a tool: they move the
  // active view rather than invoke anything. Reads with no good toast form (logs,
  // disk usage, colima status) route to the view that already shows them.
  function containerLogNav(c) {
    return [
      {
        nav: { view: 'Containers', target: { kind: 'container', container: c, open: 'logs' } },
        label: t('palette.viewLogs'),
        target: c.name,
        key: `logs:${c.id}`,
      },
    ]
  }
  const viewLabel = (v) => t(`nav.${v.toLowerCase()}`)
  const viewJumps = $derived(VIEWS.map((v) => ({ nav: { view: v }, label: t('palette.goTo', { view: viewLabel(v) }), target: '', key: `nav:${v}` })))
  const readJumps = $derived([
    { nav: { view: 'Dashboard' }, label: t('palette.diskUsage'), target: 'dashboard', key: 'nav:disk' },
    { nav: { view: 'Dashboard' }, label: t('palette.colimaStatus'), target: 'dashboard', key: 'nav:colima' },
  ])

  const items = $derived([
    ...containers.list.flatMap(containerActions),
    ...containers.list.flatMap(containerLogNav),
    ...images.list.flatMap(imageActions),
    ...volumes.list.flatMap(volumeActions),
    ...networks.list.flatMap(networkActions),
    ...stacks.list.flatMap(stackActions),
    ...globalActions,
    ...viewJumps,
    ...readJumps,
  ])

  const filtered = $derived.by(() => {
    const q = query.trim()
    // Nothing shows until the user types, the palette opens as an empty prompt
    // rather than dumping every action for every entity.
    if (!q) return []

    const scored = items
      .map((it) => ({ it, score: fuzzyScore(q, `${it.label} ${it.target}`) }))
      .filter((x) => x.score >= 0)
      .sort((a, b) => b.score - a.score || a.it.target.localeCompare(b.it.target))
      .slice(0, 50)
      .map((x) => x.it)

    return scored
  })

  $effect(() => {
    if (palette.open) {
      query = ''
      selected = 0
      // Refresh the entity lists the catalog draws from. Suggestions only appear
      // once the user types, so these have time to land before they're needed.
      refreshContainers()
      refreshImages()
      refreshVolumes()
      refreshNetworks()
      refreshStacks()
      queueMicrotask(() => inputEl?.focus())
    }
  })

  $effect(() => {
    // Keep selection in range as the filtered list changes.
    if (selected >= filtered.length) selected = Math.max(0, filtered.length - 1)
  })

  function confirmMessage(it) {
    if (it.tool === 'image.prune') return t('palette.confirm.pruneImages')
    if (it.tool === 'volume.prune') return t('palette.confirm.pruneVolumes')
    if (it.op === 'down') return t('palette.confirm.stackDown', { name: it.target })
    if (it.tool === 'container.remove' && it.args.force) {
      return t('palette.confirm.containerRemoveRunning', { name: it.target })
    }
    return t('palette.confirm.removeDefault', { name: it.target })
  }

  function refreshAfter(tool) {
    const ns = tool.split('.')[0]
    if (ns === 'container') refreshContainers()
    else if (ns === 'image') refreshImages()
    else if (ns === 'volume') refreshVolumes()
    else if (ns === 'network') refreshNetworks()
  }

  async function run(it) {
    closePalette()
    if (it.nav) {
      navigate(it.nav.view, it.nav.target ?? null)
      return
    }
    if (it.op) {
      if (it.danger) {
        const ok = await confirm({ title: t('palette.confirmTitle', { action: it.label }), message: confirmMessage(it), confirmLabel: it.verb })
        if (!ok) return
      }
      // Streams in the op tray (cancellable), same as the Stacks view buttons.
      stackOp(it.target, it.op, refreshStacks)
      return
    }
    if (it.danger) {
      const ok = await confirm({
        title: t('palette.confirmTitle', { action: it.label }),
        message: confirmMessage(it),
        confirmLabel: it.verb,
      })
      if (!ok) return
    }
    await invoke(it.tool, it.args, { success: t('common.opSuccess', { verb: it.verb, name: it.target }) })
    refreshAfter(it.tool)
  }

  function onKeydown(e) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      selected = Math.max(0, Math.min(selected + 1, filtered.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      selected = Math.max(selected - 1, 0)
    } else if (e.key === 'Enter') {
      e.preventDefault()
      if (filtered[selected]) run(filtered[selected])
    }
  }
</script>

{#if palette.open}
  <div
    class="fixed inset-0 z-50 flex items-start justify-center bg-black/50 p-4 pt-[12vh]"
    onclick={(e) => { if (e.target === e.currentTarget) closePalette() }}
    role="presentation"
  >
    <div
      class="w-full max-w-lg overflow-hidden rounded-[var(--overlay-radius)] border border-border bg-surface shadow-[var(--overlay-shadow)]"
      role="dialog"
      aria-modal="true"
      aria-label={t('palette.ariaLabel')}
      tabindex="-1"
      use:trapFocus
    >
      <input
        bind:this={inputEl}
        bind:value={query}
        onkeydown={onKeydown}
        placeholder={t('palette.placeholder')}
        class="w-full bg-transparent px-4 py-3.5 text-sm outline-none placeholder:text-muted"
      />
      <div class="max-h-80 overflow-auto border-t border-border">
        {#if !query.trim()}
          <div class="px-4 py-6 text-center text-sm text-muted">
            {t('palette.hint')}
          </div>
        {:else if filtered.length === 0}
          <div class="px-4 py-6 text-center text-sm text-muted">{t('palette.noResults')}</div>
        {:else}
          {#each filtered as it, i (it.key)}
            <button
              class="flex w-full items-center gap-2 px-4 py-2.5 text-left text-sm transition-colors
                {i === selected ? 'bg-surface-2' : 'hover:bg-surface-2/50'}"
              onmouseenter={() => (selected = i)}
              onclick={() => run(it)}
            >
              <span class="{it.danger ? 'text-danger' : 'text-fg'}">{it.label}</span>
              <span class="truncate font-mono text-xs text-muted">{it.target}</span>
            </button>
          {/each}
        {/if}
      </div>
      <div
        class="flex items-center justify-between border-t border-border px-4 py-2 text-[11px] text-muted"
      >
        <span>{t('palette.footerHelp')}</span>
        <span>{tn('palette.actionsCount', filtered.length)}</span>
      </div>
    </div>
  </div>
{/if}
