// AI activity: the tool calls an MCP client / assistant made (the operator's own
// UI clicks aren't recorded). Loaded on demand from /api/audit, newest first.
import { apiGet } from './api.js'

export const audit = $state({ entries: [], loading: false, error: '' })

export async function loadAudit() {
  audit.loading = true
  audit.error = ''
  try {
    audit.entries = (await apiGet('/api/audit')) || []
  } catch (e) {
    audit.error = e?.message || 'Could not load activity'
  }
  audit.loading = false
}
