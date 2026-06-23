import { discovery, setAlias } from './discovery.svelte.js'

// Headless controller for the "rename in Oriel" flow. An alias is a display-only
// label keyed by the real project name, so one instance works for any stack
// (running or discovered). Editions/themes bind markup; the config read/write
// lives here so nobody reimplements it.
export class AliasEditor {
  editing = $state(null) // project name being edited, or null
  draft = $state('')

  // What to show for a project: its alias if set, otherwise the real name.
  display(name) {
    return discovery.config.aliases?.[name] || name
  }
  start(name) {
    this.editing = name
    this.draft = discovery.config.aliases?.[name] || ''
  }
  cancel() {
    this.editing = null
  }
  save() {
    setAlias(this.editing, this.draft)
    this.editing = null
  }
}
