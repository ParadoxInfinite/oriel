// `use:trapFocus` on a modal: focuses in on mount, cycles Tab within, restores
// focus to the prior element on destroy. Node needs tabindex="-1".

const SELECTOR =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

export function trapFocus(node) {
  const restore = document.activeElement instanceof HTMLElement ? document.activeElement : null

  const focusables = () =>
    [...node.querySelectorAll(SELECTOR)].filter((el) => el.offsetParent !== null || el === document.activeElement)

  // Move focus inside once the content has rendered.
  queueMicrotask(() => {
    const f = focusables()
    ;(f[0] ?? node).focus?.({ preventScroll: true })
  })

  function onKeydown(e) {
    if (e.key !== 'Tab') return
    const f = focusables()
    if (!f.length) {
      e.preventDefault()
      node.focus?.()
      return
    }
    const first = f[0]
    const last = f[f.length - 1]
    const active = document.activeElement
    if (e.shiftKey && (active === first || active === node)) {
      e.preventDefault()
      last.focus()
    } else if (!e.shiftKey && active === last) {
      e.preventDefault()
      first.focus()
    }
  }

  node.addEventListener('keydown', onKeydown)
  return {
    destroy() {
      node.removeEventListener('keydown', onKeydown)
      restore?.focus?.({ preventScroll: true })
    },
  }
}
