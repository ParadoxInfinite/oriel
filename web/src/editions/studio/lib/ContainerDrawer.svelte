<script>
  import { sse, apiGet, fmt } from '../../../platform/index.js'
  import Icon from './Icon.svelte'
  import StatusPill from './StatusPill.svelte'

  let { container, onClose } = $props()

  let tab = $state('logs') // 'logs' | 'inspect'

  // ── Logs (streaming) ───────────────────────────────────────────────────────
  let lines = $state([])
  let paused = $state(false)
  let scroller = $state(null)
  const MAX = 2000
  let es
  $effect(() => {
    lines = []
    es = sse(`/api/containers/${container.id}/logs`, ['log', 'error'], (name, data) => {
      if (name === 'error') {
        lines = [...lines, { stream: 'error', line: data.error || 'stream error' }]
        return
      }
      const next = [...lines, data]
      lines = next.length > MAX ? next.slice(next.length - MAX) : next
    })
    return () => es?.close()
  })
  $effect(() => {
    void lines.length
    if (tab === 'logs' && !paused && scroller) queueMicrotask(() => (scroller.scrollTop = scroller.scrollHeight))
  })
  function onScroll() {
    if (!scroller) return
    paused = scroller.scrollHeight - scroller.scrollTop - scroller.clientHeight >= 40
  }
  const lineColor = (s) => (s === 'stderr' ? 'text-[var(--amber)]' : s === 'error' ? 'text-[var(--red)]' : 'text-[var(--text)]')

  // ── Inspect (fetched on first open) ────────────────────────────────────────
  let detail = $state(null)
  let inspectErr = $state('')
  let loaded = false
  $effect(() => {
    if (tab === 'inspect' && !loaded) {
      loaded = true
      apiGet(`/api/containers/${container.id}/inspect`)
        .then((d) => (detail = d))
        .catch((e) => (inspectErr = e.message))
    }
  })

  const rows = $derived(
    detail
      ? [
          ['Image', detail.image],
          ['Image ID', (detail.imageId || '').replace('sha256:', '').slice(0, 12)],
          ['Command', detail.command],
          ['Working dir', detail.workingDir || '—'],
          ['Restart policy', detail.restartPolicy || 'no'],
          ['Started', detail.startedAt && detail.running ? new Date(detail.startedAt).toLocaleString() : '—'],
          ['Exit code', detail.running ? '—' : String(detail.exitCode)],
          ['Health', detail.health || '—'],
        ]
      : []
  )
</script>

<div class="fixed inset-0 z-[60] flex justify-end bg-black/40 backdrop-blur-[1px]" role="presentation" onclick={(e) => e.target === e.currentTarget && onClose()}>
  <div class="flex h-full w-[760px] max-w-[95vw] flex-col border-l border-[var(--border)] bg-[var(--bg)] shadow-[var(--shadow-lg)]" role="presentation" onclick={(e) => e.stopPropagation()}>
    <div class="flex shrink-0 items-center justify-between gap-3 border-b border-[var(--border)] bg-[var(--panel)] px-5 py-3">
      <div class="flex min-w-0 items-center gap-2.5">
        <StatusPill state={container.state} />
        <div class="min-w-0">
          <div class="truncate text-[13px] font-semibold">{container.name}</div>
          <div class="mono truncate text-[11px] text-[var(--text-3)]">{container.image}</div>
        </div>
      </div>
      <button class="btn btn-default btn-sm" onclick={onClose}>Close</button>
    </div>

    <div class="flex shrink-0 gap-1 border-b border-[var(--border)] bg-[var(--panel)] px-3">
      {#each [['logs', 'Logs'], ['inspect', 'Inspect']] as [id, label]}
        <button class="relative px-3 py-2 text-[13px] font-medium transition-colors {tab === id ? 'text-[var(--accent)]' : 'text-[var(--text-2)] hover:text-[var(--text)]'}" onclick={() => (tab = id)}>
          {label}
          {#if tab === id}<span class="absolute inset-x-2 -bottom-px h-0.5 rounded-full bg-[var(--accent)]"></span>{/if}
        </button>
      {/each}
    </div>

    {#if tab === 'logs'}
      <div bind:this={scroller} onscroll={onScroll} class="mono min-h-0 flex-1 overflow-auto bg-[var(--panel-2)] text-[12px] leading-relaxed">
        {#each lines as l, i}
          <div class="flex gap-3 px-3 hover:bg-[var(--hover)]">
            <span class="tnum w-10 shrink-0 select-none border-r border-[var(--border)] py-px pr-2 text-right text-[var(--text-3)]">{i + 1}</span>
            <span class="flex-1 whitespace-pre-wrap break-words py-px {lineColor(l.stream)}">{l.line}</span>
          </div>
        {/each}
        {#if !lines.length}<div class="px-3 py-3 text-[var(--text-3)]">Waiting for logs…</div>{/if}
      </div>
      {#if paused}
        <button class="shrink-0 border-t border-[var(--border)] bg-[var(--panel-2)] py-1.5 text-center text-xs text-[var(--accent)]" onclick={() => { paused = false; scroller.scrollTop = scroller.scrollHeight }}>↓ Jump to latest</button>
      {/if}
    {:else}
      <div class="min-h-0 flex-1 overflow-auto p-5">
        {#if inspectErr}
          <div class="card border-[color-mix(in_srgb,var(--red)_40%,var(--border))] p-4 text-sm text-[var(--red)]">{inspectErr}</div>
        {:else if !detail}
          <div class="flex items-center gap-2 text-sm text-[var(--text-3)]"><span class="h-4 w-4 animate-spin rounded-full border-2 border-[var(--border-strong)] border-t-[var(--accent)]"></span> Loading…</div>
        {:else}
          <div class="card overflow-hidden">
            {#each rows as [k, v], i}
              <div class="flex gap-4 px-4 py-2.5 {i ? 'border-t border-[var(--border)]' : ''}">
                <span class="w-28 shrink-0 text-xs text-[var(--text-3)]">{k}</span>
                <span class="mono min-w-0 flex-1 break-words text-[12.5px] text-[var(--text)]">{v}</span>
              </div>
            {/each}
          </div>

          {#if detail.networks?.length}
            <div class="mt-4"><span class="eyebrow">Networks</span>
              <div class="mt-2 card overflow-hidden">
                {#each detail.networks as n, i}
                  <div class="flex items-center justify-between gap-3 px-4 py-2.5 {i ? 'border-t border-[var(--border)]' : ''}">
                    <span class="text-[13px]">{n.name}</span><span class="mono text-[12px] text-[var(--text-2)]">{n.ipAddress || '—'}</span>
                  </div>
                {/each}
              </div>
            </div>
          {/if}

          {#if detail.mounts?.length}
            <div class="mt-4"><span class="eyebrow">Mounts</span>
              <div class="mt-2 card overflow-hidden">
                {#each detail.mounts as m, i}
                  <div class="px-4 py-2.5 {i ? 'border-t border-[var(--border)]' : ''}">
                    <div class="mono flex items-center gap-2 text-[12px]"><span class="rounded bg-[var(--chip-bg)] px-1.5 text-[10px] text-[var(--text-2)]">{m.type}</span><span class="truncate text-[var(--text)]">{m.destination}</span><span class="ml-auto shrink-0 text-[10px] text-[var(--text-3)]">{m.rw ? 'rw' : 'ro'}</span></div>
                    <div class="mono mt-0.5 truncate text-[11px] text-[var(--text-3)]">{m.name || m.source}</div>
                  </div>
                {/each}
              </div>
            </div>
          {/if}

          {#if detail.env?.length}
            <div class="mt-4"><span class="eyebrow">Environment · {detail.env.length}</span>
              <div class="mono mt-2 max-h-56 overflow-auto rounded-lg border border-[var(--border)] bg-[var(--panel-2)] p-3 text-[11.5px] leading-relaxed">
                {#each detail.env as e}<div class="break-all text-[var(--text-2)]">{e}</div>{/each}
              </div>
            </div>
          {/if}
        {/if}
      </div>
    {/if}
  </div>
</div>
