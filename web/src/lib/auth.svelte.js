// GUI auth state from GET /api/auth: with the gate on, a remote browser logs in
// once (token → session cookie); loopback is exempt. The app gates render on it.
import { apiGet, apiPost, setOnUnauthorized } from './api.js'

export const auth = $state({
  checked: false, // has the initial /api/auth check returned?
  enabled: false, // is the token gate on?
  authenticated: true, // is this client authed? (always true when the gate is off)
  localAdmin: false, // may this client change the token from here? (loopback only)
})

export async function checkAuth() {
  try {
    const d = await apiGet('/api/auth')
    auth.enabled = !!d?.enabled
    auth.authenticated = d?.authenticated ?? true
    auth.localAdmin = !!d?.localAdmin
  } catch {
    // /api/auth unreachable (e.g. the demo build): treat as open, no login.
    auth.enabled = false
    auth.authenticated = true
  }
  auth.checked = true
}

// login exchanges the token for a session cookie. Throws the server's message on
// a bad token or while rate-limited, so the screen can show it.
export async function login(token) {
  await apiPost('/api/login', { token })
  auth.authenticated = true
}

export async function logout() {
  try {
    await apiPost('/api/logout')
  } catch {
    /* clearing locally is enough even if the call fails */
  }
  auth.authenticated = false
}

// A 401 anywhere means the session lapsed; fall back to the login screen.
setOnUnauthorized(() => {
  if (auth.enabled) auth.authenticated = false
})
