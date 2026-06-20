<script>
  // Clickable sort toggle for a table header. Render inside a <th>; pass a `col`
  // with { label, key } and the shared `sort` state from createSort().
  import { toggleSort } from '../../../platform/index.js'
  let { col, sort } = $props()
  const active = $derived(sort && col.key && sort.key === col.key)
</script>

{#if col.key && sort}
  <button type="button" class="th-sort {active ? 'active' : ''}" onclick={() => toggleSort(sort, col.key)}>
    <span>{col.label}</span>
    <span class="th-arrow">{active ? (sort.dir === 'asc' ? '↑' : '↓') : '↕'}</span>
  </button>
{:else}
  {col.label}
{/if}
