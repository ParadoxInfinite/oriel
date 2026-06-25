// Shared navigation intent. The palette is a global overlay above the one active
// edition, so "go to this view / open this entity" can't be a theme's private
// state, it lives here and every theme reads it. The theme contract (bind to
// nav.view; optionally honour a target) is documented in docs/THEMES.md.
export const VIEWS = ['Dashboard', 'Containers', 'Images', 'Volumes', 'Networks', 'Stacks', 'Settings']

export const nav = $state({ view: 'Dashboard', target: null })

// Switch the active view, optionally carrying a deep-link intent for the
// destination to claim, by convention { kind, ...payload, open? }.
export function navigate(view, target = null) {
  nav.view = view
  nav.target = target
}

// A view claims and clears any pending intent addressed to it. Reading nav here
// makes the caller's $effect re-run on re-navigation, so it fires each time.
export function takeTarget(view) {
  if (nav.view !== view || !nav.target) return null
  const t = nav.target
  nav.target = null
  return t
}
