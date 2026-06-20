<script>
  import { baseName } from '../../../platform/index.js'
  import Icon from './Icon.svelte'

  // `field` is a platform PathField instance; `onEnter` fires on Enter with no
  // suggestion highlighted (i.e. "accept what I typed").
  let { field, placeholder = '/absolute/path', onEnter } = $props()

  let el = $state(null)
  let dropUp = $state(false)
  // Open the suggestion list upward when it would otherwise run past the
  // bottom of the viewport (e.g. this field sits low in the Settings grid).
  function place() {
    if (!el) return
    const r = el.getBoundingClientRect()
    const below = window.innerHeight - r.bottom
    dropUp = below < 264 && r.top > below
  }
</script>

<div class="relative min-w-0 flex-1">
  <Icon name="layers" size={14} class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-3)]" />
  <input
    bind:this={el}
    class="input has-icon mono w-full"
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
    <div class="absolute left-0 right-0 z-20 max-h-60 overflow-auto rounded-lg border border-[var(--border)] bg-[var(--panel)] shadow-[var(--shadow-md)] {dropUp ? 'bottom-full mb-1' : 'top-full mt-1'}">
      {#each field.entries as p, i (p)}
        <button
          class="flex w-full items-center gap-2 px-3 py-1.5 text-left {i === field.highlight ? 'bg-[var(--accent-tint)]' : 'hover:bg-[var(--panel-2)]'}"
          onmousedown={(e) => {
            e.preventDefault()
            field.pick(p)
          }}
        >
          <Icon name="database" size={13} class="shrink-0 text-[var(--text-3)]" />
          <span class="mono truncate text-[12.5px] text-[var(--text)]">{baseName(p)}</span>
        </button>
      {/each}
    </div>
  {/if}
</div>
