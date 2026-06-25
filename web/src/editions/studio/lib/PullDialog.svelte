<script>
  import { PullController, fmtStars, registerEscape } from '../../../platform/index.js'
  import Icon from './Icon.svelte'

  let { onClose, initial = '' } = $props()

  const pc = new PullController(initial)
  let inputEl = $state(null)
  $effect(() => {
    inputEl?.focus()
  })
  $effect(() => () => pc.destroy())
  $effect(() => registerEscape(() => !pc.pulling && onClose()))
</script>

<div class="fixed inset-0 z-[70] flex items-start justify-center bg-black/45 p-4 pt-[10vh] backdrop-blur-sm" role="presentation" onclick={(e) => e.target === e.currentTarget && !pc.pulling && onClose()}>
  <div class="w-full max-w-md overflow-hidden rounded-xl border border-[var(--border)] bg-[var(--panel)] shadow-[var(--shadow-lg)]">
    <div class="flex items-center gap-2.5 border-b border-[var(--border)] px-5 py-3.5">
      <Icon name="download" size={16} class="text-[var(--text-3)]" />
      <h2 class="text-[14px] font-semibold tracking-tight">Pull image</h2>
    </div>

    <div class="p-5">
      <div class="flex items-center gap-3">
        <span class="text-[12px] font-medium text-[var(--text-2)]">Registry</span>
        <select bind:value={pc.sourceId} onchange={pc.onSourceChange} disabled={pc.pulling} class="input ml-auto cursor-pointer py-1.5">
          {#each pc.sources as s}<option value={s.id}>{s.label}{s.search ? '' : ' (no search)'}</option>{/each}
        </select>
      </div>

      <div class="mt-3 flex gap-2">
        <div class="relative min-w-0 flex-1">
          <Icon name="harddrive" size={14} class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-3)]" />
          <input bind:this={inputEl} bind:value={pc.ref} oninput={pc.onInput} onkeydown={pc.onKeydown} placeholder={pc.source.hint} disabled={pc.pulling} autocomplete="off" spellcheck="false" class="input has-icon w-full" />
        </div>
        <button class="btn btn-primary" disabled={pc.pulling || !pc.ref.trim()} onclick={pc.pull}>{pc.pulling ? 'Pulling…' : 'Pull'}</button>
      </div>

      {#if pc.view === 'search'}
        <div class="mt-2 max-h-72 overflow-auto rounded-lg border border-[var(--border)]">
          {#if pc.busy && !pc.results.length}
            <div class="flex items-center gap-2 px-3 py-3 text-[12px] text-[var(--text-3)]"><span class="h-3 w-3 animate-spin rounded-full border-2 border-[var(--border-strong)] border-t-[var(--accent)]"></span>Searching {pc.source.label}…</div>
          {:else if !pc.results.length}
            <div class="px-3 py-3 text-[12px] text-[var(--text-3)]">No matching images.</div>
          {:else}
            {#each pc.results as r, i (r.name)}
              <button class="flex w-full items-start gap-2.5 border-b border-[var(--border)] px-3 py-2 text-left transition-colors last:border-0 {i === pc.highlight ? 'bg-[var(--accent-tint)]' : 'hover:bg-[var(--panel-2)]'}" onmouseenter={() => (pc.highlight = i)} onclick={() => pc.pickRepo(r)}>
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-1.5">
                    <span class="mono truncate text-[12.5px] font-medium text-[var(--text)]">{r.name}</span>
                    {#if r.official}<span class="shrink-0 rounded bg-[var(--accent-tint)] px-1.5 text-[10px] font-medium text-[var(--accent)]">official</span>{/if}
                  </div>
                  {#if r.description}<div class="truncate text-[11.5px] text-[var(--text-3)]">{r.description}</div>{/if}
                </div>
                {#if r.stars > 0}<span class="mono mt-0.5 flex shrink-0 items-center gap-1 text-[11px] text-[var(--text-3)]"><svg viewBox="0 0 24 24" width="11" height="11" fill="currentColor" stroke="none"><path d="M12 2l2.9 6.3 6.9.7-5.1 4.7 1.4 6.8L12 17.8 5.9 20.5l1.4-6.8L2.2 9l6.9-.7z" /></svg>{fmtStars(r.stars)}</span>{/if}
              </button>
            {/each}
          {/if}
        </div>
      {:else if pc.view === 'tags'}
        <div class="mt-2 rounded-lg border border-[var(--border)] p-2.5">
          <div class="mb-1.5 flex items-center gap-2 px-0.5 text-[11px] text-[var(--text-3)]">
            <span>Recent tags · {pc.repoBase.split('/').pop()}</span>
            {#if pc.busy}<span class="h-2.5 w-2.5 animate-spin rounded-full border-2 border-[var(--border-strong)] border-t-[var(--accent)]"></span>{/if}
          </div>
          {#if !pc.busy && !pc.shownTags.length}
            <div class="px-0.5 py-1 text-[12px] text-[var(--text-3)]">No tags found, type one manually.</div>
          {:else}
            <div class="flex max-h-40 flex-wrap gap-1.5 overflow-auto">
              {#each pc.shownTags.slice(0, 40) as t, i (t)}
                <button class="mono rounded-md border px-2 py-1 text-[11.5px] transition-colors {i === pc.highlight ? 'border-[var(--accent)] bg-[var(--accent-tint)] text-[var(--accent)]' : 'border-[var(--border)] text-[var(--text-2)] hover:border-[var(--border-strong)] hover:bg-[var(--panel-2)]'}" onmouseenter={() => (pc.highlight = i)} onclick={() => pc.pickTag(t)}>{t}</button>
              {/each}
            </div>
          {/if}
        </div>
      {:else if !pc.source.search}
        <p class="mt-2 text-[11.5px] text-[var(--text-3)]">{pc.source.label.split('·')[0].trim()} has no public search, type the full image name above. Pulls work the same.</p>
      {/if}

      {#if pc.error}
        <div class="mono mt-3 break-words rounded-lg border border-[color-mix(in_srgb,var(--red)_35%,var(--border))] bg-[var(--red-tint)] px-3 py-2 text-[12px] text-[var(--red)]">{pc.error}</div>
      {:else if pc.status}
        <div class="mono mt-3 flex items-center gap-2 truncate text-[12px] text-[var(--text-2)]">
          {#if pc.pulling}<span class="h-3 w-3 shrink-0 animate-spin rounded-full border-2 border-[var(--border-strong)] border-t-[var(--accent)]"></span>{/if}
          <span class="truncate">{pc.status}</span>
        </div>
      {/if}
    </div>

    <div class="flex justify-end border-t border-[var(--border)] px-5 py-3">
      <button class="btn btn-default btn-sm" disabled={pc.pulling} onclick={onClose}>{pc.done || !pc.status ? 'Close' : 'Working…'}</button>
    </div>
  </div>
</div>
