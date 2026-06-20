<script>
  import { baseName } from '../lib/pathfield.svelte.js'

  // `field` is a PathField instance; `onEnter` fires on Enter with no suggestion
  // highlighted ("accept what I typed").
  let { field, placeholder = '/absolute/path', onEnter } = $props()
  const f = 'w-full rounded-[--radius] border border-border bg-bg px-3 py-1.5 font-mono text-sm outline-none placeholder:text-muted focus:border-accent/50'

  let el = $state(null)
  let dropUp = $state(false)
  // Flip the suggestion list above the field when it would run off the bottom.
  function place() {
    if (!el) return
    const r = el.getBoundingClientRect()
    const below = window.innerHeight - r.bottom
    dropUp = below < 264 && r.top > below
  }
</script>

<div class="relative min-w-0 flex-1">
  <input
    bind:this={el}
    class={f}
    bind:value={field.value}
    {placeholder}
    autocomplete="off"
    spellcheck="false"
    oninput={() => { place(); field.input() }}
    onfocus={() => { place(); field.focus() }}
    onblur={field.blur}
    onkeydown={(e) => {
      field.keydown(e)
      if (e.key === 'Enter' && field.highlight < 0) onEnter?.()
    }}
  />
  {#if field.open && field.entries.length}
    <div class="absolute left-0 right-0 z-20 max-h-60 overflow-auto rounded-[--radius] border border-border bg-surface shadow-2xl {dropUp ? 'bottom-full mb-1' : 'top-full mt-1'}">
      {#each field.entries as p, i (p)}
        <button
          class="flex w-full items-center gap-2 px-3 py-1.5 text-left transition-colors {i === field.highlight ? 'bg-surface-2' : 'hover:bg-surface-2/50'}"
          onmousedown={(e) => {
            e.preventDefault()
            field.pick(p)
          }}
        >
          <span class="truncate font-mono text-[13px] text-fg">{baseName(p)}</span>
        </button>
      {/each}
    </div>
  {/if}
</div>
