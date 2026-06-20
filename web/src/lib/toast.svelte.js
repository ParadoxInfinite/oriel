// Minimal transient notifications.
export const toasts = $state({ items: [] })

let nextId = 0
export function toast(message, kind = 'info') {
  const t = { id: ++nextId, message, kind }
  toasts.items = [...toasts.items, t]
  setTimeout(() => dismissToast(t.id), 4000)
}
export function dismissToast(id) {
  toasts.items = toasts.items.filter((t) => t.id !== id)
}
