// Command palette open/close state. A deterministic fuzzy matcher over the tool
// registry, every entry maps to a validated {tool, args} call.
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
