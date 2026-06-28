<script>
  // Shown only in the VITE_DEMO (GitHub Pages) build. A small fixed pill making
  // clear nothing here is a real Docker host and that a refresh resets state.
  import { t, locale, AVAILABLE, setLocale } from '../locale.svelte.js'
  let dismissed = $state(false)
</script>

{#if !dismissed}
  <div class="demo-pill">
    <span class="dot"></span>
    <span class="txt"><strong>{t('demo.banner.tag')}</strong> · {t('demo.banner.text')}</span>
    {#if AVAILABLE.length > 1}
      <select class="lang" value={locale.tag} aria-label={t('demo.banner.language')} onchange={(e) => setLocale(e.currentTarget.value)}>
        {#each AVAILABLE as l (l.tag)}<option value={l.tag}>{l.name}</option>{/each}
      </select>
    {/if}
    <a href="https://github.com/ParadoxInfinite/oriel" target="_blank" rel="noreferrer">{t('demo.banner.github')}</a>
    <button aria-label={t('demo.banner.dismiss')} onclick={() => (dismissed = true)}>×</button>
  </div>
{/if}

<style>
  .demo-pill {
    position: fixed;
    bottom: 16px;
    left: 50%;
    transform: translateX(-50%);
    z-index: 9999;
    display: flex;
    align-items: center;
    gap: 10px;
    max-width: calc(100vw - 24px);
    padding: 8px 10px 8px 14px;
    font-size: 12.5px;
    color: var(--text, #e7e7ea);
    background: color-mix(in srgb, var(--panel, #1b1b1f) 88%, transparent);
    border: 1px solid var(--border, #34343a);
    border-radius: 999px;
    backdrop-filter: blur(8px);
    box-shadow: 0 6px 24px rgb(0 0 0 / 0.28);
  }
  .txt { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .dot {
    width: 8px;
    height: 8px;
    border-radius: 999px;
    background: var(--accent, #6ea8fe);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent, #6ea8fe) 25%, transparent);
    flex: none;
  }
  a {
    color: var(--accent, #6ea8fe);
    text-decoration: none;
    font-weight: 600;
    flex: none;
  }
  a:hover { text-decoration: underline; }
  .lang {
    flex: none;
    padding: 2px 6px;
    font-size: 12px;
    color: var(--text, #e7e7ea);
    background: var(--chip-bg, #2a2a30);
    border: 1px solid var(--border, #34343a);
    border-radius: 999px;
    cursor: pointer;
  }
  button {
    flex: none;
    width: 20px;
    height: 20px;
    border: 0;
    border-radius: 999px;
    background: transparent;
    color: var(--text-3, #8a8a93);
    font-size: 16px;
    line-height: 1;
    cursor: pointer;
  }
  button:hover { color: var(--text, #e7e7ea); background: var(--chip-bg, #2a2a30); }
</style>
