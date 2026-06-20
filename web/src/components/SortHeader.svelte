<script>
  // Header cell content: a clickable sort toggle when `col.key` is set, else
  // plain label text. Wrap it in a <th> at the call site.
  import { toggleSort } from '../lib/sort.svelte.js'
  let { col, sort } = $props()
  const active = $derived(sort && sort.key === col.key)
</script>

{#if col.key && sort}
  <button
    type="button"
    class="group inline-flex items-center gap-1 uppercase tracking-wide transition-colors hover:text-fg {active ? 'text-fg' : ''}"
    onclick={() => toggleSort(sort, col.key)}
  >
    <span>{col.label}</span>
    <span class="text-[9px] leading-none transition-opacity {active ? 'text-accent' : 'text-faint opacity-0 group-hover:opacity-100'}">
      {active ? (sort.dir === 'asc' ? '▲' : '▼') : '▲'}
    </span>
  </button>
{:else}
  {col.label}
{/if}
