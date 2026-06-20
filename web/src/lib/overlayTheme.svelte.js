// The host renders the global overlays (command palette, confirm, toasts, op
// progress) outside any edition, so they'd otherwise always wear Classic's
// tokens. Each edition publishes an "overlay theme" here; the host applies it as
// CSS-variable overrides on the overlay wrapper so they match the active edition.
//
//   scheme 'classic' → no override (Classic's own @theme tokens)
//   scheme 'light' | 'dark' → Studio-style neutrals + the edition's accent
export const overlayTheme = $state({ scheme: 'classic', accent: '#2dd4bf' })

export function setOverlayTheme(scheme, accent) {
  overlayTheme.scheme = scheme
  if (accent) overlayTheme.accent = accent
}

const SCHEMES = {
  light: { bg: '#f6f7f9', surface: '#ffffff', surface2: '#f1f2f5', elevated: '#ffffff', border: '#e9eaee', borderLight: '#dfe1e6', fg: '#15181e', muted: '#545b68', faint: '#868d9a', accentFg: '#ffffff', danger: '#dc2626', ok: '#16a34a', warn: '#b45309' },
  dark: { bg: '#0e0f13', surface: '#16181d', surface2: '#1f222a', elevated: '#1b1d23', border: '#25272f', borderLight: '#33363f', fg: '#e9ebef', muted: '#9ea4b0', faint: '#6a717e', accentFg: '#0c0d11', danger: '#f06a6a', ok: '#36c97f', warn: '#e0a02a' },
}

// Build the inline `style` string that remaps Classic's --color-* tokens (which
// the overlays use) to the active edition's palette. '' = use Classic defaults.
export function overlayVars(t) {
  if (!t || t.scheme === 'classic') return ''
  const s = SCHEMES[t.scheme] || SCHEMES.dark
  return [
    `--color-bg:${s.bg}`,
    `--color-surface:${s.surface}`,
    `--color-surface-2:${s.surface2}`,
    `--color-elevated:${s.elevated}`,
    `--color-border:${s.border}`,
    `--color-border-light:${s.borderLight}`,
    `--color-fg:${s.fg}`,
    `--color-muted:${s.muted}`,
    `--color-faint:${s.faint}`,
    `--color-accent:${t.accent}`,
    `--color-accent-fg:${s.accentFg}`,
    `--color-danger:${s.danger}`,
    `--color-ok:${s.ok}`,
    `--color-warn:${s.warn}`,
    `color-scheme:${t.scheme}`,
  ].join(';')
}
