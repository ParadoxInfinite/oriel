// The host renders the global overlays (command palette, confirm, toasts, op
// progress) outside any edition, so without this they'd wear the base palette.
// Each edition publishes an "overlay theme" here; the host applies it as
// CSS-variable overrides on the overlay wrapper so they match the active edition.
//
//   scheme 'base' → no override (the global app.css @theme palette)
//   scheme 'light' | 'dark' → Studio-style neutrals + the edition's accent
export const overlayTheme = $state({ scheme: 'base', accent: '#2dd4bf' })

export function setOverlayTheme(scheme, accent) {
  overlayTheme.scheme = scheme
  if (accent) overlayTheme.accent = accent
}

const SCHEMES = {
  light: { bg: '#f6f7f9', surface: '#ffffff', surface2: '#f1f2f5', elevated: '#ffffff', border: '#e9eaee', borderLight: '#dfe1e6', fg: '#15181e', muted: '#545b68', faint: '#868d9a', accentFg: '#ffffff', danger: '#dc2626', ok: '#16a34a', warn: '#b45309' },
  dark: { bg: '#0e0f13', surface: '#16181d', surface2: '#1f222a', elevated: '#1b1d23', border: '#25272f', borderLight: '#33363f', fg: '#e9ebef', muted: '#9ea4b0', faint: '#6a717e', accentFg: '#0c0d11', danger: '#f06a6a', ok: '#36c97f', warn: '#e0a02a' },
}

// Overlay shape per scheme (radius + shadow), so shared overlays adopt each
// edition's modal language, not just its palette. The base scheme keeps the
// app.css look; light/dark match Studio's rounded-xl + --shadow-lg.
const SHAPE = {
  base: { radius: '0.625rem', shadow: '0 25px 50px -12px rgb(0 0 0 / 0.25)' },
  light: { radius: '0.75rem', shadow: '0 12px 32px -8px rgba(18,22,31,0.16)' },
  dark: { radius: '0.75rem', shadow: '0 18px 44px -14px rgba(0,0,0,0.7)' },
}

// Build the inline `style` string applied to the overlay wrapper: always the
// shape tokens, plus a full --color-* remap for the light/dark schemes.
export function overlayVars(t) {
  const scheme = t?.scheme || 'base'
  const shape = SHAPE[scheme] || SHAPE.base
  const out = [`--overlay-radius:${shape.radius}`, `--overlay-shadow:${shape.shadow}`]
  if (scheme === 'base') return out.join(';')

  const s = SCHEMES[scheme] || SCHEMES.dark
  out.push(
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
    `color-scheme:${scheme}`,
  )
  return out.join(';')
}
