// Command palette open/close state. The deterministic resolver today; the same
// surface gains an NL input mode when a provider plugin is configured.
export const palette = $state({ open: false })

export function openPalette() {
  palette.open = true
}
export function closePalette() {
  palette.open = false
}
export function togglePalette() {
  palette.open = !palette.open
}
