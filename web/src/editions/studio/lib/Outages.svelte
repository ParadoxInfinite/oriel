<script>
  import { outages, fmt, registerEscape } from '../../../platform/index.js'

  let open = $state(false)
  $effect(() => {
    if (open) return registerEscape(() => (open = false))
  })

  const meta = {
    down: { label: 'Colima down', cls: 'bad' },
    offline: { label: 'Oriel offline', cls: 'off' },
  }
  const sorted = $derived([...outages.list].sort((a, b) => b.end - a.end))
  const recent = $derived(sorted.slice(0, 3))
  const has = $derived(outages.list.length > 0)
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<!-- role + tabindex are set together only when `has` (it's a real button then);
     the static analyzer can't correlate the two ternaries. -->
<div
  class="mx-3 mb-2 rounded-lg border border-[var(--border)] bg-[var(--panel)] p-3 shadow-[var(--shadow-sm)] transition-colors {has ? 'cursor-pointer hover:border-[var(--border-strong)] hover:bg-[var(--panel-2)]' : ''}"
  role={has ? 'button' : undefined}
  tabindex={has ? 0 : undefined}
  title={has ? 'View all outages' : undefined}
  onclick={() => has && (open = true)}
  onkeydown={(e) => {
    if (has && (e.key === 'Enter' || e.key === ' ')) {
      e.preventDefault()
      open = true
    }
  }}
>
  <div class="flex w-full items-center justify-between">
    <span class="eyebrow">Recent outages</span>
    {#if has}
      <span class="flex items-center gap-1 text-[var(--text-3)]"><span class="count">{outages.list.length}</span><span class="text-[11px]">›</span></span>
    {/if}
  </div>

  {#if !has}
    <div class="mt-2 flex items-center gap-1.5 text-[11px] text-[var(--text-3)]"><span class="h-1.5 w-1.5 rounded-full bg-[var(--green)]"></span> All clear · 30 days</div>
  {:else}
    <div class="mt-2 flex flex-col gap-2">
      {#each recent as o (o.start)}
        <div>
          <div class="flex items-center gap-2">
            <span class="h-1.5 w-1.5 shrink-0 rounded-full {meta[o.kind]?.cls === 'bad' ? 'bg-[var(--red)]' : 'bg-[var(--slate-dot)]'}"></span>
            <span class="text-[11px] text-[var(--text-2)]">{meta[o.kind]?.label ?? o.kind}</span>
          </div>
          <div class="mono ml-3.5 text-[10px] text-[var(--text-3)]">{fmt.duration(o.end - o.start)} · {fmt.relativeTime(o.end / 1000)}</div>
        </div>
      {/each}
    </div>
  {/if}
</div>

{#if open}
  <div class="fixed inset-0 z-[70] flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm" role="presentation" onclick={(e) => e.target === e.currentTarget && (open = false)}>
    <div class="flex max-h-[80vh] w-full max-w-md flex-col overflow-hidden rounded-xl border border-[var(--border)] bg-[var(--panel)] shadow-[var(--shadow-lg)]">
      <div class="flex items-center justify-between border-b border-[var(--border)] px-5 py-3.5">
        <div class="flex items-center gap-2">
          <h2 class="text-[14px] font-semibold tracking-tight">Outage history</h2>
          <span class="count">{sorted.length}</span>
          <span class="rounded-full border border-[var(--border)] px-1.5 text-[10px] text-[var(--text-3)]">30-day window</span>
        </div>
        <button class="btn btn-default btn-sm" onclick={() => (open = false)}>Close</button>
      </div>

      <div class="min-h-0 flex-1 overflow-auto">
        {#each sorted as o (o.start)}
          <div class="flex items-center gap-3 border-b border-[var(--border)] px-5 py-2.5 last:border-0">
            <span class="h-2 w-2 shrink-0 rounded-full {meta[o.kind]?.cls === 'bad' ? 'bg-[var(--red)]' : 'bg-[var(--slate-dot)]'}"></span>
            <div class="min-w-0 flex-1">
              <div class="flex items-center justify-between gap-2">
                <span class="text-[13px] text-[var(--text)]">{meta[o.kind]?.label ?? o.kind}</span>
                <span class="mono tnum shrink-0 text-xs text-[var(--text-2)]">{fmt.duration(o.end - o.start)}</span>
              </div>
              <div class="mono mt-0.5 truncate text-[11px] text-[var(--text-3)]">{fmt.dateTime(o.start)} → {fmt.timeOnly(o.end)}</div>
            </div>
            <span class="shrink-0 text-[11px] text-[var(--text-3)]">{fmt.relativeTime(o.end / 1000)}</span>
          </div>
        {:else}
          <div class="px-5 py-10 text-center text-sm text-[var(--text-2)]">No outages recorded</div>
        {/each}
      </div>

      <div class="border-t border-[var(--border)] px-5 py-3 text-[11px] leading-relaxed">
        <div class="text-[var(--text-2)]">Brief flapping is merged into one entry.</div>
        <div class="text-[var(--text-3)]">Recoveries less than a minute apart count as a single outage.</div>
      </div>
    </div>
  </div>
{/if}
