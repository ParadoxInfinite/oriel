<script>
  import { tick } from 'svelte'
  import { LogsController } from '../lib/logs.svelte.js'

  let { container, onClose } = $props()

  const logs = new LogsController()
  let paused = $state(false)
  let scroller = $state(null)
  // Remembered for the browser session only (resets on a full reopen).
  let width = $state(Number(sessionStorage.getItem('cgui.logsWidth')) || 720)

  $effect(() => {
    logs.start(container.id)
    return () => logs.stop()
  })

  // Auto-scroll to bottom on new lines unless the user scrolled up.
  $effect(() => {
    void logs.lines.length
    if (!paused && scroller) {
      queueMicrotask(() => (scroller.scrollTop = scroller.scrollHeight))
    }
  })

  // Lazy-load older lines when near the top, preserving the viewport position
  // (content is added above, so shift scrollTop by the height gained).
  async function loadOlder() {
    if (!scroller) return
    const before = scroller.scrollHeight
    const prevTop = scroller.scrollTop
    const n = await logs.loadOlder()
    if (n > 0) {
      await tick()
      scroller.scrollTop = prevTop + (scroller.scrollHeight - before)
    }
  }

  function onScroll() {
    if (!scroller) return
    const nearBottom = scroller.scrollHeight - scroller.scrollTop - scroller.clientHeight < 40
    paused = !nearBottom
    logs.setFollowing(nearBottom)
    if (scroller.scrollTop < 60) loadOlder()
  }

  function startResize(e) {
    e.preventDefault()
    const move = (ev) => {
      width = Math.max(440, Math.min(window.innerWidth - 80, window.innerWidth - ev.clientX))
    }
    const up = () => {
      window.removeEventListener('mousemove', move)
      window.removeEventListener('mouseup', up)
      document.body.style.userSelect = ''
      sessionStorage.setItem('cgui.logsWidth', String(Math.round(width)))
    }
    document.body.style.userSelect = 'none'
    window.addEventListener('mousemove', move)
    window.addEventListener('mouseup', up)
  }

  function lineColor(stream) {
    return stream === 'stderr' ? 'text-warn' : stream === 'error' ? 'text-danger' : 'text-fg/85'
  }

  // Only close when a click both starts AND ends on the backdrop itself — so a
  // resize drag that releases outside the drawer never dismisses it.
  let pressedBackdrop = false
  function backdropDown(e) {
    pressedBackdrop = e.target === e.currentTarget
  }
  function backdropClick(e) {
    if (pressedBackdrop && e.target === e.currentTarget) onClose()
    pressedBackdrop = false
  }
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && onClose()} />

<div
  class="fixed inset-0 z-40 flex justify-end bg-black/40"
  role="presentation"
  onmousedown={backdropDown}
  onclick={backdropClick}
>
  <div
    class="relative flex h-full flex-col border-l border-border bg-bg shadow-2xl"
    style="width:{width}px; max-width:95vw"
    role="presentation"
    onclick={(e) => e.stopPropagation()}
  >
    <!-- drag handle to resize the drawer -->
    <div
      class="group absolute left-0 top-0 z-10 h-full w-2 -translate-x-1/2 cursor-col-resize"
      role="separator"
      aria-orientation="vertical"
      onmousedown={startResize}
    >
      <div class="mx-auto h-full w-px bg-border transition-colors group-hover:bg-accent"></div>
    </div>

    <div class="flex shrink-0 items-center justify-between border-b border-border px-5 py-3">
      <div class="min-w-0">
        <div class="truncate text-sm font-medium">{container.name}</div>
        <div class="truncate font-mono text-xs text-muted">{container.image}</div>
      </div>
      <div class="flex items-center gap-3">
        <div class="hidden items-center gap-2.5 text-[11px] text-muted sm:flex">
          <span class="inline-flex items-center gap-1"><span class="h-1.5 w-1.5 rounded-full bg-fg/60"></span>stdout</span>
          <span class="inline-flex items-center gap-1"><span class="h-1.5 w-1.5 rounded-full bg-warn"></span>stderr</span>
        </div>
        <button
          class="rounded-[--radius] px-2 py-1 text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg"
          onclick={onClose}>Close</button
        >
      </div>
    </div>

    <div
      bind:this={scroller}
      onscroll={onScroll}
      class="min-h-0 flex-1 overflow-auto bg-surface font-mono text-xs leading-relaxed"
    >
      {#if logs.loadingOlder}
        <div class="px-3 py-1.5 text-center text-[11px] text-faint">Loading older lines…</div>
      {:else if logs.noMore && logs.lines.length}
        <div class="px-3 py-1.5 text-center text-[11px] text-faint">Beginning of available logs</div>
      {/if}
      {#each logs.lines as l, i (i)}
        <div class="group flex gap-3 px-3 transition-colors hover:bg-accent/5 {i % 2 ? 'bg-white/[0.015]' : ''}">
          <span class="flex-1 whitespace-pre-wrap break-words py-px {lineColor(l.stream)}">{l.line}</span>
        </div>
      {/each}
      {#if logs.lines.length === 0}
        <div class="px-3 py-3 text-muted">
          {#if logs.error}{logs.error}
          {:else if !logs.connected}Connecting…
          {:else}No logs yet — this container hasn't written anything to stdout/stderr.{/if}
        </div>
      {/if}
    </div>

    {#if paused}
      <button
        class="shrink-0 border-t border-border bg-surface-2 py-1.5 text-center text-xs text-accent"
        onclick={() => {
          paused = false
          logs.setFollowing(true)
          scroller.scrollTop = scroller.scrollHeight
        }}>↓ Jump to latest</button
      >
    {/if}
  </div>
</div>
