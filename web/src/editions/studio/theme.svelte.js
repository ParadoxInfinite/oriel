// Studio's own appearance: light/dark/system + a per-theme accent colour.
// Persisted locally and independent of the edition registry — this only governs
// how Studio looks, not which edition is mounted. Light and dark each remember
// their own accent (indigo for light, teal for dark by default), so switching
// theme swaps the accent too.

const KEY = 'oriel.appearance'

// Default accent per base theme.
const DEFAULT_ACCENTS = { light: '#5b5bd6', dark: '#0d9488' } // indigo / teal

// Built-in accent presets. `add` lets users append their own (a custom theme).
export const ACCENTS = [
  { id: 'indigo', name: 'Indigo', value: '#5b5bd6' },
  { id: 'violet', name: 'Violet', value: '#7c5cff' },
  { id: 'blue', name: 'Blue', value: '#2563eb' },
  { id: 'teal', name: 'Teal', value: '#0d9488' },
  { id: 'emerald', name: 'Emerald', value: '#059669' },
  { id: 'amber', name: 'Amber', value: '#d97706' },
  { id: 'rose', name: 'Rose', value: '#e11d48' },
  { id: 'slate', name: 'Graphite', value: '#475569' },
]

function load() {
  try {
    return JSON.parse(localStorage.getItem(KEY)) || {}
  } catch {
    return {}
  }
}
const saved = load()

export const appearance = $state({
  mode: saved.mode || 'system', // 'light' | 'dark' | 'system'
  // Migrate an older single `accent` to both bases; otherwise use saved/defaults.
  accents: saved.accents ?? (saved.accent ? { light: saved.accent, dark: saved.accent } : { ...DEFAULT_ACCENTS }),
  custom: Array.isArray(saved.custom) ? saved.custom : [], // [{ id, name, value }]
})

// Tracks the OS colour scheme so `mode: 'system'` stays reactive.
export const systemPref = $state({ dark: false })

let mql
export function initAppearance() {
  if (mql || typeof window === 'undefined' || !window.matchMedia) return
  mql = window.matchMedia('(prefers-color-scheme: dark)')
  systemPref.dark = mql.matches
  mql.addEventListener('change', (e) => (systemPref.dark = e.matches))
}

// The effective light/dark base, resolving 'system' against the OS preference.
export function resolvedBase() {
  return appearance.mode === 'system' ? (systemPref.dark ? 'dark' : 'light') : appearance.mode
}

function persist() {
  try {
    localStorage.setItem(KEY, JSON.stringify({ mode: appearance.mode, accents: appearance.accents, custom: appearance.custom }))
  } catch {
    /* private mode — just won't persist */
  }
}

export function setMode(mode) {
  appearance.mode = mode
  persist()
}
// Accent edits apply to whichever base is currently showing.
export function setAccent(value) {
  appearance.accents = { ...appearance.accents, [resolvedBase()]: value }
  persist()
}
export function addCustomAccent(name, value) {
  const id = 'custom-' + Math.random().toString(36).slice(2, 8)
  appearance.custom = [...appearance.custom, { id, name: name?.trim() || value, value }]
  setAccent(value)
}
export function removeCustomAccent(id) {
  appearance.custom = appearance.custom.filter((c) => c.id !== id)
  persist()
}
