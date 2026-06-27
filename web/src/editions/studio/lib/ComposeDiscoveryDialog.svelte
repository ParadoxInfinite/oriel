<script>
  import { discovery, ensureDiscovery, updateRoot, removeRoot, rootResult, setFilter, removePattern, FILTER_MODES, DiscoveryForm, registerEscape, trapFocus } from '../../../platform/index.js'
  import Icon from './Icon.svelte'
  import PathInput from './PathInput.svelte'

  let { onClose } = $props()

  ensureDiscovery()
  const df = new DiscoveryForm()
  $effect(() => registerEscape(() => onClose()))
</script>

<div class="fixed inset-0 z-[70] flex items-start justify-center bg-black/45 p-4 pt-[8vh] backdrop-blur-sm" role="presentation" onclick={(e) => e.target === e.currentTarget && onClose()}>
  <div class="flex max-h-[84vh] w-full max-w-2xl flex-col overflow-hidden rounded-xl border border-[var(--border)] bg-[var(--panel)] shadow-[var(--shadow-lg)]" role="dialog" aria-modal="true" aria-label="Compose discovery" tabindex="-1" use:trapFocus>
    <div class="flex items-center gap-2.5 border-b border-[var(--border)] px-5 py-3.5">
      <Icon name="layers" size={16} class="text-[var(--text-3)]" />
      <h2 class="text-[14px] font-semibold tracking-tight">Compose discovery</h2>
      <button class="btn btn-ghost btn-icon btn-sm ml-auto" title="Close" aria-label="Close" onclick={onClose}><Icon name="x" size={15} /></button>
    </div>

    <div class="min-h-0 flex-1 overflow-y-auto p-5">
      <p class="text-[13px] text-[var(--text-2)]">Find Docker Compose projects on disk so you can deploy them from the Stacks tab.</p>

      {#if discovery.config.roots.length}
        <div class="mt-4 flex flex-col gap-2">
          {#each discovery.config.roots as root (root.id)}
            {@const rr = rootResult(root.id)}
            <div class="flex items-center gap-3 rounded-lg border border-[var(--border)] bg-[var(--panel-2)] px-3 py-2.5">
              <input type="checkbox" checked={root.enabled} onchange={() => updateRoot(root.id, { enabled: !root.enabled })} class="h-4 w-4 shrink-0" style="accent-color:var(--accent)" title="Enabled" />
              <div class="min-w-0 flex-1">
                <div class="mono truncate text-[13px] {root.enabled ? 'text-[var(--text)]' : 'text-[var(--text-3)] line-through'}">{root.path}</div>
                <div class="mt-0.5 text-[11px]">
                  {#if rr?.error}<span class="text-[var(--red)]">{rr.error}</span>
                  {:else if root.enabled}<span class="text-[var(--text-3)]">{rr?.found ?? 0} project{(rr?.found ?? 0) === 1 ? '' : 's'}</span>
                  {:else}<span class="text-[var(--text-3)]">disabled</span>{/if}
                </div>
              </div>
              <label class="flex shrink-0 cursor-pointer items-center gap-1.5 text-[12px] text-[var(--text-2)]" title="Walk subdirectories recursively">
                <input type="checkbox" checked={root.traverse} onchange={() => updateRoot(root.id, { traverse: !root.traverse })} class="h-3.5 w-3.5" style="accent-color:var(--accent)" /> Traverse
              </label>
              <button class="btn btn-ghost btn-icon btn-sm" title="Remove" aria-label="Remove directory" onclick={() => removeRoot(root.id)}><Icon name="trash" size={14} /></button>
            </div>
          {/each}
        </div>
      {/if}

      <div class="mt-3 flex gap-2">
        <PathInput field={df.pathField} onEnter={() => df.addDir()} placeholder="Add a directory…  /Users/you/projects" />
        <button class="btn btn-default btn-sm" onclick={() => df.addDir()} disabled={!df.pathField.value.trim()}>Add</button>
      </div>
      <p class="mt-1.5 text-[11px] text-[var(--text-3)]">Turn on <span class="font-medium text-[var(--text-2)]">Traverse</span> to scan subdirectories; off treats the directory itself as one project.</p>

      <div class="mt-5 border-t border-[var(--border)] pt-4">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <span class="text-[13px] font-medium">Filter <span class="font-normal text-[var(--text-3)]">· discovered stacks only</span></span>
          <div class="seg">
            {#each FILTER_MODES as [m, label]}
              <button class="seg-btn {discovery.config.filter.mode === m ? 'on' : ''}" onclick={() => setFilter({ mode: m })}>{label}</button>
            {/each}
          </div>
        </div>
        {#if discovery.config.filter.mode !== 'off'}
          <div class="mt-3 flex flex-wrap gap-1.5">
            {#each discovery.config.filter.patterns as p (p)}
              <span class="mono inline-flex items-center gap-1 rounded-md border border-[var(--border-strong)] bg-[var(--panel-2)] px-2 py-1 text-[11.5px] text-[var(--text-2)]">{p}<button class="text-[var(--text-3)] hover:text-[var(--red)]" aria-label="Remove pattern" onclick={() => removePattern(p)}>×</button></span>
            {/each}
          </div>
          <div class="mt-2 flex gap-2">
            <input bind:value={df.pattern} placeholder="web-*  ·  My App  ·  ~/lab/**" class="input mono min-w-0 flex-1" onkeydown={(e) => e.key === 'Enter' && df.addPattern()} />
            <button class="btn btn-default btn-sm" onclick={() => df.addPattern()} disabled={!df.pattern.trim()}>Add</button>
          </div>
          <p class="mt-1.5 text-[11px] text-[var(--text-3)]">Matches a project name, its Oriel name, or a directory path (globs &amp; <span class="mono">**</span> allowed). Running stacks are never hidden.</p>
        {/if}
      </div>
    </div>

    <div class="flex justify-end border-t border-[var(--border)] px-5 py-3">
      <button class="btn btn-default btn-sm" onclick={onClose}>Done</button>
    </div>
  </div>
</div>
