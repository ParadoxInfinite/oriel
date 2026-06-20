// Shared control styles: the `.pop` spring + a colourful pill per tone — coloured
// at rest, brighter with a glow on hover.
const base = 'pop rounded-md border px-2.5 py-1 text-xs font-medium'

const tones = {
  accent: 'border-accent/35 bg-accent/10 text-accent hover:border-accent/70 hover:bg-accent/20 hover:shadow-[0_0_16px_-5px_var(--color-accent)]',
  ok: 'border-ok/35 bg-ok/10 text-ok hover:border-ok/70 hover:bg-ok/20 hover:shadow-[0_0_16px_-5px_var(--color-ok)]',
  warn: 'border-warn/35 bg-warn/10 text-warn hover:border-warn/70 hover:bg-warn/20 hover:shadow-[0_0_16px_-5px_var(--color-warn)]',
  danger: 'border-danger/35 bg-danger/10 text-danger hover:border-danger/70 hover:bg-danger/20 hover:shadow-[0_0_16px_-5px_var(--color-danger)]',
}

// action('ok'|'warn'|'accent'|'danger') → a colourful, lively button.
export const action = (tone) => `${base} ${tones[tone] ?? tones.accent}`

export const btn = action('accent')
export const btnDanger = action('danger')

export const btnPrimary =
  'pop rounded-[--radius] bg-accent px-3 py-1.5 text-sm font-medium text-accent-fg shadow-[0_2px_12px_-4px_var(--color-accent)] hover:shadow-[0_6px_22px_-4px_var(--color-accent)]'
