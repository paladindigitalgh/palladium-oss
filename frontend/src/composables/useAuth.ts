import { ref, computed } from 'vue'

/**
 * Authentication is App-owned state (docs/11-COMPONENT-ARCHITECTURE.md,
 * "State Ownership" table), shared by every component -- the same
 * module-scope-singleton pattern useTheme.ts and useSidebar.ts already
 * use for their own App-owned state.
 *
 * The token is stored in localStorage for this milestone's simplicity.
 * A future hardening pass should move to an httpOnly cookie issued by
 * the backend -- that requires backend cookie support that does not
 * exist yet, so it is out of scope here.
 */
const STORAGE_KEY = 'palladium.auth.token'

interface TokenClaims {
  user_id: string
  email: string
  exp: number
}

function decodeClaims(token: string): TokenClaims | null {
  try {
    const payload = token.split('.')[1]
    return JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')))
  } catch {
    return null
  }
}

function storedToken(): string | null {
  return localStorage.getItem(STORAGE_KEY)
}

const token = ref<string | null>(storedToken())
const claims = ref<TokenClaims | null>(token.value ? decodeClaims(token.value) : null)

/** Read directly by services/api/httpClient.ts -- not exported as part of the composable, since it is an implementation detail of request signing, not something a component should reach for. */
export function getToken(): string | null {
  return token.value
}

/** Called by services/api/httpClient.ts when a request comes back 401 (an expired or otherwise invalid token). */
export function clearToken(): void {
  token.value = null
  claims.value = null
  localStorage.removeItem(STORAGE_KEY)
}

interface LoginResponse {
  accessToken: string
}

export function useAuth() {
  const isAuthenticated = computed(() => token.value !== null)
  const email = computed(() => claims.value?.email ?? null)

  async function login(loginEmail: string, password: string): Promise<void> {
    // Imported lazily inside the function, not at module scope: httpClient.ts
    // imports getToken/clearToken from this module, so a top-level
    // import in the other direction would form a circular import.
    const { apiFetch } = await import('@/services/api/httpClient')

    const response = await apiFetch<LoginResponse>('/auth/login', {
      method: 'POST',
      body: { email: loginEmail, password },
      skipAuth: true,
    })

    token.value = response.accessToken
    claims.value = decodeClaims(response.accessToken)
    localStorage.setItem(STORAGE_KEY, response.accessToken)
  }

  function logout(): void {
    clearToken()
  }

  return { isAuthenticated, email, login, logout }
}
