<script>
  import { onMount, onDestroy } from 'svelte'
  import { startLive, stopLive, connection } from './lib/live.svelte.js'
  import { refreshContainers } from './lib/containers.svelte.js'
  import { refreshStacks } from './lib/stacks.svelte.js'
  import { startUpdateChecks, stopUpdateChecks } from './lib/update.svelte.js'
  import { togglePalette } from './lib/palette.svelte.js'
  import { resumeOps } from './lib/op.svelte.js'

  import { activeEdition, loadDiskThemes } from './editions/registry.svelte.js'
  import { overlayTheme, overlayVars } from './lib/overlayTheme.svelte.js'
  import OpOverlay from './components/OpOverlay.svelte'
  import CommandPalette from './components/CommandPalette.svelte'
  import ConfirmDialog from './components/ConfirmDialog.svelte'
  import Toasts from './components/Toasts.svelte'
  import DemoBanner from './lib/demo/DemoBanner.svelte'

  // The host mounts one edition; the switcher swaps it live. Keying the render
  // on the id remounts cleanly so each edition starts from a fresh tree.
  const active = $derived(activeEdition())
  const Edition = $derived(active.component)
  // The active edition publishes how the global overlays should look (see
  // lib/overlayTheme); apply it as token overrides on the overlay wrapper.
  const overlayStyle = $derived(overlayVars(overlayTheme))

  function onKeydown(e) {
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
      e.preventDefault()
      togglePalette()
    }
  }

  onMount(() => {
    // One live stream feeds status/self/outages/stats/history; docker events drive
    // list refreshes. No polling loops.
    startLive()
    refreshContainers()
    refreshStacks()
    startUpdateChecks() // checks now, then re-checks every few hours while open
    loadDiskThemes()
    resumeOps() // re-attach to any prune still running from before a refresh
  })
  onDestroy(() => {
    stopLive()
    stopUpdateChecks()
  })
</script>

<svelte:window onkeydown={onKeydown} />

{#key active.id}
  <Edition />
{/key}

{#if !connection.ok}
  <div role="status" style="position:fixed;top:0;left:0;right:0;z-index:60;padding:5px 12px;text-align:center;font-size:12px;font-weight:500;background:#b45309;color:#fff;">
    Live connection lost — reconnecting…
  </div>
{/if}

<div style={overlayStyle}>
  <OpOverlay />
  <CommandPalette />
  <ConfirmDialog />
  <Toasts />
  {#if __ORIEL_DEMO__}
    <DemoBanner />
  {/if}
</div>
