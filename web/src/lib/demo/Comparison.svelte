<script>
  // Demo-only: the full Oriel-vs-alternatives breakdown. The README carries a
  // trimmed version; this is the exhaustive one, honest about where Oriel loses.
  import { compare } from './compare.svelte.js'

  const tools = ['Oriel', 'Docker Desktop', 'OrbStack', 'Podman Desktop', 'lazydocker', 'Portainer']

  // A cell is 'y' (yes), 'n' (no), '~' (partial), or any other string (shown verbatim).
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
        { label: 'Idle RAM', cells: ['~15–30 MB', 'Heavy (Electron + VM)', 'Light (native)', 'Heavy (Electron + VM)', 'Light', '≥2 GB host'] },
        { label: 'Bundles its own VM / engine', cells: ['n — uses yours', 'y', 'y', 'y (Podman machine)', 'n', 'n'] },
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
        { label: 'Built-in MCP server (drive via AI)', cells: ['y — safety-gated', 'MCP Toolkit (runs other servers)', 'n', 'n', 'n', 'n'] },
        { label: 'Secret masking + destructive grant', cells: ['y', 'n', 'n', 'n', 'n', 'n'] },
        { label: 'Headless / scriptable', cells: ['y (oriel mcp, CLI)', '~ (CLI)', '~ (CLI)', '~ (CLI)', 'n', 'API'] },
      ],
    },
    {
      title: 'Access & security',
      rows: [
        { label: 'Runs locally, no server', cells: ['y', 'y', 'y', 'y', 'y', 'n (it is the server)'] },
        { label: 'Remote access', cells: ['Reverse proxy + token (private net)', 'n', 'n', '~', 'n', 'y (built for it)'] },
        { label: 'Multi-user auth / RBAC', cells: ['n (single operator)', 'n', 'n', 'n', 'n', 'y'] },
      ],
    },
    {
      title: "Where Oriel doesn't (yet) win — be honest",
      honest: true,
      rows: [
        { label: 'Windows support', cells: ['n (macOS · Linux)', 'y', 'n (macOS only)', 'y', 'y', 'y (server)'] },
        { label: 'In-browser shell / exec UI', cells: ['~ (roadmap)', 'y', 'y', 'y', 'y (exec)', 'y (console)'] },
        { label: 'Kubernetes UI', cells: ['n', 'y', 'y', 'y', 'n', 'y'] },
        { label: 'Multi-host / cluster / teams', cells: ['n', 'n', 'n', 'n', 'n', 'y'] },
        { label: 'Maturity / ecosystem', cells: ['New (2025)', 'Established', 'Established', 'Established', 'Established', 'Established'] },
      ],
    },
  ]

  function close() { compare.open = false }
  function onKey(e) { if (e.key === 'Escape') close() }
</script>

<svelte:window onkeydown={compare.open ? onKey : null} />

{#if compare.open}
  <div class="cmp-backdrop" role="presentation" onclick={(e) => { if (e.target === e.currentTarget) close() }}>
    <div class="cmp" role="dialog" tabindex="-1" aria-modal="true" aria-label="Oriel compared to alternatives">
      <header class="cmp-head">
        <div>
          <h2>How Oriel compares</h2>
          <p>The full breakdown — including where it doesn't win. Tiers and figures drift; treat as a snapshot.</p>
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
        <span>Spot something out of date? <a href="https://github.com/ParadoxInfinite/oriel/issues" target="_blank" rel="noreferrer">Open an issue ↗</a></span>
        <button class="cmp-done" onclick={close}>Done</button>
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
    background: rgb(0 0 0 / 0.55);
    backdrop-filter: blur(3px);
  }
  .cmp {
    display: flex;
    flex-direction: column;
    width: min(1040px, 100%);
    max-height: 88vh;
    background: var(--panel, #1b1b1f);
    color: var(--text, #e7e7ea);
    border: 1px solid var(--border, #34343a);
    border-radius: 14px;
    box-shadow: 0 24px 64px rgb(0 0 0 / 0.45);
    overflow: hidden;
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
  .cmp-scroll { overflow: auto; }
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
  .cmp-foot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 12px 20px;
    border-top: 1px solid var(--border, #34343a);
    font-size: 12px;
    color: var(--text-3, #8a8a93);
  }
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
