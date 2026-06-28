#!/usr/bin/env node
// Validates the translation catalogs in web/src/i18n against the English source
// of truth (en.json). Hard-fails on anything that would ship a broken locale:
// invalid JSON, a key that English doesn't have (a typo or a stale key), a value
// whose shape disagrees with English (a plain string vs a plural map), or a
// manifest entry with no catalog file. Incomplete translations are fine — missing
// keys fall back to English — and are reported as coverage, not failures.
import { readFileSync, readdirSync } from 'node:fs'

const dir = new URL('../src/i18n/', import.meta.url)
const read = (name) => JSON.parse(readFileSync(new URL(name, dir), 'utf8'))

const PLURAL_CATEGORIES = new Set(['zero', 'one', 'two', 'few', 'many', 'other'])
const TAG_RE = /^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$/
const isPlural = (v) => v !== null && typeof v === 'object' && !Array.isArray(v)

const errors = []
const warnings = []
const err = (m) => errors.push(m)
const warn = (m) => warnings.push(m)

let en
try {
  en = read('en.json')
} catch (e) {
  console.error('cannot read en.json:', e.message)
  process.exit(1)
}
const enKeys = Object.keys(en)

const catalogs = readdirSync(dir).filter((f) => f.endsWith('.json') && f !== 'en.json' && f !== 'manifest.json')

// manifest.json: an array of { tag, name } that must include English and may only
// list locales that actually have a catalog.
const manifestTags = new Set()
try {
  const manifest = read('manifest.json')
  if (!Array.isArray(manifest)) {
    err('manifest.json must be an array of { tag, name }')
  } else {
    for (const e of manifest) {
      if (!e || typeof e.tag !== 'string' || typeof e.name !== 'string') {
        err(`manifest.json: each entry needs a string "tag" and "name" (got ${JSON.stringify(e)})`)
        continue
      }
      manifestTags.add(e.tag)
      if (!TAG_RE.test(e.tag)) err(`manifest.json: "${e.tag}" is not a valid locale tag`)
      if (e.tag !== 'en' && !catalogs.includes(`${e.tag}.json`)) {
        err(`manifest.json lists "${e.tag}" but web/src/i18n/${e.tag}.json is missing`)
      }
    }
    if (!manifestTags.has('en')) err('manifest.json must list "en"')
  }
} catch (e) {
  err(`manifest.json: ${e.message}`)
}

function checkPlural(tag, key, v) {
  if (!isPlural(v)) {
    err(`${tag}: "${key}" must be a plural object (English uses one), e.g. { "one": …, "other": … }`)
    return
  }
  for (const cat of Object.keys(v)) {
    if (!PLURAL_CATEGORIES.has(cat)) err(`${tag}: "${key}" has an invalid plural category "${cat}"`)
    else if (typeof v[cat] !== 'string') err(`${tag}: "${key}.${cat}" must be a string`)
  }
  if (!('other' in v)) err(`${tag}: "${key}" is missing the required "other" plural form`)
}

for (const file of catalogs) {
  const tag = file.replace(/\.json$/, '')
  if (!TAG_RE.test(tag)) warn(`${file}: filename is not a valid locale tag`)
  if (!manifestTags.has(tag)) warn(`${tag}.json exists but isn't listed in manifest.json, so it won't be offered`)

  let cat
  try {
    cat = read(file)
  } catch (e) {
    err(`${file}: invalid JSON (${e.message})`)
    continue
  }
  if (!isPlural(cat)) {
    err(`${file}: must be a JSON object of key → value`)
    continue
  }
  for (const [key, v] of Object.entries(cat)) {
    if (!(key in en)) {
      err(`${tag}: unknown key "${key}" — not in en.json (a typo, or a key that was removed)`)
      continue
    }
    if (isPlural(en[key])) checkPlural(tag, key, v)
    else if (typeof v !== 'string') err(`${tag}: "${key}" must be a string, like English`)
  }
  const done = Object.keys(cat).filter((k) => k in en).length
  console.log(`${tag}: ${done}/${enKeys.length} keys translated (${Math.round((done / enKeys.length) * 100)}%)`)
}

if (!catalogs.length) console.log('only English so far — no other catalogs to check')
for (const w of warnings) console.warn(`warning: ${w}`)

if (errors.length) {
  for (const e of errors) console.error(`error: ${e}`)
  console.error(`\n${errors.length} error(s) — fix before merging.`)
  process.exit(1)
}
console.log(`\nok — ${enKeys.length} English keys, ${catalogs.length} other catalog(s), no errors.`)
