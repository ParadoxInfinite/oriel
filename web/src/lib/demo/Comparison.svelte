<script>
  // Demo-only: the full Oriel-vs-alternatives breakdown. The README carries a
  // trimmed version; this is the exhaustive one, honest about where Oriel loses
  // and what it'll do (or won't) about each gap.
  import { onMount } from 'svelte'
  import { fade, scale } from 'svelte/transition'
  import { cubicOut } from 'svelte/easing'
  import { compare } from './compare.svelte.js'
  import { trapFocus } from '../focustrap.js'
  import { t } from '../locale.svelte.js'

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
  // $derived so labels/notes re-translate when the language switches.
  const groups = $derived([
    {
      title: t('demo.compare.group.cost'),
      rows: [
        { label: t('demo.compare.row.license'), cells: ['Apache-2.0', t('demo.compare.cell.proprietary'), t('demo.compare.cell.proprietary'), 'Apache-2.0', 'MIT', t('demo.compare.cell.licenseZlib')] },
        { label: t('demo.compare.row.freeCommercial'), cells: ['y', { v: '~', note: t('demo.compare.note.freeUnder250') }, { v: 'n', note: t('demo.compare.note.paidCommercial') }, 'y', 'y', { v: '~', note: t('demo.compare.note.ceFreeBizPaid') }] },
        { label: t('demo.compare.row.account'), cells: ['n', { v: '~', note: t('demo.compare.note.pushed') }, 'n', 'n', 'n', { v: 'y', note: t('demo.compare.note.userLogins') }] },
      ],
    },
    {
      title: t('demo.compare.group.footprint'),
      rows: [
        { label: t('demo.compare.row.distribution'), cells: [t('demo.compare.cell.distSingleStatic'), t('demo.compare.cell.distInstaller'), t('demo.compare.cell.distInstaller'), t('demo.compare.cell.distInstaller'), t('demo.compare.cell.distSingleBinary'), t('demo.compare.cell.distContainer')] },
        { label: t('demo.compare.row.size'), cells: [t('demo.compare.cell.size13'), t('demo.compare.cell.size600'), t('demo.compare.cell.size150'), t('demo.compare.cell.size250'), t('demo.compare.cell.size25'), t('demo.compare.cell.size300')] },
        { label: t('demo.compare.row.idleRam'), cells: ['~15-30 MB', '~3-4 GB', { v: '~0.2-1 GB', note: t('demo.compare.note.dynamic') }, '~2 GB', t('demo.compare.cell.ramTens'), '~200-300 MB'] },
        { label: t('demo.compare.row.bundlesVm'), cells: [{ v: 'n', note: t('demo.compare.note.usesYourEngine') }, { v: 'y', note: t('demo.compare.note.lockin') }, { v: 'y', note: t('demo.compare.note.lockin') }, { v: 'y', note: t('demo.compare.note.podmanMachine') }, 'n', 'n'] },
      ],
    },
    {
      title: t('demo.compare.group.engine'),
      rows: [
        { label: t('demo.compare.row.byoEngine'), cells: [{ v: t('demo.compare.cell.byoAny'), note: t('demo.compare.note.byoEngines') }, { v: t('demo.compare.cell.byoBundled'), note: t('demo.compare.note.lockin') }, { v: t('demo.compare.cell.byoBundled'), note: t('demo.compare.note.lockin') }, { v: 'Podman', note: t('demo.compare.note.dockerCompat') }, t('demo.compare.cell.byoAnyDocker'), t('demo.compare.cell.byoAnyK8s')] },
        { label: t('demo.compare.row.vmLifecycle'), cells: [{ v: 'y', note: t('demo.compare.note.colimaStartStop') }, { v: 'y', note: t('demo.compare.note.itsOwn') }, { v: 'y', note: t('demo.compare.note.itsOwn') }, { v: 'y', note: t('demo.compare.note.podmanMachine') }, 'n', 'n'] },
      ],
    },
    {
      title: t('demo.compare.group.interface'),
      rows: [
        { label: t('demo.compare.row.interfaceType'), cells: [t('demo.compare.cell.typeWebGui'), t('demo.compare.cell.typeDesktop'), t('demo.compare.cell.typeNative'), t('demo.compare.cell.typeDesktop'), t('demo.compare.cell.typeTerminal'), t('demo.compare.cell.typeWebServer')] },
        { label: t('demo.compare.row.runsBrowser'), cells: ['y', 'n', 'n', 'n', 'n', 'y'] },
        { label: t('demo.compare.row.themes'), cells: ['y', 'n', 'n', '~', 'n', '~'] },
        { label: t('demo.compare.row.commandPalette'), cells: [{ v: 'y', note: '⌘K / Ctrl-K' }, 'n', 'n', 'n', { v: '~', note: t('demo.compare.note.keyboardDriven') }, 'n'] },
      ],
    },
    {
      title: t('demo.compare.group.features'),
      rows: [
        { label: t('demo.compare.row.containers'), cells: ['y', 'y', 'y', 'y', 'y', 'y'] },
        { label: t('demo.compare.row.images'), cells: ['y', 'y', 'y', 'y', '~', 'y'] },
        { label: t('demo.compare.row.composeManage'), cells: ['y', 'y', 'y', 'y', 'y', 'y'] },
        { label: t('demo.compare.row.composeDiscover'), cells: ['y', 'n', 'n', 'n', 'n', 'n'] },
        { label: t('demo.compare.row.dashboard'), cells: ['y', '~', '~', '~', '~', 'y'] },
        { label: t('demo.compare.row.browserShell'), cells: ['y', 'y', 'y', 'y', 'y', 'y'] },
      ],
    },
    {
      title: t('demo.compare.group.ai'),
      rows: [
        { label: t('demo.compare.row.mcp'), cells: [{ v: 'y', note: t('demo.compare.note.safetyGated') }, { v: '~', note: t('demo.compare.note.mcpToolkit') }, 'n', 'n', 'n', 'n'] },
        { label: t('demo.compare.row.secretMasking'), cells: ['y', 'n', 'n', 'n', 'n', 'n'] },
        { label: t('demo.compare.row.auditLog'), cells: ['y', 'n', 'n', 'n', 'n', { v: '~', note: t('demo.compare.note.businessEdition') }] },
        { label: t('demo.compare.row.headless'), cells: [{ v: 'y', note: 'oriel mcp, CLI' }, { v: '~', note: 'CLI' }, { v: '~', note: 'CLI' }, { v: '~', note: 'CLI' }, 'n', { v: 'y', note: 'HTTP API' }] },
      ],
    },
    {
      title: t('demo.compare.group.access'),
      rows: [
        { label: t('demo.compare.row.runsLocally'), cells: ['y', 'y', 'y', 'y', 'y', { v: 'n', note: t('demo.compare.note.itIsServer') }] },
        { label: t('demo.compare.row.remoteAccess'), cells: [{ v: '~', note: t('demo.compare.note.reverseProxy') }, 'n', 'n', '~', 'n', { v: 'y', note: t('demo.compare.note.builtForIt') }] },
      ],
    },
    {
      title: t('demo.compare.group.weaker'),
      honest: true,
      rows: [
        { label: 'Windows', cells: [{ v: 'n', tag: 'demand' }, 'y', { v: 'n', note: t('demo.compare.note.macosOnly') }, 'y', 'y', { v: 'y', note: t('demo.compare.note.server') }] },
        { label: t('demo.compare.row.nativeApp'), cells: [{ v: 'n', tag: 'design', note: t('demo.compare.note.webUiPurpose') }, { v: 'y', note: 'Electron' }, { v: 'y', note: t('demo.compare.note.native') }, { v: 'y', note: 'Electron' }, { v: 'n', note: t('demo.compare.note.terminal') }, { v: 'n', note: t('demo.compare.note.web') }] },
        { label: 'Kubernetes', cells: [{ v: 'n', tag: 'scope' }, 'y', 'y', 'y', 'n', 'y'] },
        { label: t('demo.compare.row.multiHost'), cells: [{ v: 'n', tag: 'design', note: t('demo.compare.note.singleOperator') }, 'n', 'n', 'n', 'n', 'y'] },
        { label: t('demo.compare.row.maturity'), cells: [{ v: t('demo.compare.cell.maturityNew'), note: t('demo.compare.note.smallMovingFast') }, { v: t('demo.compare.cell.maturityHuge'), note: t('demo.compare.note.industryDefault') }, { v: t('demo.compare.cell.maturityGrowing'), note: t('demo.compare.note.popularMac') }, { v: t('demo.compare.cell.maturityRedHat'), note: t('demo.compare.note.extensions') }, { v: t('demo.compare.cell.maturityOss'), note: t('demo.compare.note.bigFollowing') }, { v: t('demo.compare.cell.maturityMature'), note: t('demo.compare.note.largeEnterprise') }] },
      ],
    },
  ])

  const TAGS = $derived({
    design: { label: t('demo.compare.tag.design'), cls: 'design' },
    scope: { label: t('demo.compare.tag.scope'), cls: 'design' },
    demand: { label: t('demo.compare.tag.demand'), cls: 'demand' },
  })

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
      aria-label={t('demo.compare.dialogAria')}
      use:trapFocus
      in:scale={{ duration: 210, start: 0.97, opacity: 0, easing: cubicOut }}
      out:scale={{ duration: 130, start: 0.98, opacity: 0, easing: cubicOut }}
    >
      <header class="cmp-head">
        <div>
          <h2>{t('demo.compare.title')}</h2>
          <p>{t('demo.compare.subtitle')}</p>
        </div>
        <button class="cmp-x" aria-label={t('demo.compare.close')} onclick={close}>×</button>
      </header>

      <div class="cmp-scroll">
        <table>
          <thead>
            <tr>
              <th class="rowhead"></th>
              {#each tools as tool}<th class:me={tool === 'Oriel'}>{tool}</th>{/each}
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

      <p class="cmp-fn">{t('demo.compare.footnote')}</p>

      <footer class="cmp-foot">
        <div class="legend">
          <span class="tag design">{t('demo.compare.tag.design')}</span> {t('demo.compare.legend.design')}
          <span class="sep">·</span>
          <span class="tag demand">{t('demo.compare.tag.demand')}</span> {t('demo.compare.legend.demand')}
        </div>
        <div class="foot-actions">
          <a href="https://github.com/ParadoxInfinite/oriel/blob/main/ROADMAP.md" target="_blank" rel="noreferrer">{t('demo.compare.roadmapLink')}</a>
          <a href="https://github.com/ParadoxInfinite/oriel/issues" target="_blank" rel="noreferrer">{t('demo.compare.issueLink')}</a>
          <button class="cmp-done" onclick={close}>{t('demo.compare.done')}</button>
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
