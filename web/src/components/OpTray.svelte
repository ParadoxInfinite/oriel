<script>
  import { ops, focusOp, dismissOp } from '../lib/op.svelte.js'

  // Operations not currently in the modal (ops.focused) surface here, so hidden or
  // backgrounded jobs stay visible and clickable. Shared global overlay: one
  // component, re-themed per edition via the overlayVars wrapper in App.svelte.
  const tray = $derived(ops.list.filter((o) => o.id !== ops.focused))
  const lastLine = (o) => o.lines[o.lines.length - 1] || ''
</script>

{#if tray.length}
  <div class="fixed bottom-4 left-4 z-40 flex w-72 flex-col gap-1.5">
    {#each tray as o (o.id)}
      <div class="group flex items-center gap-2.5 rounded-[--radius] border border-border bg-surface px-3 py-2.5 shadow-lg">
        <button class="flex min-w-0 flex-1 items-center gap-2.5 text-left" onclick={() => focusOp(o.id)} title="Show details">
          {#if !o.done}
            <span class="h-3.5 w-3.5 shrink-0 animate-spin rounded-full border-2 border-accent border-t-transparent"></span>
          {:else if o.error}
            <span class="h-2.5 w-2.5 shrink-0 rounded-full bg-danger"></span>
          {:else}
            <span class="h-2.5 w-2.5 shrink-0 rounded-full bg-ok"></span>
          {/if}
          <span class="min-w-0 flex-1">
            <span class="block truncate text-[13px] font-medium text-fg">{o.title}</span>
            <span class="block truncate font-mono text-[11px] text-faint">{o.error || lastLine(o) || (o.done ? 'Done' : 'Working…')}</span>
          </span>
        </button>
        {#if o.done}
          <button
            class="shrink-0 rounded p-0.5 text-faint opacity-0 transition hover:text-danger group-hover:opacity-100"
            title="Dismiss"
            aria-label="Dismiss operation"
            onclick={() => dismissOp(o.id)}
          >
            <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M18 6 6 18M6 6l12 12" /></svg>
          </button>
        {/if}
      </div>
    {/each}
  </div>
{/if}
