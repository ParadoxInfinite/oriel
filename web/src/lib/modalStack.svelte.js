// One global Escape handler so that, with overlays stacked, a single Escape
// closes only the top layer instead of every overlay's own window handler firing
// at once. Bubble phase (not capture) so an inner handler that stopPropagation's
// Escape, e.g. the pull dialog's suggestion list, still wins.
const stack = []
let listening = false

function onKeydown(e) {
  if (e.key !== 'Escape' || e.defaultPrevented || !stack.length) return
  e.preventDefault()
  e.stopPropagation()
  stack[stack.length - 1]()
}

// Pushes a close handler, returns an unregister fn. Wire to an overlay's lifetime:
// `$effect(() => registerEscape(onClose))`, or gated: `if (open) return registerEscape(close)`.
export function registerEscape(close) {
  if (!listening && typeof window !== 'undefined') {
    window.addEventListener('keydown', onKeydown)
    listening = true
  }
  stack.push(close)
  return () => {
    const i = stack.lastIndexOf(close)
    if (i >= 0) stack.splice(i, 1)
  }
}
