<script>
  import { toasts, dismissToast } from '../lib/toast.svelte.js'
  import { t } from '../platform/index.js'

  const color = {
    ok: 'border-ok/40 text-ok',
    error: 'border-danger/40 text-danger',
    info: 'border-border text-fg',
  }
</script>

<div class="fixed bottom-4 right-4 z-[60] flex w-80 flex-col gap-2" aria-live="polite" aria-atomic="false">
  {#each toasts.items as item (item.id)}
    <button
      class="rounded-[--radius] border bg-surface px-4 py-2.5 text-left text-sm shadow-lg {color[
        item.kind
      ] || color.info}"
      onclick={() => dismissToast(item.id)}
      title={t('toast.dismiss')}
    >
      {item.message}
    </button>
  {/each}
</div>
