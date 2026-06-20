// Tiny subsequence fuzzy scorer. Returns -1 if query chars don't all appear in
// order; otherwise a score where contiguous matches rank higher.
export function fuzzyScore(query, text) {
  const q = query.toLowerCase()
  const t = text.toLowerCase()
  if (!q) return 0
  let qi = 0
  let score = 0
  let last = -2
  for (let ti = 0; ti < t.length && qi < q.length; ti++) {
    if (t[ti] === q[qi]) {
      score += ti === last + 1 ? 2 : 1
      last = ti
      qi++
    }
  }
  return qi === q.length ? score : -1
}
