// Deterministic colour per compose project, so each stack reads as a group.
const PALETTE = [
  'var(--color-accent)',
  'var(--color-sky)',
  'var(--color-violet)',
  'var(--color-lime)',
  'var(--color-warn)',
  'var(--color-rose)',
]

export function stackColor(name) {
  if (!name) return 'var(--color-faint)'
  let h = 0
  for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) >>> 0
  return PALETTE[h % PALETTE.length]
}
