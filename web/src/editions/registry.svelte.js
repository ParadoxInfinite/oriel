import Classic from './Classic.svelte'
import Studio from './studio/Studio.svelte'
import { apiGet } from '../lib/api.js'

// Same base the API client uses, so theme imports resolve under ORIEL_BASE_PATH.
const BASE = import.meta.env.BASE_URL.replace(/\/$/, '')

// Built-in editions, in switcher order. Each `component` is a full-app Svelte
// component built on the platform SDK. Third parties add their own either by
// editing this array, or, without touching the bundle, by pushing manifests
// onto `window.__orielThemes` before mount (see editions/README.md).
const BUILTIN = [
  { id: 'studio', name: 'Studio', tagline: 'Clean, native-feel control panel', accent: '#5b5bd6', component: Studio },
  { id: 'classic', name: 'Classic', tagline: 'Calm teal control panel', accent: '#2dd4bf', component: Classic },
]

// Disk-installed themes: drop a built bundle (an ES module default-exporting
// { id, name, component, … }) into the server's themes directory and Oriel
// serves it same-origin. No remote URLs, install is explicit and offline.
export const diskThemes = $state({ dir: '', list: [], errors: {} })

// Discover + import every installed theme bundle. Call once at startup.
export async function loadDiskThemes() {
  let data
  try {
    data = await apiGet('/api/themes')
  } catch {
    return // backend unreachable, built-ins still work
  }
  diskThemes.dir = data.dir || ''
  for (const t of data.themes || []) {
    // A theme is JS imported into the app origin, so guard the import specifier:
    // basename ending in .js only (defense in depth, the backend already does).
    if (!t.file || !/^[\w.-]+\.js$/.test(t.file)) continue
    try {
      const mod = await import(/* @vite-ignore */ `${BASE}/api/themes/${t.file}`)
      const m = mod.default
      if (!m || !m.id || !m.component) {
        throw new Error('must default-export { id, name, component }')
      }
      const theme = { tagline: 'Installed theme', accent: '#5b5bd6', ...m, external: true }
      diskThemes.list = [...diskThemes.list.filter((x) => x.id !== m.id), theme]
      delete diskThemes.errors[t.file]
    } catch (e) {
      diskThemes.errors[t.file] = e.message
    }
  }
}

export function editions() {
  const runtime = (typeof window !== 'undefined' && window.__orielThemes) || []
  return [...BUILTIN, ...runtime, ...diskThemes.list]
}

const KEY = 'oriel.edition'
const stored = (typeof localStorage !== 'undefined' && localStorage.getItem(KEY)) || null

export const edition = $state({ active: stored ?? BUILTIN[0].id })

export function setEdition(id) {
  edition.active = id
  try {
    localStorage.setItem(KEY, id)
  } catch {
    /* private mode, selection just won't persist */
  }
}

export function activeEdition() {
  const all = editions()
  return all.find((e) => e.id === edition.active) ?? all[0]
}
