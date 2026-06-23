<script>
  import { palette, closePalette } from '../lib/palette.svelte.js'
  import { registerEscape } from '../lib/modalStack.svelte.js'
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
  import { stackOp } from '../platform/index.js'
  import { fuzzyScore } from '../lib/fuzzy.js'
  import { invoke } from '../lib/invoke.js'
  import { confirm } from '../lib/confirm.svelte.js'
  import { provider, resolveText } from '../lib/provider.svelte.js'
  import { toast } from '../lib/toast.svelte.js'

  let query = $state('')
  let selected = $state(0)
  let inputEl = $state(null)

  // Escape via the shared modal stack so it closes only the top overlay.
  $effect(() => {
    if (palette.open) return registerEscape(closePalette)
  })

  // The palette is a thin client over the tool registry: each entry maps to a
  // {tool, args} call — the same shape /api/invoke runs for UI buttons and a
  // provider emits. We enumerate live entities × the actions valid for them, so
  // the palette only ever offers something that can actually run. Reads (logs,
  // inspect, df) are deliberately absent — those navigate to a view, not here.
  const DEFAULT_NETWORKS = new Set(['bridge', 'host', 'none'])
  const shortId = (id) => (id || '').replace(/^sha256:/, '').slice(0, 12)
  const imageName = (img) => {
    const tag = img.tags?.[0]
    return tag && !tag.includes('<none>') ? tag : shortId(img.id)
  }

  function containerActions(c) {
    const running = c.state === 'running'
    const acts = []
    if (!running) acts.push({ verb: 'Start', tool: 'container.start', args: { id: c.id } })
    if (running) acts.push({ verb: 'Stop', tool: 'container.stop', args: { id: c.id } })
    if (running) acts.push({ verb: 'Restart', tool: 'container.restart', args: { id: c.id } })
    acts.push({ verb: 'Remove', tool: 'container.remove', args: { id: c.id, force: running }, danger: true })
    return acts.map((a) => ({
      ...a,
      label: `${a.verb} container`,
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
        verb: 'Remove',
        label: 'Remove image',
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
        verb: 'Remove',
        label: 'Remove volume',
        target: v.name,
        tool: 'volume.remove',
        args: { name: v.name, force: false },
        danger: true,
        key: `volume.remove:${v.name}`,
      },
    ]
  }

  function networkActions(n) {
    // The predefined bridge/host/none networks can't be removed — don't offer it.
    if (DEFAULT_NETWORKS.has(n.name)) return []
    return [
      {
        verb: 'Remove',
        label: 'Remove network',
        target: n.name,
        tool: 'network.remove',
        args: { id: n.id },
        danger: true,
        key: `network.remove:${n.id}`,
      },
    ]
  }

  // Compose stacks run through the op tray (stackOp), not invoke — so ⌘K gets the
  // same live progress + cancel as the Stacks view buttons. Marked with `op`.
  function stackActions(s) {
    const acts = []
    if (s.running < s.total) acts.push({ verb: 'Start', op: 'start' })
    if (s.running > 0) acts.push({ verb: 'Stop', op: 'stop' })
    acts.push({ verb: 'Restart', op: 'restart' })
    acts.push({ verb: 'Down', op: 'down', danger: true })
    return acts.map((a) => ({
      ...a,
      label: `${a.verb} stack`,
      target: s.name,
      key: `stack.${a.op}:${s.name}`,
    }))
  }

  // Entity-free maintenance actions, always available.
  const globalActions = [
    { verb: 'Prune', label: 'Prune unused images', target: 'dangling images', tool: 'image.prune', args: {}, danger: true, key: 'image.prune' },
    { verb: 'Prune', label: 'Prune unused volumes', target: 'unused volumes', tool: 'volume.prune', args: {}, danger: true, key: 'volume.prune' },
  ]

  // Navigation entries carry a `nav` payload instead of a tool: they move the
  // active view rather than invoke anything. Reads with no good toast form (logs,
  // disk usage, colima status) route to the view that already shows them.
  function containerLogNav(c) {
    return [
      {
        nav: { view: 'Containers', target: { kind: 'container', container: c, open: 'logs' } },
        label: 'View logs',
        target: c.name,
        key: `logs:${c.id}`,
      },
    ]
  }
  const viewJumps = VIEWS.map((v) => ({ nav: { view: v }, label: `Go to ${v}`, target: '', key: `nav:${v}` }))
  const readJumps = [
    { nav: { view: 'Dashboard' }, label: 'Disk usage', target: 'dashboard', key: 'nav:disk' },
    { nav: { view: 'Dashboard' }, label: 'Colima status', target: 'dashboard', key: 'nav:colima' },
  ]

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
    // Nothing shows until the user types — the palette opens as an empty prompt
    // rather than dumping every action for every entity.
    if (!q) return []

    const scored = items
      .map((it) => ({ it, score: fuzzyScore(q, `${it.label} ${it.target}`) }))
      .filter((x) => x.score >= 0)
      .sort((a, b) => b.score - a.score || a.it.target.localeCompare(b.it.target))
      .slice(0, 50)
      .map((x) => x.it)

    // With a provider configured, offer free-text interpretation of the query
    // as the first option — the same execution path, just an NL resolver.
    if (provider.enabled) {
      return [{ ai: true, key: '__ai__', label: 'Interpret', target: q }, ...scored]
    }
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
    if (it.tool === 'image.prune') return 'All dangling images will be permanently removed.'
    if (it.tool === 'volume.prune') return 'All unused volumes will be permanently removed.'
    if (it.op === 'down') return `All containers in “${it.target}” will be stopped and removed.`
    if (it.tool === 'container.remove' && it.args.force) {
      return `“${it.target}” is running. It will be force-stopped, then permanently removed.`
    }
    return `“${it.target}” will be permanently removed.`
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
        const ok = await confirm({ title: `${it.verb} stack?`, message: confirmMessage(it), confirmLabel: it.verb })
        if (!ok) return
      }
      // Streams in the op tray (cancellable), same as the Stacks view buttons.
      stackOp(it.target, it.op, refreshStacks)
      return
    }
    if (it.ai) {
      try {
        const res = await resolveText(it.target)
        if (!res?.call) {
          toast(res?.message || 'No matching command', 'info')
          return
        }
        toast(`${res.call.tool} · ${res.call.args?.id ?? ''}`.trim(), 'ok')
      } catch (e) {
        toast(e.message, 'error')
      }
      refreshContainers()
      return
    }
    if (it.danger) {
      const ok = await confirm({
        title: `${it.label}?`,
        message: confirmMessage(it),
        confirmLabel: it.verb,
      })
      if (!ok) return
    }
    await invoke(it.tool, it.args, { success: `${it.verb} · ${it.target}` })
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
    onclick={closePalette}
    role="presentation"
  >
    <div
      class="w-full max-w-lg overflow-hidden rounded-[var(--overlay-radius)] border border-border bg-surface shadow-[var(--overlay-shadow)]"
      onclick={(e) => e.stopPropagation()}
      role="presentation"
    >
      <input
        bind:this={inputEl}
        bind:value={query}
        onkeydown={onKeydown}
        placeholder="Run a command…  (e.g. stop postgres)"
        class="w-full bg-transparent px-4 py-3.5 text-sm outline-none placeholder:text-muted"
      />
      <div class="max-h-80 overflow-auto border-t border-border">
        {#if !query.trim()}
          <div class="px-4 py-6 text-center text-sm text-muted">
            Type to search actions across containers, images, volumes &amp; networks
          </div>
        {:else if filtered.length === 0}
          <div class="px-4 py-6 text-center text-sm text-muted">No matching actions</div>
        {:else}
          {#each filtered as it, i (it.key)}
            <button
              class="flex w-full items-center gap-2 px-4 py-2.5 text-left text-sm transition-colors
                {i === selected ? 'bg-surface-2' : 'hover:bg-surface-2/50'}"
              onmouseenter={() => (selected = i)}
              onclick={() => run(it)}
            >
              {#if it.ai}
                <span class="rounded bg-accent/15 px-1.5 py-0.5 text-[10px] font-medium text-accent">AI</span>
                <span class="text-fg">Interpret</span>
              {:else}
                <span class="{it.danger ? 'text-danger' : 'text-fg'}">{it.label}</span>
              {/if}
              <span class="truncate font-mono text-xs text-muted">{it.target}</span>
            </button>
          {/each}
        {/if}
      </div>
      <div
        class="flex items-center justify-between border-t border-border px-4 py-2 text-[11px] text-muted"
      >
        <span>↑↓ navigate · ↵ run · esc close</span>
        <span>{filtered.length} actions</span>
      </div>
    </div>
  </div>
{/if}
