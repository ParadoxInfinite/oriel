import { provider, checkProvider, setProvider, resolveText } from './provider.svelte.js'
import { toast } from './toast.svelte.js'

// Headless controller for the Settings "AI / natural language" block: the
// resolver-URL form and the test box. Shared so both editions only supply markup.
export class ProviderSettings {
  urlDraft = $state('')
  testText = $state('')
  testBusy = $state(false)
  testErr = $state('')
  testResult = $state(null)

  async load() {
    await checkProvider()
    this.urlDraft = provider.url
  }

  async save() {
    try {
      await setProvider(this.urlDraft.trim())
      toast(this.urlDraft.trim() ? 'Provider connected' : 'Provider disabled', 'ok')
    } catch (e) {
      toast(e.message, 'error')
    }
  }

  async runTest() {
    if (!this.testText.trim()) return
    this.testBusy = true
    this.testErr = ''
    this.testResult = null
    try {
      this.testResult = await resolveText(this.testText.trim())
    } catch (e) {
      this.testErr = e.message
    } finally {
      this.testBusy = false
    }
  }
}
