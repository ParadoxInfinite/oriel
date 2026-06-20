// Shared column-sorting state + helpers for the resource tables.
export function createSort(key, dir = 'asc') {
  const s = $state({ key, dir })
  return s
}

// Click a header: toggle direction if it's the active column, else switch to it.
export function toggleSort(s, key) {
  if (s.key === key) s.dir = s.dir === 'asc' ? 'desc' : 'asc'
  else {
    s.key = key
    s.dir = 'asc'
  }
}

// Return a new array sorted by the active column's `get` accessor. Numbers sort
// numerically; everything else compares as strings with natural numeric order.
export function sortRows(list, columns, s) {
  const col = columns.find((c) => c.key === s.key)
  if (!col?.get) return list
  const dir = s.dir === 'asc' ? 1 : -1
  return [...list].sort((a, b) => {
    const av = col.get(a)
    const bv = col.get(b)
    if (typeof av === 'number' && typeof bv === 'number') return (av - bv) * dir
    return String(av ?? '').localeCompare(String(bv ?? ''), undefined, { numeric: true }) * dir
  })
}
