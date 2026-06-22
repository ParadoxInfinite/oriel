<script>
  import { PullController, fmtStars } from '../lib/pull.svelte.js'
  import { registerEscape } from '../lib/modalStack.svelte.js'
  import { btnPrimary } from '../lib/ui.js'

  let { onClose, initial = '' } = $props()

  const field = 'w-full rounded-[--radius] border border-border bg-bg px-3 py-1.5 text-sm outline-none placeholder:text-muted focus:border-accent/50'

  const pc = new PullController(initial)
  let inputEl = $state(null)
  $effect(() => {
    inputEl?.focus()
  })
  $effect(() => () => pc.destroy())
  $effect(() => registerEscape(() => !pc.pulling && onClose()))
</script>

<div class="fixed inset-0 z-50 flex items-start justify-center bg-black/60 p-4 pt-[10vh]" role="presentation" onclick={(e) => e.target === e.currentTarget && !pc.pulling && onClose()}>
  <div class="w-full max-w-md overflow-hidden rounded-[--radius] border border-border bg-surface shadow-2xl">
    <div class="flex items-center gap-2 border-b border-border px-5 py-3">
      <h3 class="display text-sm font-semibold tracking-tight">Pull image</h3>
    </div>

    <div class="p-5">
      <div class="flex items-center gap-3">
        <span class="text-xs font-medium text-muted">Registry</span>
        <select bind:value={pc.sourceId} onchange={pc.onSourceChange} disabled={pc.pulling} class="{field} ml-auto w-auto cursor-pointer">
          {#each pc.sources as s}<option value={s.id}>{s.label}{s.search ? '' : ' (no search)'}</option>{/each}
        </select>
      </div>

      <div class="mt-3 flex gap-2">
        <input bind:this={inputEl} bind:value={pc.ref} oninput={pc.onInput} onkeydown={pc.onKeydown} placeholder={pc.source.hint} disabled={pc.pulling} autocomplete="off" spellcheck="false" class={field} />
        <button class={btnPrimary} disabled={pc.pulling || !pc.ref.trim()} onclick={pc.pull}>{pc.pulling ? 'Pulling…' : 'Pull'}</button>
      </div>

      {#if pc.view === 'search'}
        <div class="mt-2 max-h-72 overflow-auto rounded-[--radius] border border-border">
          {#if pc.busy && !pc.results.length}
            <div class="px-3 py-3 font-mono text-xs text-faint">Searching {pc.source.label}…</div>
          {:else if !pc.results.length}
            <div class="px-3 py-3 text-xs text-faint">No matching images.</div>
          {:else}
            {#each pc.results as r, i (r.name)}
              <button class="flex w-full items-start gap-2.5 border-b border-border/60 px-3 py-2 text-left transition-colors last:border-0 {i === pc.highlight ? 'bg-surface-2' : 'hover:bg-surface-2/50'}" onmouseenter={() => (pc.highlight = i)} onclick={() => pc.pickRepo(r)}>
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-1.5">
                    <span class="truncate font-mono text-[13px] text-fg">{r.name}</span>
                    {#if r.official}<span class="shrink-0 rounded bg-accent/15 px-1.5 text-[10px] font-medium text-accent">official</span>{/if}
                  </div>
                  {#if r.description}<div class="truncate text-[11px] text-faint">{r.description}</div>{/if}
                </div>
                {#if r.stars > 0}<span class="shrink-0 font-mono text-[11px] text-faint">★ {fmtStars(r.stars)}</span>{/if}
              </button>
            {/each}
          {/if}
        </div>
      {:else if pc.view === 'tags'}
        <div class="mt-2 rounded-[--radius] border border-border p-2.5">
          <div class="mb-1.5 px-0.5 text-[11px] text-faint">Recent tags · {pc.repoBase.split('/').pop()}</div>
          {#if !pc.busy && !pc.shownTags.length}
            <div class="px-0.5 py-1 text-xs text-faint">No tags found — type one manually.</div>
          {:else}
            <div class="flex max-h-40 flex-wrap gap-1.5 overflow-auto">
              {#each pc.shownTags.slice(0, 40) as t, i (t)}
                <button class="rounded-md border px-2 py-1 font-mono text-[11px] transition-colors {i === pc.highlight ? 'border-accent/60 bg-accent/10 text-accent' : 'border-border text-muted hover:bg-surface-2'}" onmouseenter={() => (pc.highlight = i)} onclick={() => pc.pickTag(t)}>{t}</button>
              {/each}
            </div>
          {/if}
        </div>
      {:else if !pc.source.search}
        <p class="mt-2 text-[11px] text-faint">{pc.source.label.split('·')[0].trim()} has no public search — type the full image name above. Pulls work the same.</p>
      {/if}

      {#if pc.error}
        <div class="mt-3 break-words rounded-[--radius] border border-danger/30 bg-danger/10 px-3 py-2 font-mono text-xs text-danger">{pc.error}</div>
      {:else if pc.status}
        <div class="mt-3 truncate font-mono text-xs text-muted">{pc.status}</div>
      {/if}
    </div>

    <div class="flex justify-end border-t border-border px-5 py-3">
      <button class="rounded-[--radius] px-3 py-1.5 text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg disabled:opacity-40" disabled={pc.pulling} onclick={onClose}>{pc.done || !pc.status ? 'Close' : 'Working…'}</button>
    </div>
  </div>
</div>
