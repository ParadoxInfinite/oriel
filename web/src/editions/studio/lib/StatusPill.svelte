<script>
  // Maps a docker container/compose state to a labelled status pill.
  let { state } = $props()

  const MAP = {
    running: { cls: 'on', label: 'Running' },
    exited: { cls: 'off', label: 'Exited' },
    created: { cls: 'off', label: 'Created' },
    paused: { cls: 'warn', label: 'Paused' },
    restarting: { cls: 'warn', label: 'Restarting' },
    removing: { cls: 'warn', label: 'Removing' },
    dead: { cls: 'bad', label: 'Dead' },
  }
  const m = $derived(MAP[state] ?? { cls: 'off', label: state || 'Unknown' })
</script>

<span class="pill {m.cls}"><span class="dot"></span>{m.label}</span>
