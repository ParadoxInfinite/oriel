<script>
  // An interactive shell into a running container. xterm.js is fetched on demand
  // from the backend (served same-origin, never bundled), and the terminal talks
  // to /api/containers/{id}/shell over a WebSocket: binary frames carry the PTY
  // stream both ways, a text frame carries a {resize} control message.
  import { onMount } from 'svelte'
  import { t } from '../../../platform/index.js'

  let { container } = $props()

  let host = $state(null)
  let status = $state('loading') // 'loading' | 'open' | 'closed' | 'error'
  let errMsg = $state('')

  const BASE = import.meta.env.BASE_URL.replace(/\/$/, '')
  const termUrl = (file) => `${BASE}/api/term/${file}`
  const wsUrl = () => `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}${BASE}/api/containers/${container.id}/shell`

  let term, ws, ro
  const enc = new TextEncoder()

  async function loadTerminal() {
    if (!document.querySelector('link[data-xterm]')) {
      const link = document.createElement('link')
      link.rel = 'stylesheet'
      link.href = termUrl('xterm.css')
      link.dataset.xterm = ''
      document.head.appendChild(link)
    }
    const mod = await import(/* @vite-ignore */ termUrl('xterm.js'))
    return mod.Terminal
  }

  // xterm exposes the rendered cell size on its render service; we pin the xterm
  // version, so reading it to size the grid is stable (it's what the fit addon
  // does). If it's not ready, we leave the default size and refit on resize.
  function fit() {
    const cell = term?._core?._renderService?.dimensions?.css?.cell
    if (!cell?.width || !cell?.height || !host?.clientWidth || !host?.clientHeight) return
    const cols = Math.max(2, Math.floor((host.clientWidth - 8) / cell.width))
    const rows = Math.max(1, Math.floor((host.clientHeight - 8) / cell.height))
    if (cols === term.cols && rows === term.rows) return // no-op: don't spam the PTY
    term.resize(cols, rows)
    if (ws?.readyState === 1) ws.send(JSON.stringify({ resize: { cols, rows } }))
  }

  // Only hand xterm a colour it can parse; a theme may define a token as a
  // color-mix(...) expression, which would otherwise throw on construction.
  function themeColors() {
    const cs = getComputedStyle(document.documentElement)
    const safe = (n, fb) => {
      const v = cs.getPropertyValue(n).trim()
      return /^#[0-9a-f]{3,8}$|^rgb/i.test(v) ? v : fb
    }
    return { background: safe('--panel-2', '#16161a'), foreground: safe('--text', '#e7e7ea'), cursor: safe('--accent', '#6ea8fe') }
  }

  onMount(() => {
    let disposed = false
    ;(async () => {
      let Terminal
      try {
        Terminal = await loadTerminal()
      } catch (e) {
        status = 'error'
        errMsg = e?.message || 'could not load the terminal'
        return
      }
      if (disposed) return
      term = new Terminal({
        cursorBlink: true,
        fontSize: 13,
        fontFamily: "'Geist Mono Variable', ui-monospace, SFMono-Regular, Menlo, monospace",
        theme: themeColors(),
        scrollback: 5000,
      })
      term.open(host)
      term.focus()

      ws = new WebSocket(wsUrl())
      ws.binaryType = 'arraybuffer'
      ws.onopen = () => { status = 'open'; fit() }
      ws.onmessage = (e) => { if (!disposed && term) term.write(new Uint8Array(e.data)) }
      ws.onclose = () => { if (status !== 'error') status = 'closed' }
      ws.onerror = () => { status = 'error'; errMsg = 'connection failed' }
      term.onData((d) => { if (ws?.readyState === 1) ws.send(enc.encode(d)) })

      ro = new ResizeObserver(() => fit())
      ro.observe(host)
      // The cell metrics aren't ready until the (custom) font has loaded, so the
      // first fit() can no-op; refit once fonts settle and on the next frame.
      document.fonts?.ready?.then(() => !disposed && fit())
      requestAnimationFrame(() => !disposed && fit())
    })()

    return () => {
      disposed = true
      ro?.disconnect()
      try { ws?.close() } catch {}
      term?.dispose()
    }
  })
</script>

<div class="relative min-h-0 flex-1 bg-[var(--panel-2)]">
  <div bind:this={host} class="absolute inset-0 p-2"></div>
  {#if status === 'loading'}
    <div class="absolute inset-0 flex items-center gap-2 p-4 text-sm text-[var(--text-3)]">
      <span class="h-4 w-4 animate-spin rounded-full border-2 border-[var(--border-strong)] border-t-[var(--accent)]"></span>
      {t('shell.loading')}
    </div>
  {:else if status === 'error'}
    <div class="absolute inset-x-0 top-0 m-3 rounded-lg border border-[color-mix(in_srgb,var(--red)_40%,var(--border))] bg-[var(--panel)] p-3 text-sm text-[var(--red)]">
      {t('shell.error')}{errMsg ? ` · ${errMsg}` : ''}
    </div>
  {/if}
</div>
