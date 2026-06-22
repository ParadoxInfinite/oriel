<script>
  import { confirmState, resolveConfirm } from '../lib/confirm.svelte.js'

  let confirmEl = $state(null)

  $effect(() => {
    if (confirmState.open) queueMicrotask(() => confirmEl?.focus())
  })

  function onKeydown(e) {
    if (!confirmState.open) return
    if (e.key === 'Escape') {
      e.preventDefault()
      resolveConfirm(false)
    } else if (e.key === 'Enter') {
      // If a button is focused, let its native activation handle Enter — so Enter
      // on a focused Cancel cancels, instead of always confirming.
      if (document.activeElement?.tagName === 'BUTTON') return
      e.preventDefault()
      resolveConfirm(true)
    }
  }
</script>

<svelte:window onkeydown={onKeydown} />

{#if confirmState.open}
  <div
    class="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 p-4"
    onclick={() => resolveConfirm(false)}
    role="presentation"
  >
    <div
      class="w-full max-w-sm overflow-hidden rounded-[var(--overlay-radius)] border border-border bg-surface shadow-[var(--overlay-shadow)]"
      onclick={(e) => e.stopPropagation()}
      role="alertdialog"
      aria-modal="true"
    >
      <div class="px-5 pb-4 pt-5">
        <div class="flex items-center gap-2.5">
          <span
            class="grid h-7 w-7 shrink-0 place-items-center rounded-full {confirmState.danger
              ? 'bg-danger/15 text-danger'
              : 'bg-accent/15 text-accent'}"
          >
            <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 9v4" /><path d="M12 17h.01" />
              <path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z" />
            </svg>
          </span>
          <h2 class="display text-sm font-semibold tracking-tight text-fg">{confirmState.title}</h2>
        </div>
        {#if confirmState.message}
          <p class="mt-2.5 text-[13px] leading-relaxed text-muted">{confirmState.message}</p>
        {/if}
        {#if confirmState.checkbox}
          <label class="mt-3 flex cursor-pointer items-center gap-2 text-[13px] text-muted">
            <input type="checkbox" bind:checked={confirmState.checked} class="h-3.5 w-3.5 accent-accent" />
            {confirmState.checkbox}
          </label>
        {/if}
      </div>
      <div class="flex justify-end gap-2 border-t border-border bg-surface-2/40 px-5 py-3">
        <button
          class="rounded-lg border border-border bg-surface px-3.5 py-1.5 text-[13px] font-medium text-muted transition-colors hover:bg-surface-2 hover:text-fg"
          onclick={() => resolveConfirm(false)}
        >
          Cancel
        </button>
        <button
          bind:this={confirmEl}
          class="rounded-lg px-3.5 py-1.5 text-[13px] font-medium shadow-sm transition-[filter] hover:brightness-110 {confirmState.danger
            ? 'bg-danger text-white'
            : 'bg-accent text-accent-fg'}"
          onclick={() => resolveConfirm(true)}
        >
          {confirmState.confirmLabel}
        </button>
      </div>
    </div>
  </div>
{/if}
