import { apiGet } from './api.js'

// Stores for the image/volume/network lists. Refreshed on demand and by the
// live event stream (see live.svelte.js).
export const images = $state({ list: [], loading: false, error: null })
export const volumes = $state({ list: [], loading: false, error: null })
export const networks = $state({ list: [], loading: false, error: null })

async function load(store, path) {
  store.loading = true
  try {
    store.list = await apiGet(path)
    store.error = null
  } catch (e) {
    store.error = e.message
  } finally {
    store.loading = false
  }
}

export const refreshImages = () => load(images, '/api/images')
export const refreshVolumes = () => load(volumes, '/api/volumes')
export const refreshNetworks = () => load(networks, '/api/networks')
