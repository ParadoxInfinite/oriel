<script>
  import { op, dismissOp } from '../lib/op.svelte.js'
</script>

{#if op.title}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-6">
    <div class="flex max-h-[70vh] w-full max-w-xl flex-col overflow-hidden rounded-[--radius] border border-border bg-surface shadow-2xl">
      <div class="flex items-center gap-3 border-b border-border px-5 py-3.5">
        {#if op.active}
          <span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-accent border-t-transparent"></span>
        {:else if op.error}
          <span class="h-2.5 w-2.5 rounded-full bg-danger"></span>
        {:else}
          <span class="h-2.5 w-2.5 rounded-full bg-ok"></span>
        {/if}
        <span class="text-sm font-medium">{op.title}</span>
      </div>

      <pre class="flex-1 overflow-auto px-5 py-3 font-mono text-xs leading-relaxed text-muted">{#each op.lines as line}{line}
{/each}{#if op.error}
{op.error}{/if}</pre>

      <div class="flex justify-end border-t border-border px-5 py-3">
        <button
          class="rounded-[--radius] px-3 py-1.5 text-sm transition-colors disabled:opacity-40
            {op.error ? 'bg-danger/15 text-danger hover:bg-danger/25' : 'bg-surface-2 text-fg hover:bg-border'}"
          disabled={op.active}
          onclick={dismissOp}
        >
          {op.active ? 'Working…' : 'Close'}
        </button>
      </div>
    </div>
  </div>
{/if}
