<script>
  import { palette, closePalette } from '../lib/palette.svelte.js'
  import { containers, refreshContainers } from '../lib/containers.svelte.js'
  import { fuzzyScore } from '../lib/fuzzy.js'
  import { invoke } from '../lib/invoke.js'
  import { confirm } from '../lib/confirm.svelte.js'
  import { provider, resolveText } from '../lib/provider.svelte.js'
  import { toast } from '../lib/toast.svelte.js'

  let query = $state('')
  let selected = $state(0)
  let inputEl = $state(null)

  // Per-container actions, gated by state so the palette only offers what's
  // valid. Each maps to a {tool, args} call — the same shape a provider emits.
  function actionsFor(c) {
    const running = c.state === 'running'
    const acts = []
    if (!running) acts.push({ verb: 'Start', tool: 'container.start', args: { id: c.id } })
    if (running) acts.push({ verb: 'Stop', tool: 'container.stop', args: { id: c.id } })
    if (running) acts.push({ verb: 'Restart', tool: 'container.restart', args: { id: c.id } })
    acts.push({
      verb: 'Remove',
      tool: 'container.remove',
      args: { id: c.id, force: running },
      danger: true,
    })
    return acts.map((a) => ({
      ...a,
      label: `${a.verb} container`,
      target: c.name,
      key: `${a.tool}:${c.id}`,
    }))
  }

  const items = $derived(containers.list.flatMap(actionsFor))

  const filtered = $derived.by(() => {
    const scored = items
      .map((it) => ({ it, score: fuzzyScore(query, `${it.verb} ${it.target}`) }))
      .filter((x) => x.score >= 0)
      .sort((a, b) => b.score - a.score || a.it.target.localeCompare(b.it.target))
      .slice(0, 50)
      .map((x) => x.it)

    // With a provider configured, offer free-text interpretation of the query
    // as the first option — the same execution path, just an NL resolver.
    if (provider.enabled && query.trim()) {
      return [{ ai: true, key: '__ai__', label: 'Interpret', target: query.trim() }, ...scored]
    }
    return scored
  })

  $effect(() => {
    if (palette.open) {
      query = ''
      selected = 0
      refreshContainers()
      queueMicrotask(() => inputEl?.focus())
    }
  })

  $effect(() => {
    // Keep selection in range as the filtered list changes.
    if (selected >= filtered.length) selected = Math.max(0, filtered.length - 1)
  })

  async function run(it) {
    closePalette()
    if (it.ai) {
      try {
        const res = await resolveText(it.target)
        toast(`${res.call.tool} · ${res.call.args.id ?? ''}`.trim(), 'ok')
      } catch (e) {
        toast(e.message, 'error')
      }
      refreshContainers()
      return
    }
    if (it.danger) {
      const ok = await confirm({
        title: `${it.verb} container?`,
        message: `“${it.target}” will be permanently removed.`,
        confirmLabel: it.verb,
      })
      if (!ok) return
    }
    await invoke(it.tool, it.args, { success: `${it.verb} · ${it.target}` })
    refreshContainers()
  }

  function onKeydown(e) {
    if (e.key === 'Escape') return closePalette()
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      selected = Math.min(selected + 1, filtered.length - 1)
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
    onkeydown={null}
    role="presentation"
  >
    <div
      class="w-full max-w-lg overflow-hidden rounded-[--radius] border border-border bg-surface shadow-2xl"
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
        {#if filtered.length === 0}
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
