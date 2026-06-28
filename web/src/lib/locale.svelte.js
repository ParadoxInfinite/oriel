// Translation layer. English is bundled and is always the fallback, so a missing
// key in any other catalog shows English rather than breaking. The chosen locale
// is a per-browser preference, persisted to localStorage like the theme. Other
// locales are fetched through the backend (/api/i18n) so they can be published
// and updated without shipping a new binary.
import en from '../i18n/en.json'
import { apiGet } from './api.js'

const KEY = 'oriel.locale'

// The static demo (GitHub Pages) has no backend to proxy catalogs, so it reads
// them straight from the CDN the backend would otherwise fetch.
const DEMO = __ORIEL_DEMO__
const DEMO_CDN = 'https://cdn.jsdelivr.net/gh/ParadoxInfinite/oriel@main/web/src/i18n'
async function fetchJSON(url) {
  const r = await fetch(url)
  if (!r.ok) throw new Error(`HTTP ${r.status}`)
  return r.json()
}

// Locales the UI offers. English ships in the bundle; loadManifest() appends any
// others the backend reports as published. A live array so the picker updates.
export const AVAILABLE = $state([{ tag: 'en', name: 'English' }])

// Catalogs available without a network fetch. Always includes English.
const BUNDLED = { en }

// Pick the starting locale: a saved choice → the browser language (matched to an
// available tag, region-then-base) → English.
function resolveStartTag() {
  let saved = ''
  try {
    saved = localStorage.getItem(KEY) || ''
  } catch {
    /* private mode: no persisted choice */
  }
  const nav = typeof navigator !== 'undefined' ? navigator.language || '' : ''
  for (const tag of [saved, nav, nav.split('-')[0]]) {
    if (tag && AVAILABLE.some((l) => l.tag === tag)) return tag
  }
  return 'en'
}

const startTag = resolveStartTag()

// Active locale. Read `locale.catalog` / `locale.tag` inside t()/tn() so markup
// re-renders when the locale switches.
export const locale = $state({ tag: startTag, catalog: BUNDLED[startTag] || en })

// Active catalog value for a key, falling back to English (so a partial catalog
// only shows English for the keys it omits). A value is a string, or a CLDR
// plural map ({ one, other, … }) for tn().
function lookup(key) {
  const v = locale.catalog?.[key]
  return v !== undefined ? v : en[key]
}

// Substitute {placeholder} tokens from params; leaves unknown tokens untouched.
function interpolate(str, params) {
  if (!params || typeof str !== 'string') return str
  return str.replace(/\{(\w+)\}/g, (m, k) => (k in params ? String(params[k]) : m))
}

// t(key, params) → translated string with {placeholder} interpolation. Returns
// the key itself if it's missing everywhere (a visible, debuggable miss).
export function t(key, params) {
  const v = lookup(key)
  return interpolate(typeof v === 'string' ? v : key, params)
}

// tn(key, count, params) → plural-aware lookup. The catalog value is a map keyed
// by CLDR plural category; Intl.PluralRules picks the right form for the active
// locale. `count` is also exposed to interpolation as {count}.
export function tn(key, count, params) {
  const node = lookup(key)
  let str = key
  if (node && typeof node === 'object') {
    const cat = new Intl.PluralRules(locale.tag).select(count)
    str = node[cat] ?? node.other ?? key
  }
  return interpolate(str, { count, ...params })
}

// Resolve the catalog for a tag: the bundled copy if there is one, otherwise the
// backend proxy. A failed fetch yields English so the UI never lands stringless.
async function loadCatalog(tag) {
  if (BUNDLED[tag]) return BUNDLED[tag]
  try {
    const cat = DEMO ? await fetchJSON(`${DEMO_CDN}/${tag}.json`) : await apiGet(`/api/i18n/${tag}`)
    if (cat && typeof cat === 'object') return cat
  } catch {
    /* offline or not published: stay on English */
  }
  return en
}

// Pull the published-locale list from the backend and add any not already
// offered. Best-effort: with no network or no catalogs, the UI stays English.
export async function loadManifest() {
  try {
    const list = DEMO ? await fetchJSON(`${DEMO_CDN}/manifest.json`) : await apiGet('/api/i18n')
    if (!Array.isArray(list)) return
    for (const e of list) {
      if (e?.tag && !AVAILABLE.some((l) => l.tag === e.tag)) {
        AVAILABLE.push({ tag: e.tag, name: e.name || e.tag })
      }
    }
  } catch {
    /* offline or no manifest: English only */
  }
}

// Switch the active locale, persist the choice, and update <html lang>.
export async function setLocale(tag) {
  if (!AVAILABLE.some((l) => l.tag === tag)) return
  locale.catalog = await loadCatalog(tag)
  locale.tag = tag
  try {
    localStorage.setItem(KEY, tag)
  } catch {
    /* private mode: just won't persist */
  }
  if (typeof document !== 'undefined') document.documentElement.lang = tag
}

// Set <html lang> for the resolved start locale. Called once at boot.
export function initLocale() {
  if (typeof document !== 'undefined') document.documentElement.lang = locale.tag
}
