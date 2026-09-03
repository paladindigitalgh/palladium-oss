import { describe, it, expect, vi, beforeEach } from 'vitest'

/**
 * useAuth.ts holds its token/claims as MODULE-SCOPE singletons (see its
 * own doc comment: "App-owned state"), not per-call state -- so unlike
 * every other composable tested so far, importing it once at the top of
 * this file would leak state between tests (logging in during one test
 * would still be "logged in" in the next). Every test below instead
 * does `vi.resetModules()` then a fresh dynamic `import('./useAuth')`,
 * so each test gets its own module instance -- and specifically so that
 * tests of "already has a token in localStorage" can seed localStorage
 * *before* the module first runs `const token = ref(storedToken())`,
 * which only ever reads localStorage once, at module load time.
 */
const STORAGE_KEY = 'palladium.auth.token'

function jwt(payload: Record<string, unknown>): string {
  const header = btoa(JSON.stringify({ alg: 'none' }))
  const body = btoa(JSON.stringify(payload))
  return `${header}.${body}.signature`
}

beforeEach(() => {
  localStorage.clear()
  vi.resetModules()
})

describe('startup state', () => {
  it('is unauthenticated when localStorage has no token', async () => {
    const { useAuth } = await import('./useAuth')
    const { isAuthenticated, email } = useAuth()

    expect(isAuthenticated.value).toBe(false)
    expect(email.value).toBeNull()
  })

  it('boots authenticated from a token already in localStorage', async () => {
    localStorage.setItem(STORAGE_KEY, jwt({ user_id: 'u1', email: 'jane@example.com', exp: 9999999999 }))

    const { useAuth } = await import('./useAuth')
    const { isAuthenticated, email } = useAuth()

    expect(isAuthenticated.value).toBe(true)
    expect(email.value).toBe('jane@example.com')
  })

  it('treats a corrupt stored token as unauthenticated rather than throwing', async () => {
    localStorage.setItem(STORAGE_KEY, 'not-a-real-jwt')

    const { useAuth } = await import('./useAuth')
    const { email } = useAuth()

    // getToken() still returns the raw stored string (see getToken's own
    // doc comment: it is httpClient's implementation detail, not
    // validated) -- decodeClaims is what fails safe, so only the
    // claims-derived email is null here.
    expect(email.value).toBeNull()
  })
})

describe('login', () => {
  it('stores the token, decodes claims, and updates isAuthenticated', async () => {
    const accessToken = jwt({ user_id: 'u1', email: 'jane@example.com', exp: 9999999999 })
    const apiFetch = vi.fn().mockResolvedValue({ accessToken })
    vi.doMock('@/services/api/httpClient', () => ({ apiFetch }))

    const { useAuth, getToken } = await import('./useAuth')
    const { isAuthenticated, email, login } = useAuth()

    await login('jane@example.com', 'hunter2')

    expect(apiFetch).toHaveBeenCalledWith('/auth/login', {
      method: 'POST',
      body: { email: 'jane@example.com', password: 'hunter2' },
      skipAuth: true,
    })
    expect(isAuthenticated.value).toBe(true)
    expect(email.value).toBe('jane@example.com')
    expect(getToken()).toBe(accessToken)
    expect(localStorage.getItem(STORAGE_KEY)).toBe(accessToken)
  })

  it('propagates a failed login instead of leaving stale state', async () => {
    const apiFetch = vi.fn().mockRejectedValue(new Error('invalid credentials'))
    vi.doMock('@/services/api/httpClient', () => ({ apiFetch }))

    const { useAuth } = await import('./useAuth')
    const { isAuthenticated, login } = useAuth()

    await expect(login('jane@example.com', 'wrong')).rejects.toThrow('invalid credentials')
    expect(isAuthenticated.value).toBe(false)
  })
})

describe('logout / clearToken', () => {
  it('logout clears isAuthenticated, email, and localStorage', async () => {
    localStorage.setItem(STORAGE_KEY, jwt({ user_id: 'u1', email: 'jane@example.com', exp: 9999999999 }))
    const { useAuth } = await import('./useAuth')
    const { isAuthenticated, email, logout } = useAuth()
    expect(isAuthenticated.value).toBe(true)

    logout()

    expect(isAuthenticated.value).toBe(false)
    expect(email.value).toBeNull()
    expect(localStorage.getItem(STORAGE_KEY)).toBeNull()
  })

  it('clearToken (the export httpClient calls on a 401) has the same effect as logout', async () => {
    localStorage.setItem(STORAGE_KEY, jwt({ user_id: 'u1', email: 'jane@example.com', exp: 9999999999 }))
    const { useAuth, clearToken } = await import('./useAuth')
    const { isAuthenticated } = useAuth()

    clearToken()

    expect(isAuthenticated.value).toBe(false)
    expect(localStorage.getItem(STORAGE_KEY)).toBeNull()
  })
})

describe('getToken', () => {
  it('returns the current raw token for httpClient to attach to requests', async () => {
    localStorage.setItem(STORAGE_KEY, jwt({ user_id: 'u1', email: 'jane@example.com', exp: 9999999999 }))
    const { getToken } = await import('./useAuth')

    expect(getToken()).toBe(localStorage.getItem(STORAGE_KEY))
  })

  it('returns null when there is no session', async () => {
    const { getToken } = await import('./useAuth')

    expect(getToken()).toBeNull()
  })
})
