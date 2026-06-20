<script>
  let { state } = $props()

  // A small coloured pill per docker container state — adds at-a-glance colour.
  const pill = {
    running: 'text-ok bg-ok/10 ring-ok/25',
    paused: 'text-warn bg-warn/10 ring-warn/25',
    restarting: 'text-warn bg-warn/10 ring-warn/25',
    created: 'text-sky bg-sky/10 ring-sky/25',
    exited: 'text-muted bg-surface-2 ring-border',
    dead: 'text-danger bg-danger/10 ring-danger/25',
  }
  const dot = {
    running: 'bg-ok',
    paused: 'bg-warn',
    restarting: 'bg-warn',
    created: 'bg-sky',
    exited: 'bg-faint',
    dead: 'bg-danger',
  }
  const cls = $derived(pill[state] || 'text-muted bg-surface-2 ring-border')
  const dotc = $derived(dot[state] || 'bg-faint')
</script>

<span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-medium ring-1 ring-inset {cls}">
  <span class="h-1.5 w-1.5 rounded-full {dotc}"></span>
  {state}
</span>
