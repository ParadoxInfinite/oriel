import { listDirs } from './discovery.svelte.js'

// Headless controller for the Radarr-style directory typeahead: debounced
// server-side dir listing + a navigable suggestion list. Editions render its
// reactive fields; the filesystem walk stays on the backend.
export class PathField {
  value = $state('')
  entries = $state([]) // full child paths under the typed directory
  open = $state(false)
  highlight = $state(-1)
  #debounce

  constructor(initial = '') {
    this.value = initial
  }

  input = () => {
    this.open = true
    this.highlight = -1
    clearTimeout(this.#debounce)
    this.#debounce = setTimeout(this.#load, 200)
  }

  focus = () => {
    this.open = true
    this.#load()
  }

  // Hide after a tick so a click on a suggestion still registers.
  blur = () => setTimeout(() => (this.open = false), 150)

  pick = (p) => {
    this.value = p + '/'
    this.highlight = -1
    this.#load()
  }

  reset = () => {
    this.value = ''
    this.entries = []
    this.open = false
  }

  keydown = (e) => {
    if (!this.open || !this.entries.length) return
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      this.highlight = Math.min(this.highlight + 1, this.entries.length - 1)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      this.highlight = Math.max(this.highlight - 1, 0)
    } else if (e.key === 'Enter' && this.highlight >= 0) {
      e.preventDefault()
      this.pick(this.entries[this.highlight])
    } else if (e.key === 'Escape') {
      e.stopPropagation()
      this.open = false
    }
  }

  #load = async () => {
    const res = await listDirs(this.value)
    this.entries = res.entries || []
  }
}

export const baseName = (p) => p.replace(/\/+$/, '').split('/').pop() || p
