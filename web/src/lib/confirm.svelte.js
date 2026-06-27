// Global confirmation gate. Resolves true/false so call sites read:
// `if (!(await confirm({...}))) return`. When `checkbox` is given it instead
// resolves `{ ok, checked }` (e.g. for a "Don't ask again" option).
export const confirmState = $state({
  open: false,
  title: '',
  message: '',
  confirmLabel: 'Confirm',
  tone: 'danger', // 'danger' (red) | 'warn' (amber) | 'accent'; drives icon + button color
  checkbox: null, // label string, or null for the plain boolean form
  checked: false,
})

let resolver = null

export function confirm(opts = {}) {
  // Cancel any still-pending confirm so its awaiter resolves (false) instead of
  // hanging forever when we overwrite the single resolver.
  if (resolver) {
    const prev = resolver
    resolver = null
    prev(false)
  }
  confirmState.title = opts.title ?? 'Are you sure?'
  confirmState.message = opts.message ?? ''
  confirmState.confirmLabel = opts.confirmLabel ?? 'Confirm'
  // `tone` is the source of truth; `danger:false` stays supported as a shorthand
  // for the neutral accent tone (back-compat with existing call sites).
  confirmState.tone = opts.tone ?? (opts.danger === false ? 'accent' : 'danger')
  confirmState.checkbox = opts.checkbox ?? null
  confirmState.checked = opts.checked ?? false
  confirmState.open = true
  return new Promise((resolve) => {
    resolver = resolve
  })
}

export function resolveConfirm(ok) {
  if (!confirmState.open) return
  confirmState.open = false
  const r = resolver
  resolver = null
  if (r) r(confirmState.checkbox ? { ok, checked: confirmState.checked } : ok)
}
