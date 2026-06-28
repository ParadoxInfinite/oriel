<script>
  // Maps a docker container/compose state to a labelled status pill.
  import { t } from '../../../platform/index.js'

  let { state } = $props()

  const MAP = $derived({
    running: { cls: 'on', label: t('status.running') },
    exited: { cls: 'off', label: t('status.exited') },
    created: { cls: 'off', label: t('status.created') },
    paused: { cls: 'warn', label: t('status.paused') },
    restarting: { cls: 'warn', label: t('status.restarting') },
    removing: { cls: 'warn', label: t('status.removing') },
    dead: { cls: 'bad', label: t('status.dead') },
  })
  const m = $derived(MAP[state] ?? { cls: 'off', label: state || t('status.unknown') })
</script>

<span class="pill {m.cls}"><span class="dot"></span>{m.label}</span>
