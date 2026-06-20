import { apiPost } from './api.js'
import { toast } from './toast.svelte.js'

// Runs a tool via /api/invoke, toasting the outcome. `success` may be a string
// or a fn(result)=>string. Returns the tool result, or false on failure.
export async function invoke(tool, args, { success } = {}) {
  try {
    const res = await apiPost('/api/invoke', { tool, args })
    if (success) {
      toast(typeof success === 'function' ? success(res?.result) : success, 'ok')
    }
    return res?.result ?? true
  } catch (e) {
    toast(e.message, 'error')
    return false
  }
}
