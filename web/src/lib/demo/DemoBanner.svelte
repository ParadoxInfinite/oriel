<script>
  // Shown only in the VITE_DEMO (GitHub Pages) build. A small fixed pill making
  // clear nothing here is a real Docker host and that a refresh resets state.
  let dismissed = $state(false)
</script>

{#if !dismissed}
  <div class="demo-pill">
    <span class="dot"></span>
    <span class="txt"><strong>Demo</strong> · mock data, no real Docker, a refresh resets it.</span>
    <a href="https://github.com/ParadoxInfinite/oriel" target="_blank" rel="noreferrer">GitHub ↗</a>
    <button aria-label="Dismiss" onclick={() => (dismissed = true)}>×</button>
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
