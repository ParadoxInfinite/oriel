<script>
  import { audit, loadAudit, registerEscape, trapFocus, t } from '../../../platform/index.js'
  import Icon from './Icon.svelte'

  let { onClose } = $props()

  const argStr = (args) => Object.entries(args || {}).map(([k, v]) => `${k}=${v}`).join('  ')
  function fmtTime(iso) {
    try {
      return new Date(iso).toLocaleString()
    } catch {
      return iso
    }
  }
  $effect(() => registerEscape(() => onClose()))
</script>

<div class="fixed inset-0 z-[70] flex items-start justify-center bg-black/45 p-4 pt-[8vh] backdrop-blur-sm" role="presentation" onclick={(e) => e.target === e.currentTarget && onClose()}>
  <div class="flex max-h-[84vh] w-full max-w-2xl flex-col overflow-hidden rounded-xl border border-[var(--border)] bg-[var(--panel)] shadow-[var(--shadow-lg)]" role="dialog" aria-modal="true" aria-label={t('audit.aria')} tabindex="-1" use:trapFocus>
    <div class="flex items-center gap-2.5 border-b border-[var(--border)] px-5 py-3.5">
      <Icon name="sparkles" size={16} class="text-[var(--text-3)]" />
      <h2 class="text-[14px] font-semibold tracking-tight">{t('audit.title')}</h2>
      <button class="btn btn-sm btn-default ml-auto" onclick={loadAudit} disabled={audit.loading}>{audit.loading ? t('common.loading') : t('audit.refresh')}</button>
      <button class="btn btn-ghost btn-icon btn-sm" title={t('common.close')} aria-label={t('common.close')} onclick={onClose}><Icon name="x" size={15} /></button>
    </div>

    <div class="min-h-0 flex-1 overflow-y-auto p-5">
      <p class="text-[12px] text-[var(--text-3)]">{t('audit.intro')}</p>

      {#if audit.error}
        <p class="mt-3 text-[12px] text-[var(--red)]">{audit.error}</p>
      {:else if !audit.entries.length}
        <p class="mt-3 text-[13px] text-[var(--text-3)]">{t('audit.empty.pre')} <span class="mono">oriel mcp</span> {t('audit.empty.post')}</p>
      {:else}
        <div class="mt-3 flex flex-col gap-1.5">
          {#each audit.entries as e, i (i)}
            <div class="flex items-start gap-2 rounded-lg border border-[var(--border)] bg-[var(--panel-2)] px-3 py-2 text-[12px]">
              <span class="mt-1 h-1.5 w-1.5 shrink-0 rounded-full {e.ok ? 'bg-[var(--green)]' : 'bg-[var(--red)]'}" title={e.ok ? t('audit.status.ok') : t('audit.status.failed')}></span>
              <div class="min-w-0 flex-1">
                <div class="flex items-baseline justify-between gap-2">
                  <span class="mono font-medium text-[var(--text)]">{e.tool}</span>
                  <span class="shrink-0 text-[11px] text-[var(--text-3)]">{fmtTime(e.time)}</span>
                </div>
                {#if e.args}<div class="mono mt-0.5 break-all text-[var(--text-3)]">{argStr(e.args)}</div>{/if}
                {#if e.error}<div class="mt-0.5 text-[var(--red)]">{e.error}</div>{/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>
