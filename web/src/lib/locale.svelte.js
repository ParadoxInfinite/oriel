// Translation layer. English is bundled and is always the fallback, so a missing
// key in any other catalog shows English rather than breaking. The chosen locale
// is a per-browser preference, persisted to localStorage like the theme.
import en from '../i18n/en.json'

const KEY = 'oriel.locale'

// Locales the UI offers; more are appended here as their catalogs become available.
export const AVAILABLE = [{ tag: 'en', name: 'English' }]

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

// Resolve the catalog for a tag. Only English is bundled today; other locales
// resolve here when added.
async function loadCatalog(tag) {
  if (BUNDLED[tag]) return BUNDLED[tag]
  return en
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
