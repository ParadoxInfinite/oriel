<script>
  import { toasts, dismissToast } from '../lib/toast.svelte.js'

  const color = {
    ok: 'border-ok/40 text-ok',
    error: 'border-danger/40 text-danger',
    info: 'border-border text-fg',
  }
</script>

<div class="fixed bottom-4 right-4 z-[60] flex w-80 flex-col gap-2">
  {#each toasts.items as t (t.id)}
    <button
      class="rounded-[--radius] border bg-surface px-4 py-2.5 text-left text-sm shadow-lg {color[
        t.kind
      ] || color.info}"
      onclick={() => dismissToast(t.id)}
    >
      {t.message}
    </button>
  {/each}
</div>
