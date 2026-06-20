// Global confirmation gate. Resolves true/false so call sites read:
// `if (!(await confirm({...}))) return`. When `checkbox` is given it instead
// resolves `{ ok, checked }` (e.g. for a "Don't ask again" option).
export const confirmState = $state({
  open: false,
  title: '',
  message: '',
  confirmLabel: 'Confirm',
  danger: true,
  checkbox: null, // label string, or null for the plain boolean form
  checked: false,
})

let resolver = null

export function confirm(opts = {}) {
  confirmState.title = opts.title ?? 'Are you sure?'
  confirmState.message = opts.message ?? ''
  confirmState.confirmLabel = opts.confirmLabel ?? 'Confirm'
  confirmState.danger = opts.danger ?? true
  confirmState.checkbox = opts.checkbox ?? null
  confirmState.checked = false
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
