<script>
  import { fmt } from '../../../platform/index.js'

  // items: [{ id, primary, secondary?, size }]. onPrune receives the chosen items.
  let { title, note = '', items, onClose, onPrune } = $props()

  let selected = $state(new Set(items.map((i) => i.id)))
  let busy = $state(false)

  const chosen = $derived(items.filter((i) => selected.has(i.id)))
  const total = $derived(chosen.reduce((a, i) => a + (i.size || 0), 0))
  const allOn = $derived(selected.size === items.length && items.length > 0)

  function toggle(id) {
    const s = new Set(selected)
    s.has(id) ? s.delete(id) : s.add(id)
    selected = s
  }
  function toggleAll() {
    selected = allOn ? new Set() : new Set(items.map((i) => i.id))
  }
  async function run() {
    if (!chosen.length || busy) return
    busy = true
    await onPrune(chosen)
    busy = false
    onClose()
  }
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && !busy && onClose()} />

<div class="fixed inset-0 z-[70] flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm" role="presentation" onclick={(e) => e.target === e.currentTarget && !busy && onClose()}>
  <div class="flex max-h-[82vh] w-full max-w-lg flex-col overflow-hidden rounded-xl border border-[var(--border)] bg-[var(--panel)] shadow-[var(--shadow-lg)]">
    <div class="flex items-center justify-between gap-3 border-b border-[var(--border)] px-5 py-3.5">
      <div>
        <h2 class="text-[14px] font-semibold tracking-tight">{title}</h2>
        {#if note}<p class="mt-0.5 text-xs text-[var(--text-3)]">{note}</p>{/if}
      </div>
      {#if items.length}
        <button class="btn btn-ghost btn-sm shrink-0" onclick={toggleAll}>{allOn ? 'Deselect all' : 'Select all'}</button>
      {/if}
    </div>

    <div class="min-h-0 flex-1 overflow-auto">
      {#each items as it (it.id)}
        <label class="flex cursor-pointer items-center gap-3 border-b border-[var(--border)] px-5 py-2.5 transition-colors hover:bg-[var(--panel-2)]">
          <input type="checkbox" checked={selected.has(it.id)} onchange={() => toggle(it.id)} class="h-4 w-4 rounded" style="accent-color:var(--accent)" />
          <div class="min-w-0 flex-1">
            <div class="mono truncate text-[13px] text-[var(--text)]">{it.primary}</div>
            {#if it.secondary}<div class="truncate text-[11px] text-[var(--text-3)]">{it.secondary}</div>{/if}
          </div>
          <span class="mono shrink-0 text-xs text-[var(--text-2)]">{fmt.bytes(it.size)}</span>
        </label>
      {:else}
        <div class="px-5 py-12 text-center text-sm text-[var(--text-2)]">Nothing to prune — all clean.</div>
      {/each}
    </div>

    <div class="flex items-center justify-between gap-3 border-t border-[var(--border)] px-5 py-3">
      <span class="text-xs text-[var(--text-2)]">{chosen.length} of {items.length} · <span class="mono font-medium text-[var(--green)]">{fmt.bytes(total)}</span> reclaimable</span>
      <div class="flex gap-2">
        <button class="btn btn-default btn-sm" onclick={onClose} disabled={busy}>Cancel</button>
        <button class="btn btn-sm" style="background:var(--red);color:#fff" onclick={run} disabled={busy || chosen.length === 0}>
          {busy ? 'Pruning…' : `Prune ${chosen.length}`}
        </button>
      </div>
    </div>
  </div>
</div>
