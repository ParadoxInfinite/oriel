import { streamPost } from './api.js'
import { refreshImages } from './resources.svelte.js'
import { REGISTRY_SOURCES, REGISTRY_HOSTS, searchRegistry, listImageTags } from './registry.js'

// Headless controller for the image-pull dialog: registry selection, live search,
// tag suggestions, and the streaming pull, all the behaviour, none of the look.
// Editions construct one and render its reactive state however they like, so the
// Studio and Classic pull dialogs share a single source of truth.
export class PullController {
  sourceId = $state('dockerhub')
  ref = $state('')
  pulling = $state(false)
  status = $state('')
  error = $state(null)
  done = $state(false)

  results = $state([])
  tags = $state([])
  view = $state(null) // 'search' | 'tags' | null
  busy = $state(false)
  highlight = $state(-1)

  #debounce
  #reqId = 0
  #tagsRepo = ''

  constructor(initial = '') {
    this.ref = initial
  }

  // ── Derived ────────────────────────────────────────────────────────────────
  get sources() {
    return REGISTRY_SOURCES
  }
  get source() {
    return REGISTRY_SOURCES.find((s) => s.id === this.sourceId)
  }
  get repoBase() {
    return this.ref.trim().replace(/:[\w.-]*$/, '')
  }
  get tagQuery() {
    return this.ref.includes(':') ? this.ref.slice(this.ref.lastIndexOf(':') + 1) : ''
  }
  get shownTags() {
    return this.tagQuery ? this.tags.filter((t) => t.includes(this.tagQuery)) : this.tags
  }
  // The list the keyboard navigates: tags when picking a tag, else search hits.
  get activeList() {
    return this.view === 'tags' ? this.shownTags : this.view === 'search' ? this.results : []
  }

  // ── Handlers ─────────────────────────────────────────────────────────────────
  onInput = () => {
    this.error = null
    clearTimeout(this.#debounce)
    this.highlight = -1
    const tagMode = this.ref.includes(':') && this.source.tags && this.repoBase.length >= 2
    if (tagMode) {
      this.view = 'tags'
      this.busy = true
      this.#debounce = setTimeout(() => this.#loadTags(this.repoBase), 250)
    } else if (this.source.search && this.repoBase.length >= 2) {
      this.view = 'search'
      this.busy = true
      this.#debounce = setTimeout(() => this.#runSearch(this.repoBase), 250)
    } else {
      // Nothing will fire; clearTimeout above cancelled any pending request, so
      // clear the spinner here or it sticks on after a debounced search is dropped.
      this.view = null
      this.results = []
      this.busy = false
    }
  }

  // destroy clears the pending debounce so a dialog closed mid-type doesn't fire
  // a wasted registry fetch (and mutate state on a detached controller).
  destroy() {
    clearTimeout(this.#debounce)
  }

  onSourceChange = () => {
    this.results = []
    this.tags = []
    this.#tagsRepo = ''
    this.view = null
    this.highlight = -1
    this.error = null
    this.busy = false
    const cur = this.ref.trim()
    if (!this.source.search && (cur === '' || REGISTRY_HOSTS.includes(cur))) this.ref = this.source.host
    else if (this.source.search && REGISTRY_HOSTS.includes(cur)) this.ref = ''
    else this.onInput()
  }

  onKeydown = (e) => {
    const list = this.activeList
    if (list.length) {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        this.highlight = Math.min(this.highlight + 1, list.length - 1)
        return
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        this.highlight = Math.max(this.highlight - 1, 0)
        return
      }
      if (e.key === 'Enter' && this.highlight >= 0) {
        e.preventDefault()
        this.view === 'tags' ? this.pickTag(list[this.highlight]) : this.pickRepo(list[this.highlight])
        return
      }
      if (e.key === 'Escape') {
        e.stopPropagation()
        this.view = null
        return
      }
    }
    if (e.key === 'Enter') this.pull()
  }

  pickRepo = (r) => {
    this.ref = r.name
    this.results = []
    if (this.source.tags) {
      this.view = 'tags'
      this.#loadTags(r.name)
    } else {
      this.view = null
    }
  }

  pickTag = (t) => {
    this.ref = `${this.repoBase}:${t}`
    this.view = null
  }

  pull = async () => {
    if (!this.ref.trim() || this.pulling) return
    this.view = null
    this.pulling = true
    this.error = null
    this.done = false
    this.status = 'Starting…'
    try {
      await streamPost(`/api/images/pull?ref=${encodeURIComponent(this.ref.trim())}`, {
        onEvent: (name, data) => {
          if (name === 'progress') this.status = data.id ? `${data.status} ${data.id}` : data.status
          else if (name === 'done') {
            this.done = true
            if (!data.ok) this.error = data.error
            else this.status = 'Done.'
          }
        },
      })
    } catch (e) {
      this.error = e.message
      this.done = true
    } finally {
      this.pulling = false
      refreshImages()
    }
  }

  // ── Private ──────────────────────────────────────────────────────────────────
  #runSearch = async (term) => {
    const id = ++this.#reqId
    try {
      const list = await searchRegistry(this.sourceId, term)
      if (id !== this.#reqId) return
      this.results = list.sort((a, b) => (b.official ? 1 : 0) - (a.official ? 1 : 0) || b.stars - a.stars)
    } catch {
      if (id === this.#reqId) this.results = []
    } finally {
      if (id === this.#reqId) this.busy = false
    }
  }

  #loadTags = async (repo) => {
    if (!this.source.tags || repo === this.#tagsRepo) {
      this.busy = false // nothing to fetch; don't leave the spinner stuck
      return
    }
    const id = ++this.#reqId
    this.busy = true
    try {
      const list = await listImageTags(this.sourceId, repo)
      if (id !== this.#reqId) return
      this.tags = list
      this.#tagsRepo = repo
    } catch {
      if (id === this.#reqId) this.tags = []
    } finally {
      if (id === this.#reqId) this.busy = false
    }
  }
}

// Star count → compact label (1234 → "1.2k").
export const fmtStars = (n) => (n >= 1000 ? `${(n / 1000).toFixed(n >= 10000 ? 0 : 1)}k` : String(n))
