<script>
  import { outages } from '../lib/outages.svelte.js'
  import { duration, relativeTime, dateTime, timeOnly } from '../lib/format.js'
  import { registerEscape } from '../lib/modalStack.svelte.js'

  let open = $state(false)

  const meta = {
    down: { label: 'Colima down', dot: 'bg-danger' },
    offline: { label: 'Oriel offline', dot: 'bg-faint' },
  }
  const sorted = $derived([...outages.list].sort((a, b) => b.end - a.end))
  const recent = $derived(sorted.slice(0, 4))
  const has = $derived(outages.list.length > 0)

  $effect(() => {
    if (open) return registerEscape(() => (open = false))
  })
  const openIfAny = () => {
    if (has) open = true
  }
</script>

<!-- the whole section is the click target, not just the heading -->
<div
  class="border-t border-border px-3 py-3 transition-colors {has ? 'cursor-pointer hover:bg-surface-2/30' : ''}"
  role={has ? 'button' : undefined}
  tabindex={has ? 0 : undefined}
  title={has ? 'View all outages' : undefined}
  onclick={openIfAny}
  onkeydown={(e) => {
    if (has && (e.key === 'Enter' || e.key === ' ')) {
      e.preventDefault()
      openIfAny()
    }
  }}
>
  <div class="mb-2 flex items-center justify-between px-1">
    <span class="text-[10px] font-medium uppercase tracking-[0.15em] text-faint">Recent outages</span>
    {#if has}
      <span class="flex items-center gap-1 text-faint">
        <span class="rounded-full bg-surface-2 px-1.5 text-[9px] text-muted">{outages.list.length}</span>
        <span class="text-[11px] leading-none">›</span>
      </span>
    {/if}
  </div>

  {#if !has}
    <div class="flex items-center gap-1.5 px-1 text-[11px] text-faint">
      <span class="h-1.5 w-1.5 rounded-full bg-ok"></span> All clear · 30 days
    </div>
  {:else}
    <div class="flex flex-col gap-2">
      {#each recent as o (o.start)}
        <div class="px-1">
          <div class="flex items-center gap-2">
            <span class="h-1.5 w-1.5 shrink-0 rounded-full {meta[o.kind]?.dot ?? 'bg-faint'}"></span>
            <span class="text-[11px] text-muted">{meta[o.kind]?.label ?? o.kind}</span>
          </div>
          <div class="ml-3.5 font-mono text-[10px] text-faint">{duration(o.end - o.start)} · {relativeTime(o.end / 1000)}</div>
        </div>
      {/each}
    </div>
  {/if}
</div>

{#if open}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
    role="presentation"
    onclick={(e) => {
      if (e.target === e.currentTarget) open = false
    }}
  >
    <div class="flex max-h-[80vh] w-full max-w-md flex-col overflow-hidden rounded-[--radius] border border-border bg-surface shadow-2xl">
      <div class="flex items-center justify-between border-b border-border px-5 py-3">
        <div class="flex items-center gap-2">
          <h2 class="display text-sm font-semibold tracking-tight">Outage history</h2>
          <span class="rounded-full bg-surface-2 px-1.5 py-0.5 text-[10px] text-muted">{sorted.length}</span>
          <span class="rounded-full border border-border px-1.5 py-0.5 text-[10px] text-faint">30-day window</span>
        </div>
        <button
          class="rounded-[--radius] px-2 py-1 text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg"
          onclick={() => (open = false)}>Close</button
        >
      </div>

      <div class="min-h-0 flex-1 overflow-auto">
        {#each sorted as o (o.start)}
          <div class="flex items-center gap-3 border-b border-border/60 px-5 py-2.5">
            <span class="h-2 w-2 shrink-0 rounded-full {meta[o.kind]?.dot ?? 'bg-faint'}"></span>
            <div class="min-w-0 flex-1">
              <div class="flex items-center justify-between gap-2">
                <span class="text-sm text-fg">{meta[o.kind]?.label ?? o.kind}</span>
                <span class="tnum shrink-0 font-mono text-xs text-muted">{duration(o.end - o.start)}</span>
              </div>
              <div class="mt-0.5 truncate font-mono text-[11px] text-faint">
                {dateTime(o.start)} → {timeOnly(o.end)}
              </div>
            </div>
            <span class="shrink-0 text-[11px] text-faint">{relativeTime(o.end / 1000)}</span>
          </div>
        {:else}
          <div class="px-5 py-10 text-center text-sm text-muted">No outages recorded</div>
        {/each}
      </div>

      <div class="border-t border-border px-5 py-3 text-[11px] leading-relaxed">
        <div class="text-muted">Brief flapping is merged into one entry.</div>
        <div class="text-faint">Recoveries less than a minute apart count as a single outage.</div>
      </div>
    </div>
  </div>
{/if}
