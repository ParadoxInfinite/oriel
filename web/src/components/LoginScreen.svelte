<script>
  import { login } from '../lib/auth.svelte.js'

  let token = $state('')
  let error = $state('')
  let busy = $state(false)

  async function submit() {
    if (!token.trim() || busy) return
    busy = true
    error = ''
    try {
      await login(token.trim())
      // On success the app swaps this screen out; leave busy set so the button
      // doesn't flash back to "Sign in" during the transition.
    } catch (e) {
      error = e?.message || 'Sign in failed'
      busy = false
    }
  }

  function focusOnMount(node) {
    node.focus()
  }
</script>

<div class="wrap">
  <form class="card" onsubmit={(e) => { e.preventDefault(); submit() }}>
    <div class="brand">Oriel</div>
    <h1 class="title">Sign in</h1>
    <p class="sub">This Oriel is protected. Enter your access token to continue.</p>

    <input
      class="input field"
      type="password"
      autocomplete="current-password"
      placeholder="Access token"
      aria-label="Access token"
      bind:value={token}
      disabled={busy}
      use:focusOnMount
    />

    {#if error}<p class="error" role="alert">{error}</p>{/if}

    <button class="btn btn-primary submit" type="submit" disabled={busy || !token.trim()}>
      {busy ? 'Signing in…' : 'Sign in'}
    </button>

    <p class="hint">The token is set on the machine running Oriel: <span class="mono">oriel config auth-token</span>.</p>
  </form>
</div>

<style>
  .wrap {
    position: fixed;
    inset: 0;
    z-index: 100;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    background: var(--bg, #0b0b0f);
  }
  .card {
    width: 100%;
    max-width: 360px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 28px;
    background: var(--panel, #15151c);
    border: 1px solid var(--border, #2a2a33);
    border-radius: 16px;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.35);
  }
  .brand {
    font-size: 13px;
    font-weight: 700;
    letter-spacing: 0.02em;
    color: var(--accent, #5b5bd6);
  }
  .title {
    font-size: 20px;
    font-weight: 650;
    color: var(--text, #e8e8ee);
  }
  .sub {
    font-size: 13px;
    color: var(--text-3, #8a8a96);
    margin-top: -4px;
  }
  .field {
    width: 100%;
    margin-top: 8px;
  }
  .submit {
    width: 100%;
    margin-top: 2px;
  }
  .error {
    font-size: 12.5px;
    color: var(--red, #e5484d);
  }
  .hint {
    font-size: 11.5px;
    color: var(--text-3, #8a8a96);
    margin-top: 2px;
  }
  .mono {
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  }
</style>
