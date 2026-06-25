// Non-loopback Hosts allowed to reach /api, the opt-in for private-network access.
import { apiGet, apiPut } from './api.js'

export const remote = $state({ hosts: [], loaded: false, saving: false, error: '' })

export async function loadRemote() {
  try {
    const d = await apiGet('/api/remote')
    remote.hosts = d.hosts || []
  } catch {
    /* not reachable, leave empty */
  } finally {
    remote.loaded = true
  }
}

async function save(hosts) {
  remote.saving = true
  remote.error = ''
  try {
    const d = await apiPut('/api/remote', { hosts })
    remote.hosts = d.hosts || []
  } catch (e) {
    remote.error = e.message
  } finally {
    remote.saving = false
  }
}

export const addRemoteHost = (h) => save([...remote.hosts, h])
export const removeRemoteHost = (h) => save(remote.hosts.filter((x) => x !== h))

// The "add a host" input both editions render: a draft plus the trim-and-clear
// submit. Editions supply only the markup bound to `draft` and `add`.
export class RemoteHostForm {
  draft = $state('')
  add() {
    const h = this.draft.trim()
    if (h) {
      addRemoteHost(h)
      this.draft = ''
    }
  }
}
