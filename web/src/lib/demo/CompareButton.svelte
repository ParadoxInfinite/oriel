<script>
  // Demo-only: a fixed, always-visible button that opens the comparison overlay.
  // The demo pill is dismissable; this isn't, so the comparison stays reachable.
  import { compare } from './compare.svelte.js'
  import { t } from '../locale.svelte.js'
</script>

<button class="cmp-fab" onclick={() => (compare.open = true)} title={t('demo.compareButton.title')}>
  <span class="ic" aria-hidden="true">⚖</span>
  <span>{t('demo.compareButton.label')}</span>
</button>

<style>
  .cmp-fab {
    position: fixed;
    right: 18px;
    bottom: 18px;
    z-index: 9998;
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 10px 16px;
    border: 0;
    border-radius: 999px;
    background: var(--accent, #6ea8fe);
    color: #08131f;
    font-size: 13px;
    font-weight: 650;
    cursor: pointer;
    box-shadow: 0 6px 22px rgb(0 0 0 / 0.3);
    transition: transform 0.12s ease, box-shadow 0.12s ease;
  }
  .cmp-fab:hover {
    transform: translateY(-1px);
    box-shadow: 0 10px 28px rgb(0 0 0 / 0.38);
  }
  .cmp-fab:active { transform: translateY(0); }
  .ic { font-size: 15px; line-height: 1; }
  @media (max-width: 640px) {
    .cmp-fab { right: 12px; bottom: 12px; padding: 9px 13px; }
    .cmp-fab span:not(.ic) { display: none; }
  }
</style>
