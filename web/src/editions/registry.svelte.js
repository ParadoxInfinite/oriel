import Classic from './Classic.svelte'
import Studio from './studio/Studio.svelte'

// Built-in editions, in switcher order. Each `component` is a full-app Svelte
// component built on the platform SDK. Third parties add their own either by
// editing this array, or — without touching the bundle — by pushing manifests
// onto `window.__orielThemes` before mount (see editions/README.md).
const BUILTIN = [
  { id: 'studio', name: 'Studio', tagline: 'Clean, native-feel control panel', accent: '#5b5bd6', component: Studio },
  { id: 'classic', name: 'Classic', tagline: 'Calm teal control panel', accent: '#2dd4bf', component: Classic },
]

// External theme bundles, loaded by URL at runtime (Settings → Themes). Each
// must be an ES module default-exporting a manifest { id, name, component, … }.
// URLs are persisted; the modules are re-imported on the next load.
const EXT_KEY = 'oriel.externalThemes'
function loadExtUrls() {
  try {
    return JSON.parse(localStorage.getItem(EXT_KEY)) || []
  } catch {
    return []
  }
}
export const externalThemes = $state({ urls: loadExtUrls(), list: [], errors: {} })

function persistExtUrls() {
  try {
    localStorage.setItem(EXT_KEY, JSON.stringify(externalThemes.urls))
  } catch {
    /* ignore */
  }
}

async function importTheme(url) {
  const mod = await import(/* @vite-ignore */ url)
  const m = mod.default
  if (!m || !m.id || !m.component) {
    throw new Error('Theme must default-export { id, name, component }')
  }
  return { tagline: 'External theme', accent: '#5b5bd6', ...m, _url: url, external: true }
}

// Import every persisted external theme. Call once at startup.
export async function loadExternalThemes() {
  for (const url of externalThemes.urls) {
    try {
      const m = await importTheme(url)
      externalThemes.list = [...externalThemes.list.filter((x) => x.id !== m.id), m]
      delete externalThemes.errors[url]
    } catch (e) {
      externalThemes.errors[url] = e.message
    }
  }
}

// Add (and immediately load) a theme by URL.
export async function addExternalTheme(url) {
  url = url.trim()
  if (!url) throw new Error('Enter a URL')
  const m = await importTheme(url) // validate before persisting
  if (!externalThemes.urls.includes(url)) {
    externalThemes.urls = [...externalThemes.urls, url]
    persistExtUrls()
  }
  externalThemes.list = [...externalThemes.list.filter((x) => x.id !== m.id), m]
  return m
}

export function removeExternalTheme(url) {
  externalThemes.urls = externalThemes.urls.filter((u) => u !== url)
  externalThemes.list = externalThemes.list.filter((x) => x._url !== url)
  delete externalThemes.errors[url]
  persistExtUrls()
}

export function editions() {
  const runtime = (typeof window !== 'undefined' && window.__orielThemes) || []
  return [...BUILTIN, ...runtime, ...externalThemes.list]
}

const KEY = 'oriel.edition'
const stored = (typeof localStorage !== 'undefined' && localStorage.getItem(KEY)) || null

export const edition = $state({ active: stored ?? BUILTIN[0].id })

export function setEdition(id) {
  edition.active = id
  try {
    localStorage.setItem(KEY, id)
  } catch {
    /* private mode — selection just won't persist */
  }
}

export function activeEdition() {
  const all = editions()
  return all.find((e) => e.id === edition.active) ?? all[0]
}
