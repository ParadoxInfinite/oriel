<script>
  // Demo-only: the full Oriel-vs-alternatives breakdown. The README carries a
  // trimmed version; this is the exhaustive one, honest about where Oriel loses
  // and what it'll do (or won't) about each gap.
  import { onMount } from 'svelte'
  import { fade, scale } from 'svelte/transition'
  import { cubicOut } from 'svelte/easing'
  import { compare } from './compare.svelte.js'

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
        { label: 'Free for commercial use', cells: ['y', 'Paid above ~250 staff / $10M', 'Paid for commercial', 'y', 'y', 'CE free; Business paid'] },
        { label: 'Account / sign-in required', cells: ['n', '~ (pushed)', 'n', 'n', 'n', 'Server login (it has users)'] },
      ],
    },
    {
      title: 'Footprint & install',
      rows: [
        { label: 'Distribution', cells: ['Single static binary', 'Installer', 'Installer', 'Installer', 'Single binary', 'Run a container'] },
        { label: 'Size', cells: ['~13 MB', 'Hundreds of MB', '~tens of MB', 'Hundreds of MB', '~tens of MB', 'Container image'] },
        { label: 'Idle RAM', cells: ['~15-30 MB', 'Heavy (Electron + VM)', 'Light (native)', 'Heavy (Electron + VM)', 'Light', '≥2 GB host'] },
        { label: 'Bundles its own VM / engine', cells: ['No, uses yours', 'y', 'y', 'y (Podman machine)', 'n', 'n'] },
      ],
    },
    {
      title: 'Engine support',
      rows: [
        { label: 'Bring-your-own engine', cells: ['Colima · Docker · OrbStack · Podman · remote', 'Bundled', 'Bundled', 'Podman (+ Docker compat)', 'Any Docker socket', 'Any Docker / K8s endpoint'] },
        { label: 'Manages the VM lifecycle', cells: ['y (Colima start/stop)', 'y (its own)', 'y (its own)', 'y (Podman machine)', 'n', 'n'] },
      ],
    },
    {
      title: 'Interface',
      rows: [
        { label: 'Type', cells: ['Web GUI', 'Desktop app', 'Native app', 'Desktop app', 'Terminal (TUI)', 'Web UI (server)'] },
        { label: 'Runs in the browser', cells: ['y', 'n', 'n', 'n', 'n', 'y'] },
        { label: 'Themes / swappable editions', cells: ['y', 'n', 'n', '~', 'n', '~'] },
        { label: 'Command palette', cells: ['y (⌘K)', 'n', 'n', 'n', '~ (keys)', 'n'] },
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
        { label: 'Built-in MCP server (drive via AI)', cells: ['y, safety-gated', 'MCP Toolkit (runs other servers)', 'n', 'n', 'n', 'n'] },
        { label: 'Secret masking + destructive grant', cells: ['y', 'n', 'n', 'n', 'n', 'n'] },
        { label: 'Headless / scriptable', cells: ['y (oriel mcp, CLI)', '~ (CLI)', '~ (CLI)', '~ (CLI)', 'n', 'API'] },
      ],
    },
    {
      title: 'Access & security',
      rows: [
        { label: 'Runs locally, no server', cells: ['y', 'y', 'y', 'y', 'y', 'No, it is the server'] },
        { label: 'Remote access', cells: ['Reverse proxy + token (private net)', 'n', 'n', '~', 'n', 'y (built for it)'] },
      ],
    },
    {
      title: "Where Oriel is weaker, and what we'll do about it",
      honest: true,
      rows: [
        { label: 'Windows', cells: [{ v: 'No', tag: 'demand' }, 'y', 'No (macOS only)', 'y', 'y', 'y (server)'] },
        { label: 'Native desktop app', cells: [{ v: 'No', tag: 'design', note: 'web UI on purpose' }, 'y (Electron)', 'y (native)', 'y (Electron)', 'No (terminal)', 'No (web)'] },
        { label: 'In-browser shell / exec', cells: [{ v: 'Not yet', tag: 'road' }, 'y', 'y', 'y', 'y', 'y'] },
        { label: 'Kubernetes', cells: [{ v: 'No', tag: 'scope' }, 'y', 'y', 'y', 'n', 'y'] },
        { label: 'Multi-host / clusters / teams', cells: [{ v: 'No', tag: 'design', note: 'single host by design' }, 'n', 'n', 'n', 'n', 'y'] },
        { label: 'Per-user identity / RBAC for remote', cells: [{ v: 'Not yet', tag: 'road', note: 'richer auth planned' }, 'n', 'n', 'n', 'n', 'y'] },
        { label: 'Audit log of AI/automation actions', cells: [{ v: 'Not yet', tag: 'road' }, 'n', 'n', 'n', 'n', '~'] },
        { label: 'Maturity / ecosystem', cells: ['New (2025), small', 'Established', 'Established', 'Established', 'Established', 'Established'] },
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
                    <td class:me={i === 0}>
                      {#if c === 'y'}<span class="yes">✓</span>
                      {:else if c === 'n'}<span class="no">✕</span>
                      {:else if c === '~'}<span class="part">◐</span>
                      {:else if typeof c === 'object'}
                        <span class="txt">{c.v}</span>
                        <span class="tag {TAGS[c.tag].cls}">{TAGS[c.tag].label}</span>
                        {#if c.note}<span class="cellnote">{c.note}</span>{/if}
                      {:else}<span class="txt">{c}</span>{/if}
                    </td>
                  {/each}
                </tr>
              {/each}
            {/each}
          </tbody>
        </table>
      </div>

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
