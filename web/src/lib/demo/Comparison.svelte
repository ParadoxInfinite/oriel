<script>
  // Demo-only: the full Oriel-vs-alternatives breakdown. The README carries a
  // trimmed version; this is the exhaustive one, honest about where Oriel loses
  // and what it'll do (or won't) about each gap.
  import { onMount } from 'svelte'
  import { fade, scale } from 'svelte/transition'
  import { cubicOut } from 'svelte/easing'
  import { compare } from './compare.svelte.js'
  import { trapFocus } from '../focustrap.js'

  // Deep link: /#compare opens this straight away, so the README can point at it.
  onMount(() => {
    const sync = () => { if (location.hash.slice(1) === 'compare') compare.open = true }
    sync()
    window.addEventListener('hashchange', sync)
    return () => window.removeEventListener('hashchange', sync)
  })

  const tools = ['Oriel', 'Docker Desktop', 'OrbStack', 'Podman Desktop', 'lazydocker', 'Portainer']

  // A cell is 'y' (yes), 'n' (no), '~' (partial), a string (verbatim), or an
  // object {v, tag, note} that pairs a value with a roadmap/by-design label.
  const groups = [
    {
      title: 'Cost & licensing',
      rows: [
        { label: 'License', cells: ['Apache-2.0', 'Proprietary', 'Proprietary', 'Apache-2.0', 'MIT', 'zlib (CE) / paid (Business)'] },
        { label: 'Free for commercial use', cells: ['y', { v: '~', note: 'free under ~250 staff / $10M' }, { v: 'n', note: 'paid for commercial' }, 'y', 'y', { v: '~', note: 'CE free; Business paid' }] },
        { label: 'Account / sign-in required', cells: ['n', { v: '~', note: 'increasingly pushed' }, 'n', 'n', 'n', { v: 'y', note: 'it has user logins' }] },
      ],
    },
    {
      title: 'Footprint & install',
      rows: [
        { label: 'Distribution', cells: ['Single static binary', 'Installer', 'Installer', 'Installer', 'Single binary', 'Container'] },
        { label: 'Download / size', cells: ['~13 MB binary', '~600 MB → ~1.5 GB app', '~150 MB app', '~250 MB app', '~25 MB binary', '~300 MB image'] },
        { label: 'Idle RAM *', cells: ['~15-30 MB', '~3-4 GB', { v: '~0.2-1 GB', note: 'dynamic' }, '~2 GB', '~tens of MB', '~200-300 MB'] },
        { label: 'Bundles its own VM / engine', cells: [{ v: 'n', note: 'uses your engine' }, { v: 'y', note: 'lock-in' }, { v: 'y', note: 'lock-in' }, { v: 'y', note: 'Podman machine' }, 'n', 'n'] },
      ],
    },
    {
      title: 'Engine support',
      rows: [
        { label: 'Bring-your-own engine', cells: [{ v: 'Any engine / socket', note: 'Colima · Docker · OrbStack · Podman · remote, no lock-in' }, { v: 'Bundled', note: 'lock-in' }, { v: 'Bundled', note: 'lock-in' }, { v: 'Podman', note: '+ Docker-compat socket' }, 'Any Docker socket', 'Any Docker / K8s endpoint'] },
        { label: 'Manages the VM lifecycle', cells: [{ v: 'y', note: 'Colima start/stop' }, { v: 'y', note: 'its own' }, { v: 'y', note: 'its own' }, { v: 'y', note: 'Podman machine' }, 'n', 'n'] },
      ],
    },
    {
      title: 'Interface',
      rows: [
        { label: 'Type', cells: ['Web GUI', 'Desktop app', 'Native app', 'Desktop app', 'Terminal (TUI)', 'Web UI (server)'] },
        { label: 'Runs in the browser', cells: ['y', 'n', 'n', 'n', 'n', 'y'] },
        { label: 'Themes / swappable editions', cells: ['y', 'n', 'n', '~', 'n', '~'] },
        { label: 'Command palette', cells: [{ v: 'y', note: '⌘K / Ctrl-K' }, 'n', 'n', 'n', { v: '~', note: 'keyboard-driven' }, 'n'] },
      ],
    },
    {
      title: 'Features',
      rows: [
        { label: 'Containers: logs · stats · inspect', cells: ['y', 'y', 'y', 'y', 'y', 'y'] },
        { label: 'Images: pull · search · prune · tag', cells: ['y', 'y', 'y', 'y', '~', 'y'] },
        { label: 'Compose: manage', cells: ['y', 'y', 'y', 'y', 'y', 'y'] },
        { label: 'Compose: discover from disk', cells: ['y', 'n', 'n', 'n', 'n', 'n'] },
        { label: 'Dashboard: CPU/mem/disk history', cells: ['y', '~', '~', '~', '~', 'y'] },
      ],
    },
    {
      title: 'AI & automation',
      rows: [
        { label: 'Built-in MCP server (drive via AI)', cells: [{ v: 'y', note: 'safety-gated' }, { v: '~', note: 'MCP Toolkit runs other servers' }, 'n', 'n', 'n', 'n'] },
        { label: 'Secret masking + destructive grant', cells: ['y', 'n', 'n', 'n', 'n', 'n'] },
        { label: 'Headless / scriptable', cells: [{ v: 'y', note: 'oriel mcp, CLI' }, { v: '~', note: 'CLI' }, { v: '~', note: 'CLI' }, { v: '~', note: 'CLI' }, 'n', { v: 'y', note: 'HTTP API' }] },
      ],
    },
    {
      title: 'Access & security',
      rows: [
        { label: 'Runs locally, no server', cells: ['y', 'y', 'y', 'y', 'y', { v: 'n', note: 'it is the server' }] },
        { label: 'Remote access', cells: [{ v: '~', note: 'reverse proxy + token, private net' }, 'n', 'n', '~', 'n', { v: 'y', note: 'built for it' }] },
      ],
    },
    {
      title: "Where Oriel is weaker, and what we'll do about it",
      honest: true,
      rows: [
        { label: 'Windows', cells: [{ v: 'n', tag: 'demand' }, 'y', { v: 'n', note: 'macOS only' }, 'y', 'y', { v: 'y', note: 'server' }] },
        { label: 'Native desktop app', cells: [{ v: 'n', tag: 'design', note: 'web UI on purpose' }, { v: 'y', note: 'Electron' }, { v: 'y', note: 'native' }, { v: 'y', note: 'Electron' }, { v: 'n', note: 'terminal' }, { v: 'n', note: 'web' }] },
        { label: 'In-browser shell / exec', cells: [{ v: 'n', tag: 'road' }, 'y', 'y', 'y', 'y', 'y'] },
        { label: 'Kubernetes', cells: [{ v: 'n', tag: 'scope' }, 'y', 'y', 'y', 'n', 'y'] },
        { label: 'Multi-host / clusters / teams', cells: [{ v: 'n', tag: 'design', note: 'single-operator, single-host' }, 'n', 'n', 'n', 'n', 'y'] },
        { label: 'Audit log of actions', cells: [{ v: 'n', tag: 'road' }, 'n', 'n', 'n', 'n', { v: '~', note: 'Business edition' }] },
        { label: 'Maturity & ecosystem', cells: [{ v: 'New (2025)', note: 'small, moving fast' }, { v: 'Huge', note: 'industry default, extensions' }, { v: 'Growing', note: 'popular on Mac' }, { v: 'Red Hat-backed', note: 'extensions' }, { v: 'Popular OSS', note: 'big following' }, { v: 'Mature', note: 'large enterprise base' }] },
      ],
    },
  ]

  const TAGS = {
    road: { label: 'on the roadmap', cls: 'road' },
    design: { label: 'by design', cls: 'design' },
    scope: { label: 'out of scope', cls: 'design' },
    demand: { label: 'demand-gated', cls: 'demand' },
  }

  function close() {
    compare.open = false
    if (location.hash.slice(1) === 'compare') history.replaceState(null, '', location.pathname + location.search)
  }
  function onKey(e) { if (e.key === 'Escape') close() }
</script>

<svelte:window onkeydown={compare.open ? onKey : null} />

{#if compare.open}
  <div class="cmp-backdrop" role="presentation" transition:fade={{ duration: 160 }} onclick={(e) => { if (e.target === e.currentTarget) close() }}>
    <div
      class="cmp"
      role="dialog"
      tabindex="-1"
      aria-modal="true"
      aria-label="Oriel compared to alternatives"
      use:trapFocus
      in:scale={{ duration: 210, start: 0.97, opacity: 0, easing: cubicOut }}
      out:scale={{ duration: 130, start: 0.98, opacity: 0, easing: cubicOut }}
    >
      <header class="cmp-head">
        <div>
          <h2>How Oriel compares</h2>
          <p>The full breakdown, including where it loses. Tiers and figures drift; treat as a snapshot.</p>
        </div>
        <button class="cmp-x" aria-label="Close" onclick={close}>×</button>
      </header>

      <div class="cmp-scroll">
        <table>
          <thead>
            <tr>
              <th class="rowhead"></th>
              {#each tools as t}<th class:me={t === 'Oriel'}>{t}</th>{/each}
            </tr>
          </thead>
          <tbody>
            {#each groups as g}
              <tr class="grouprow" class:honest={g.honest}><td colspan={tools.length + 1}>{g.title}</td></tr>
              {#each g.rows as r}
                <tr>
                  <td class="rowhead">{r.label}</td>
                  {#each r.cells as c, i}
                    {@const cell = c && typeof c === 'object' ? c : { v: c }}
                    <td class:me={i === 0}>
                      {#if cell.v === 'y'}<span class="yes">✓</span>
                      {:else if cell.v === 'n'}<span class="no">✕</span>
                      {:else if cell.v === '~'}<span class="part">◐</span>
                      {:else}<span class="txt">{cell.v}</span>{/if}
                      {#if cell.tag}<span class="tag {TAGS[cell.tag].cls}">{TAGS[cell.tag].label}</span>{/if}
                      {#if cell.note}<span class="cellnote">{cell.note}</span>{/if}
                    </td>
                  {/each}
                </tr>
              {/each}
            {/each}
          </tbody>
        </table>
      </div>

      <p class="cmp-fn">* Idle RAM is the tool itself. Oriel, lazydocker, and Portainer drive the engine you already run (its RAM is separate); Docker Desktop, OrbStack, and Podman Desktop bundle a Linux VM, so theirs includes it. Figures are rough, vary by machine and settings, and drift over time.</p>

      <footer class="cmp-foot">
        <div class="legend">
          <span class="tag road">on the roadmap</span> building it
          <span class="sep">·</span>
          <span class="tag design">by design</span> we deliberately won't
          <span class="sep">·</span>
          <span class="tag demand">demand-gated</span> only if enough ask
        </div>
        <div class="foot-actions">
          <a href="https://github.com/ParadoxInfinite/oriel/blob/main/ROADMAP.md" target="_blank" rel="noreferrer">Roadmap ↗</a>
          <a href="https://github.com/ParadoxInfinite/oriel/issues" target="_blank" rel="noreferrer">Wrong? Open an issue ↗</a>
          <button class="cmp-done" onclick={close}>Done</button>
        </div>
      </footer>
    </div>
  </div>
{/if}

<style>
  .cmp-backdrop {
    position: fixed;
    inset: 0;
    z-index: 10000;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    background: rgb(0 0 0 / 0.58);
  }
  .cmp {
    display: flex;
    flex-direction: column;
    width: min(1060px, 100%);
    max-height: 88vh;
    background: var(--panel, #1b1b1f);
    color: var(--text, #e7e7ea);
    border: 1px solid var(--border, #34343a);
    border-radius: 14px;
    box-shadow: 0 24px 64px rgb(0 0 0 / 0.45);
    overflow: hidden;
    will-change: transform, opacity;
  }
  .cmp-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    padding: 18px 20px;
    border-bottom: 1px solid var(--border, #34343a);
  }
  .cmp-head h2 { margin: 0; font-size: 17px; font-weight: 650; }
  .cmp-head p { margin: 4px 0 0; font-size: 12.5px; color: var(--text-3, #8a8a93); }
  .cmp-x {
    flex: none;
    width: 30px;
    height: 30px;
    border: 0;
    border-radius: 8px;
    background: transparent;
    color: var(--text-3, #8a8a93);
    font-size: 20px;
    cursor: pointer;
  }
  .cmp-x:hover { background: var(--chip-bg, #2a2a30); color: var(--text, #e7e7ea); }
  .cmp-scroll { overflow: auto; overscroll-behavior: contain; }
  table { width: 100%; border-collapse: collapse; font-size: 12.5px; }
  th, td { padding: 8px 12px; text-align: center; border-bottom: 1px solid color-mix(in srgb, var(--border, #34343a) 60%, transparent); }
  thead th { position: sticky; top: 0; background: var(--panel, #1b1b1f); z-index: 1; font-weight: 600; }
  .rowhead { text-align: left; color: var(--text-2, #c7c7cf); white-space: nowrap; }
  td.rowhead { font-weight: 500; }
  .me { background: color-mix(in srgb, var(--accent, #6ea8fe) 12%, transparent); }
  thead th.me { color: var(--accent, #6ea8fe); }
  .grouprow td {
    text-align: left;
    padding: 12px 12px 6px;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--text-3, #8a8a93);
    background: color-mix(in srgb, var(--panel-2, #232329) 60%, transparent);
  }
  .grouprow.honest td { color: var(--amber, #e0a458); }
  .yes { color: var(--ok, #59c27a); font-weight: 700; }
  .no { color: var(--text-3, #8a8a93); }
  .part { color: var(--amber, #e0a458); }
  .txt { color: var(--text, #e7e7ea); }
  td.me .txt { font-weight: 600; }
  .cellnote { display: block; font-size: 10.5px; color: var(--text-3, #8a8a93); margin-top: 2px; }
  .cmp-fn { margin: 0; padding: 10px 20px; font-size: 11px; line-height: 1.5; color: var(--text-3, #8a8a93); border-top: 1px solid color-mix(in srgb, var(--border, #34343a) 50%, transparent); }
  .tag {
    display: inline-block;
    margin-left: 5px;
    padding: 1px 7px;
    border-radius: 999px;
    font-size: 10px;
    font-weight: 600;
    white-space: nowrap;
    vertical-align: middle;
  }
  .tag.road { background: color-mix(in srgb, var(--accent, #6ea8fe) 18%, transparent); color: var(--accent, #6ea8fe); }
  .tag.design { background: color-mix(in srgb, var(--text-3, #8a8a93) 22%, transparent); color: var(--text-2, #c7c7cf); }
  .tag.demand { background: color-mix(in srgb, var(--amber, #e0a458) 18%, transparent); color: var(--amber, #e0a458); }
  .cmp-foot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 14px;
    flex-wrap: wrap;
    padding: 12px 20px;
    border-top: 1px solid var(--border, #34343a);
    font-size: 11.5px;
    color: var(--text-3, #8a8a93);
  }
  .legend { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
  .legend .sep { opacity: 0.5; }
  .foot-actions { display: flex; align-items: center; gap: 14px; }
  .cmp-foot a { color: var(--accent, #6ea8fe); text-decoration: none; font-weight: 600; }
  .cmp-foot a:hover { text-decoration: underline; }
  .cmp-done {
    border: 1px solid var(--border, #34343a);
    background: var(--accent, #6ea8fe);
    color: #08131f;
    font-weight: 600;
    padding: 7px 16px;
    border-radius: 8px;
    cursor: pointer;
  }
</style>
