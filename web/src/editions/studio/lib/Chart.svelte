<script>
  // Compact, scrubbable CPU history — adapted from Classic's Heartbeat, themed
  // for Studio. Time-axed so outages read correctly:
  //   • accent line  — colima up; peak CPU per 15s window
  //   • red baseline — colima was DOWN while recording
  //   • dashed grey  — Oriel itself was offline (a recording gap)
  // Points are { t, cpu(percent), down(bool) }; only the last 30 min is shown.
  let { points = [], height = 132 } = $props()

  const W = 1000
  const H = 200
  const WINDOW_MS = 15000
  const VIEW_MS = 30 * 60 * 1000

  let chartEl = $state(null)
  let hoverIdx = $state(-1)
  let hoverGap = $state(null)

  function niceCeil(x) {
    if (x <= 2) return 2
    if (x <= 5) return 5
    if (x <= 10) return 10
    const step = x <= 50 ? 5 : x <= 100 ? 10 : 25
    return Math.ceil(x / step) * step
  }

  // Clock-aligned 15s windows, each keeping the peak of its up-samples. A jump of
  // more than one window key means nothing was recorded — an offline gap.
  const series = $derived.by(() => {
    if (!points.length) return []
    const cutoff = Date.now() - VIEW_MS
    const groups = new Map()
    for (const p of points) {
      if (p.t < cutoff) continue
      const key = Math.floor(p.t / WINDOW_MS)
      let g = groups.get(key)
      if (!g) {
        g = { key, t: p.t, max: 0, up: 0 }
        groups.set(key, g)
      }
      g.t = p.t
      if (!p.down) {
        g.up++
        if (p.cpu > g.max) g.max = p.cpu
      }
    }
    return [...groups.values()].sort((a, b) => a.key - b.key).map((g) => ({ key: g.key, t: g.t, cpu: g.max, down: g.up === 0 }))
  })

  function spline(pts) {
    let d = `M ${pts[0].x.toFixed(1)},${pts[0].y.toFixed(1)}`
    for (let i = 0; i < pts.length - 1; i++) {
      const p0 = pts[i - 1] || pts[i]
      const p1 = pts[i]
      const p2 = pts[i + 1]
      const p3 = pts[i + 2] || p2
      const c1x = p1.x + (p2.x - p0.x) / 6
      const c1y = p1.y + (p2.y - p0.y) / 6
      const c2x = p2.x - (p3.x - p1.x) / 6
      const c2y = p2.y - (p3.y - p1.y) / 6
      d += ` C ${c1x.toFixed(1)},${c1y.toFixed(1)} ${c2x.toFixed(1)},${c2y.toFixed(1)} ${p2.x.toFixed(1)},${p2.y.toFixed(1)}`
    }
    return d
  }

  const geo = $derived.by(() => {
    const b = series
    if (b.length < 2) return null
    const peak = Math.max(...b.map((x) => x.cpu), 0)
    const top = niceCeil(peak * 1.05)
    const range = top || 1
    const tMin = b[0].t
    const span = Math.max(1, b[b.length - 1].t - tMin)
    const baseY = H - 6
    const X = (t) => ((t - tMin) / span) * W
    const Y = (v) => H - Math.max(0, Math.min(v / range, 1)) * (H - 12) - 6
    const pts = b.map((x) => ({ x: X(x.t), y: x.down ? baseY : Y(x.cpu), cpu: x.cpu, down: x.down, t: x.t }))

    const upSegs = []
    const downSpans = []
    const gaps = []
    let seg = []
    let down = null
    for (let i = 0; i < pts.length; i++) {
      const p = pts[i]
      if (i > 0 && b[i].key - b[i - 1].key > 1) {
        if (seg.length) (upSegs.push(seg), (seg = []))
        if (down) (downSpans.push(down), (down = null))
        gaps.push({ x1: pts[i - 1].x, x2: p.x, t1: pts[i - 1].t, t2: p.t })
      }
      if (p.down) {
        if (seg.length) (upSegs.push(seg), (seg = []))
        if (down) (down.x2 = p.x), (down.t2 = p.t)
        else {
          const prev = pts[i - 1]
          down = { x1: p.x, x2: p.x, t1: prev && !prev.down ? prev.t : p.t, t2: p.t }
        }
      } else {
        if (down) (down.t2 = p.t), downSpans.push(down), (down = null)
        seg.push(p)
      }
    }
    if (seg.length) upSegs.push(seg)
    if (down) downSpans.push(down)

    const paths = upSegs
      .filter((s) => s.length >= 2)
      .map((s) => {
        const d = spline(s)
        return { d, area: `${d} L ${s[s.length - 1].x.toFixed(1)},${baseY} L ${s[0].x.toFixed(1)},${baseY} Z` }
      })
    const dots = upSegs.filter((s) => s.length === 1).map((s) => s[0])
    return { pts, paths, dots, downSpans, gaps, top, baseY }
  })

  // Percentage gridlines at ceiling, midpoint, zero.
  const grid = $derived.by(() => {
    if (!geo) return []
    const top = geo.top
    return [top, top / 2, 0].map((v) => ({ label: `${Math.round(v)}%`, topPct: ((H - (v / (top || 1)) * (H - 12) - 6) / H) * 100 }))
  })

  const n = $derived(geo ? geo.pts.length : 0)
  const active = $derived(hoverIdx >= 0 && hoverIdx < n ? hoverIdx : n - 1)
  const hovering = $derived(hoverIdx >= 0 && hoverIdx < n)
  const cur = $derived(geo && active >= 0 ? geo.pts[active] : null)
  const curSpan = $derived.by(() => {
    if (!geo || !cur || !cur.down) return null
    return geo.downSpans.find((s) => cur.x >= s.x1 && cur.x <= s.x2) || null
  })

  const pctX = (x) => (x / W) * 100
  const pctY = (y) => (y / H) * 100

  function onMove(e) {
    if (!geo || !chartEl) return
    const r = chartEl.getBoundingClientRect()
    const mx = ((e.clientX - r.left) / r.width) * W
    const g = geo.gaps.find((gp) => mx > gp.x1 && mx < gp.x2)
    if (g) {
      hoverGap = { ...g, mx }
      hoverIdx = -1
      return
    }
    hoverGap = null
    let best = 0
    let bd = Infinity
    for (let i = 0; i < geo.pts.length; i++) {
      const d = Math.abs(geo.pts[i].x - mx)
      if (d < bd) {
        bd = d
        best = i
      }
    }
    hoverIdx = best
  }
  function leave() {
    hoverIdx = -1
    hoverGap = null
  }

  const clock = (t) => new Date(t).toTimeString().slice(0, 8)
  function fmtDur(ms) {
    const s = Math.round(ms / 1000)
    return s < 60 ? `${s}s` : `${Math.floor(s / 60)}m ${s % 60}s`
  }
  function ago(t) {
    const m = Math.round((Date.now() - t) / 60000)
    return m <= 0 ? 'now' : `${m}m ago`
  }
</script>

<div class="flex flex-col" style="height:{height}px">
  <div bind:this={chartEl} class="relative min-h-0 flex-1" role="presentation" onmousemove={onMove} onmouseleave={leave}>
    <svg viewBox="0 0 {W} {H}" preserveAspectRatio="none" class="h-full w-full" style="display:block">
      <defs>
        <linearGradient id="studio-chart-fill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="var(--accent)" stop-opacity="0.2" />
          <stop offset="100%" stop-color="var(--accent)" stop-opacity="0" />
        </linearGradient>
      </defs>
      {#if geo}
        {#each geo.gaps as g}
          <line x1={g.x1} y1={geo.baseY} x2={g.x2} y2={geo.baseY} stroke="var(--text-3)" stroke-width="1.5" stroke-dasharray="2 5" vector-effect="non-scaling-stroke" />
        {/each}
        {#each geo.paths as p}
          <path d={p.area} fill="url(#studio-chart-fill)" stroke="none" />
          <path d={p.d} fill="none" stroke="var(--accent)" stroke-width="2" stroke-linejoin="round" stroke-linecap="round" vector-effect="non-scaling-stroke" />
        {/each}
        {#each geo.downSpans as d}
          <line x1={d.x1} y1={geo.baseY} x2={d.x2 === d.x1 ? d.x1 + 6 : d.x2} y2={geo.baseY} stroke="var(--red)" stroke-width="3" stroke-linecap="round" vector-effect="non-scaling-stroke" />
        {/each}
        {#each geo.dots as d}
          <circle cx={d.x} cy={d.y} r="2.5" fill="var(--accent)" />
        {/each}
      {/if}
    </svg>

    {#if geo}
      {#each grid as g, i}
        <span class="pointer-events-none absolute left-0 right-0 border-t border-[var(--border)]" style="top:{g.topPct}%"></span>
        <span class="mono pointer-events-none absolute left-0 -translate-y-1/2 bg-[var(--panel)] pr-1 text-[9px] text-[var(--text-3)] {i === grid.length - 1 ? 'translate-y-[-3px]' : ''}" style="top:{g.topPct}%">{g.label}</span>
      {/each}

      {#if hovering && cur}
        <span class="pointer-events-none absolute bottom-0 top-0 w-px bg-[var(--border-strong)]" style="left:{pctX(cur.x)}%"></span>
      {/if}

      {#if hoverGap}
        <span class="pointer-events-none absolute bottom-0 top-0 w-px bg-[var(--border-strong)]" style="left:{pctX(hoverGap.mx)}%"></span>
        <div class="pointer-events-none absolute z-10 -translate-x-1/2 -translate-y-1/2 whitespace-nowrap rounded-lg border border-[var(--border)] bg-[var(--panel)] px-2 py-1 text-center shadow-[var(--shadow-md)]" style="left:{Math.min(88, Math.max(12, pctX(hoverGap.mx)))}%; top:40%">
          <div class="mono text-[11px] text-[var(--text-2)]">Oriel offline</div>
          <div class="mono text-[10px] text-[var(--text-3)]">{fmtDur(hoverGap.t2 - hoverGap.t1)} · no data</div>
        </div>
      {/if}

      {#if cur}
        <span class="studio-cdot pointer-events-none absolute h-2.5 w-2.5 rounded-full" style="left:{pctX(cur.x)}%; top:{pctY(cur.y)}%; background:{cur.down ? 'var(--red)' : 'var(--accent)'}; box-shadow:0 0 0 3px color-mix(in srgb, {cur.down ? 'var(--red)' : 'var(--accent)'} 22%, transparent); transform:translate(-50%,-50%)"></span>
      {/if}

      {#if hovering && cur}
        <div class="pointer-events-none absolute z-10 -translate-x-1/2 -translate-y-full whitespace-nowrap rounded-lg border border-[var(--border)] bg-[var(--panel)] px-2 py-1 text-center shadow-[var(--shadow-md)]" style="left:{Math.min(92, Math.max(8, pctX(cur.x)))}%; top:{pctY(cur.y)}%; margin-top:-10px">
          {#if cur.down}
            <div class="mono text-[11px] text-[var(--red)]">colima down</div>
            <div class="mono text-[10px] text-[var(--text-3)]">{curSpan ? `${fmtDur(curSpan.t2 - curSpan.t1)} · unreachable` : clock(cur.t)}</div>
          {:else}
            <div class="mono tnum text-[12px] font-medium text-[var(--text)]">{cur.cpu.toFixed(1)}%</div>
            <div class="mono text-[10px] text-[var(--text-3)]">{clock(cur.t)}</div>
          {/if}
        </div>
      {/if}
    {/if}
  </div>

  {#if n > 1}
    <div class="mono mt-1.5 flex justify-between text-[10px] text-[var(--text-3)]">
      <span>{ago(geo.pts[0].t)}</span>
      <span>{hovering && cur ? clock(cur.t) : 'now'}</span>
    </div>
  {/if}
</div>

<style>
  .studio-cdot {
    animation: studio-cdot 1.4s ease-in-out infinite;
  }
  @keyframes studio-cdot {
    0%,
    100% {
      opacity: 0.7;
    }
    50% {
      opacity: 1;
    }
  }
</style>
