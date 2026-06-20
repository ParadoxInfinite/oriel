<script>
  // Generic table shell shared by the resource views: header columns, loading,
  // error, and empty states. Rows are provided via the children snippet. Pass a
  // `sort` state (from createSort) to make columns with a `key` clickable.
  import SortHeader from './SortHeader.svelte'
  let { columns, store, empty = 'Nothing here', sort = null, children } = $props()
</script>

{#if store.error}
  <div class="rounded-[--radius] border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">
    {store.error}
  </div>
{:else if store.list.length === 0}
  <div class="rounded-[--radius] border border-dashed border-border py-20 text-center text-sm text-muted">
    {empty}
  </div>
{:else}
  <div class="overflow-hidden rounded-[--radius] border border-border">
    <table class="w-full text-sm">
      <thead class="bg-surface text-xs uppercase tracking-wide text-muted">
        <tr>
          {#each columns as col}
            <th class="px-4 py-2.5 font-medium {col.right ? 'text-right' : 'text-left'}">
              <SortHeader {col} {sort} />
            </th>
          {/each}
        </tr>
      </thead>
      <tbody>
        {@render children()}
      </tbody>
    </table>
  </div>
{/if}
