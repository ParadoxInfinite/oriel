<script>
  import { bytes } from '../lib/format.js'

  // items: [{ id, primary, secondary?, size }]. onPrune receives the chosen items.
  let { title, note = '', items, onClose, onPrune } = $props()

  let selected = $state(new Set(items.map((i) => i.id)))
  let busy = $state(false)
  let done = $state(0)

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
    done = 0
    await onPrune(chosen, (n) => (done = n))
    busy = false
    onClose()
  }
  const onKey = (e) => {
    if (e.key === 'Escape' && !busy) onClose()
  }
</script>

<svelte:window onkeydown={onKey} />

<div
  class="fixed inset-0 z-50 flex items-center justify-center bg-black/55 p-4"
  role="presentation"
  onclick={(e) => {
    if (e.target === e.currentTarget && !busy) onClose()
  }}
>
  <div class="flex max-h-[82vh] w-full max-w-lg flex-col overflow-hidden rounded-[--radius] border border-border bg-surface shadow-2xl">
    <div class="flex items-center justify-between border-b border-border px-5 py-3">
      <div>
        <h2 class="display text-sm font-semibold tracking-tight">{title}</h2>
        {#if note}<p class="mt-0.5 text-[11px] text-faint">{note}</p>{/if}
      </div>
      {#if items.length}
        <button class="text-xs text-muted transition-colors hover:text-fg" onclick={toggleAll}>
          {allOn ? 'Deselect all' : 'Select all'}
        </button>
      {/if}
    </div>

    <div class="min-h-0 flex-1 overflow-auto">
      {#each items as it (it.id)}
        <label class="flex cursor-pointer items-center gap-3 border-b border-border/60 px-5 py-2.5 transition-colors hover:bg-surface-2/40">
          <input type="checkbox" checked={selected.has(it.id)} onchange={() => toggle(it.id)} class="h-3.5 w-3.5 accent-accent" />
          <div class="min-w-0 flex-1">
            <div class="truncate font-mono text-[13px] text-fg">{it.primary}</div>
            {#if it.secondary}<div class="truncate text-[11px] text-faint">{it.secondary}</div>{/if}
          </div>
          <span class="shrink-0 font-mono text-xs text-muted">{bytes(it.size)}</span>
        </label>
      {:else}
        <div class="px-5 py-12 text-center text-sm text-muted">Nothing to prune — all clean.</div>
      {/each}
    </div>

    {#if busy}
      <div class="border-t border-border px-5 pt-3">
        <div class="h-1.5 w-full overflow-hidden rounded-full bg-surface-2">
          <div class="h-full rounded-full bg-accent transition-all" style="width:{chosen.length ? (done / chosen.length) * 100 : 0}%"></div>
        </div>
        <div class="mt-1.5 font-mono text-[11px] text-faint">Removing {done} of {chosen.length}…</div>
      </div>
    {/if}

    <div class="flex items-center justify-between gap-3 border-t border-border px-5 py-3">
      <span class="text-xs text-muted">
        {chosen.length} of {items.length} · <span class="font-mono text-ok">{bytes(total)}</span> reclaimable
      </span>
      <div class="flex gap-2">
        <button
          class="rounded-md border border-border px-3 py-1.5 text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg disabled:opacity-40"
          onclick={onClose}
          disabled={busy}>Cancel</button
        >
        <button
          class="rounded-md bg-danger px-3 py-1.5 text-sm font-medium text-white transition-opacity hover:opacity-90 disabled:opacity-40"
          onclick={run}
          disabled={busy || chosen.length === 0}
        >
          {busy ? `Pruning ${done}/${chosen.length}…` : `Prune ${chosen.length}`}
        </button>
      </div>
    </div>
  </div>
</div>
