<script>
  import { ops, dismissOp, cancelOp, minimizeOp } from '../lib/op.svelte.js'
  import { registerEscape } from '../lib/modalStack.svelte.js'

  // The modal shows the focused operation; the rest live in the sidebar tray.
  const cur = $derived(ops.list.find((o) => o.id === ops.focused) ?? null)

  // Escape / backdrop click close the modal naturally: dismiss a finished op, hide
  // a running one (it keeps going in the tray).
  function close() {
    if (!cur) return
    if (cur.done) dismissOp(cur.id)
    else minimizeOp()
  }
  $effect(() => {
    if (cur) return registerEscape(close)
  })
  const pct = $derived(cur && cur.total > 0 ? Math.round((cur.cur / cur.total) * 100) : 0)
</script>

{#if cur}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-6" role="presentation" onclick={(e) => e.target === e.currentTarget && close()}>
    <div class="flex max-h-[70vh] w-full max-w-xl flex-col overflow-hidden rounded-[var(--overlay-radius)] border border-border bg-surface shadow-[var(--overlay-shadow)]">
      <div class="flex items-center gap-3 border-b border-border px-5 py-3.5">
        {#if !cur.done}
          <span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-accent border-t-transparent"></span>
        {:else if cur.error}
          <span class="h-2.5 w-2.5 rounded-full bg-danger"></span>
        {:else}
          <span class="h-2.5 w-2.5 rounded-full bg-ok"></span>
        {/if}
        <span class="text-sm font-medium">{cur.title}</span>
      </div>

      {#if cur.total > 0 && !cur.done}
        <div class="border-b border-border px-5 py-3.5">
          <div class="h-2 w-full overflow-hidden rounded-full bg-surface-2">
            <div class="h-full rounded-full bg-accent transition-[width] duration-200" style="width:{pct}%"></div>
          </div>
          <div class="mt-2 flex items-center justify-between font-mono text-[11px] text-faint">
            <span>{cur.cur.toLocaleString()} / {cur.total.toLocaleString()}</span>
            <span>{pct}%</span>
          </div>
        </div>
      {/if}

      {#if cur.lines.length || cur.error}
        <pre class="flex-1 overflow-auto px-5 py-3 font-mono text-xs leading-relaxed text-muted">{#each cur.lines as line}{line}
{/each}{#if cur.error}
{cur.error}{/if}</pre>
      {/if}

      <div class="flex items-center justify-between gap-3 border-t border-border bg-surface-2/40 px-5 py-3">
        {#if cur.jobId && !cur.done}
          <span class="text-[11px] text-faint">Runs in the background — safe to close.</span>
        {:else}
          <span></span>
        {/if}
        <div class="flex gap-2">
          {#if cur.jobId && !cur.done}
            <button
              class="rounded-lg bg-danger/15 px-3.5 py-1.5 text-[13px] font-medium text-danger transition-colors hover:bg-danger/25 disabled:opacity-40"
              disabled={cur.cancelling}
              onclick={() => cancelOp(cur.id)}
            >
              {cur.cancelling ? 'Cancelling…' : 'Cancel operation'}
            </button>
          {/if}
          {#if cur.done}
            <button
              class="rounded-lg px-3.5 py-1.5 text-[13px] font-medium transition-colors
                {cur.error
                  ? 'bg-danger/15 text-danger hover:bg-danger/25'
                  : 'border border-border bg-surface text-muted hover:bg-surface-2 hover:text-fg'}"
              onclick={() => dismissOp(cur.id)}
            >
              Close
            </button>
          {:else}
            <button class="rounded-lg border border-border bg-surface px-3.5 py-1.5 text-[13px] font-medium text-muted transition-colors hover:bg-surface-2 hover:text-fg" onclick={minimizeOp}>
              Hide
            </button>
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}
