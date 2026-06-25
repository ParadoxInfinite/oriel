<script>
  import { invoke, refreshNetworks, registerEscape, trapFocus } from '../../../platform/index.js'
  import Icon from './Icon.svelte'

  let { onClose } = $props()

  let name = $state('')
  let driver = $state('bridge')
  let internal = $state(false)
  let busy = $state(false)
  let inputEl = $state(null)

  $effect(() => inputEl?.focus())
  $effect(() => registerEscape(() => !busy && onClose()))

  async function create() {
    const n = name.trim()
    if (!n || busy) return
    busy = true
    const ok = await invoke('network.create', { name: n, driver, internal }, { success: `Created ${n}` })
    busy = false
    if (ok !== false) {
      refreshNetworks()
      onClose()
    }
  }
</script>

<div class="fixed inset-0 z-[70] flex items-start justify-center bg-black/45 p-4 pt-[10vh] backdrop-blur-sm" role="presentation" onclick={(e) => e.target === e.currentTarget && !busy && onClose()}>
  <div class="w-full max-w-md overflow-hidden rounded-xl border border-[var(--border)] bg-[var(--panel)] shadow-[var(--shadow-lg)]" role="dialog" aria-modal="true" aria-label="Create network" tabindex="-1" use:trapFocus>
    <div class="flex items-center gap-2.5 border-b border-[var(--border)] px-5 py-3.5">
      <Icon name="network" size={16} class="text-[var(--text-3)]" />
      <h2 class="text-[14px] font-semibold tracking-tight">Create network</h2>
    </div>
    <div class="flex flex-col gap-4 p-5">
      <label class="flex flex-col gap-1.5">
        <span class="text-[12px] font-medium text-[var(--text-2)]">Name</span>
        <input bind:this={inputEl} bind:value={name} class="input" placeholder="my-network" spellcheck="false" onkeydown={(e) => e.key === 'Enter' && create()} />
      </label>
      <label class="flex flex-col gap-1.5">
        <span class="text-[12px] font-medium text-[var(--text-2)]">Driver</span>
        <select bind:value={driver} class="input cursor-pointer">
          <option value="bridge">bridge</option>
          <option value="macvlan">macvlan</option>
          <option value="ipvlan">ipvlan</option>
          <option value="overlay">overlay</option>
        </select>
      </label>
      <label class="flex items-center gap-2.5">
        <input type="checkbox" bind:checked={internal} class="h-4 w-4" style="accent-color:var(--accent)" />
        <span class="text-[13px] text-[var(--text-2)]">Internal (no external access)</span>
      </label>
    </div>
    <div class="flex justify-end gap-2 border-t border-[var(--border)] px-5 py-3">
      <button class="btn btn-default btn-sm" onclick={onClose} disabled={busy}>Cancel</button>
      <button class="btn btn-primary btn-sm" onclick={create} disabled={!name.trim() || busy}>{busy ? 'Creating…' : 'Create'}</button>
    </div>
  </div>
</div>
