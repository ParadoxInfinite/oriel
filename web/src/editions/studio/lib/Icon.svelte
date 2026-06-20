<script>
  import { icons } from '../../../platform/index.js'
  let { name, size = 18, stroke = 1.75, class: cls = '' } = $props()
</script>

<svg
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill="none"
  stroke="currentColor"
  stroke-width={stroke}
  stroke-linecap="round"
  stroke-linejoin="round"
  class={cls}
  aria-hidden="true">{@html icons[name] ?? ''}</svg
>
