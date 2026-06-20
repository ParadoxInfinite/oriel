<script>
  import { ops, focusOp, dismissOp } from '../lib/op.svelte.js'

  // Operations not currently in the modal (ops.focused) surface here, so hidden or
  // backgrounded jobs stay visible and clickable. One shared component for both
  // editions: it styles with Tailwind semantic classes over the --color-* tokens,
  // which each edition root maps to its own palette (see studio.css bridge).
  const tray = $derived(ops.list.filter((o) => o.id !== ops.focused))
  const lastLine = (o) => o.lines[o.lines.length - 1] || ''
</script>

{#if tray.length}
  <div class="mx-3 mb-2 flex flex-col gap-1 rounded-lg border border-border bg-surface p-2">
    <div class="px-1 pb-0.5 pt-0.5 text-[10px] font-medium uppercase tracking-[0.15em] text-faint">Operations</div>
    {#each tray as o (o.id)}
      <div class="group flex items-center gap-2 rounded-md px-1.5 py-1.5 transition-colors hover:bg-surface-2/60">
        <button class="flex min-w-0 flex-1 items-center gap-2 text-left" onclick={() => focusOp(o.id)} title="Show details">
          {#if !o.done}
            <span class="h-3 w-3 shrink-0 animate-spin rounded-full border-2 border-accent border-t-transparent"></span>
          {:else if o.error}
            <span class="h-2 w-2 shrink-0 rounded-full bg-danger"></span>
          {:else}
            <span class="h-2 w-2 shrink-0 rounded-full bg-ok"></span>
          {/if}
          <span class="min-w-0 flex-1">
            <span class="block truncate text-[12.5px] font-medium text-fg">{o.title}</span>
            <span class="block truncate font-mono text-[10px] text-faint">{o.error || lastLine(o) || (o.done ? 'Done' : 'Working…')}</span>
          </span>
        </button>
        {#if o.done}
          <button
            class="shrink-0 rounded p-0.5 text-faint opacity-0 transition hover:text-danger group-hover:opacity-100"
            title="Dismiss"
            aria-label="Dismiss operation"
            onclick={() => dismissOp(o.id)}
          >
            <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M18 6 6 18M6 6l12 12" /></svg>
          </button>
        {/if}
      </div>
    {/each}
  </div>
{/if}
