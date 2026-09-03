import { describe, it, expect, vi, beforeEach } from 'vitest'
import { apiFetch, ApiError } from './httpClient'

/**
 * httpClient.ts is the one seam every repository in this codebase talks
 * through (see its own doc comment) -- a bug here would silently affect
 * every single request, so unlike a normal *Repository.ts this mocks
 * `fetch` itself, one level lower than everywhere else mocks apiFetch.
 */
const { getToken, clearToken } = vi.hoisted(() => ({ getToken: vi.fn(), clearToken: vi.fn() }))

vi.mock('@/composables/useAuth', () => ({ getToken, clearToken }))

function jsonResponse(status: number, body: unknown) {
  return {
    status,
    ok: status >= 200 && status < 300,
    json: () => Promise.resolve(body),
  } as Response
}

beforeEach(() => {
  getToken.mockReset()
  clearToken.mockReset()
  vi.stubGlobal('fetch', vi.fn())
})

describe('request construction', () => {
  it('defaults to GET with no body', async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse(200, { ok: true }))

    await apiFetch('/customers/')

    const [, init] = vi.mocked(fetch).mock.calls[0]
    expect(init?.method).toBe('GET')
    expect(init?.body).toBeUndefined()
  })

  it('appends the given path to the configured API base URL', async () => {
    // Not hardcoded to the fallback default: VITE_API_URL is
    // machine-specific (see frontend/.env.local), so this only checks
    // that the path was appended, not what the base itself resolves to.
    vi.mocked(fetch).mockResolvedValue(jsonResponse(200, {}))

    await apiFetch('/customers/abc')

    const [url] = vi.mocked(fetch).mock.calls[0]
    expect(url as string).toMatch(/\/customers\/abc$/)
  })

  it('JSON-encodes a given body and sets the method', async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse(201, {}))

    await apiFetch('/customers/', { method: 'POST', body: { name: 'Acme' } })

    const [, init] = vi.mocked(fetch).mock.calls[0]
    expect(init?.method).toBe('POST')
    expect(init?.body).toBe(JSON.stringify({ name: 'Acme' }))
  })

  it('attaches an Authorization header when a token is present', async () => {
    getToken.mockReturnValue('token-123')
    vi.mocked(fetch).mockResolvedValue(jsonResponse(200, {}))

    await apiFetch('/customers/')

    const [, init] = vi.mocked(fetch).mock.calls[0]
    expect((init?.headers as Record<string, string>).Authorization).toBe('Bearer token-123')
  })

  it('sends no Authorization header when there is no stored token', async () => {
    getToken.mockReturnValue(null)
    vi.mocked(fetch).mockResolvedValue(jsonResponse(200, {}))

    await apiFetch('/customers/')

    const [, init] = vi.mocked(fetch).mock.calls[0]
    expect((init?.headers as Record<string, string>).Authorization).toBeUndefined()
  })

  it('skips the Authorization header entirely when skipAuth is set, even with a token available', async () => {
    getToken.mockReturnValue('token-123')
    vi.mocked(fetch).mockResolvedValue(jsonResponse(200, {}))

    await apiFetch('/auth/login', { method: 'POST', body: {}, skipAuth: true })

    expect(getToken).not.toHaveBeenCalled()
    const [, init] = vi.mocked(fetch).mock.calls[0]
    expect((init?.headers as Record<string, string>).Authorization).toBeUndefined()
  })
})

describe('response handling', () => {
  it('returns the decoded JSON body on success', async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse(200, { id: 'c1', name: 'Acme' }))

    const result = await apiFetch<{ id: string; name: string }>('/customers/c1')

    expect(result).toEqual({ id: 'c1', name: 'Acme' })
  })

  it('returns undefined for a 204 No Content response, without attempting to parse a body', async () => {
    const response = jsonResponse(204, null)
    const jsonSpy = vi.spyOn(response, 'json')
    vi.mocked(fetch).mockResolvedValue(response)

    const result = await apiFetch<void>('/customers/c1', { method: 'DELETE' })

    expect(result).toBeUndefined()
    expect(jsonSpy).not.toHaveBeenCalled()
  })

  it('clears the stored token when the response is 401', async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse(401, { error: 'expired', kind: 'unauthenticated' }))

    await apiFetch('/customers/').catch(() => {})

    expect(clearToken).toHaveBeenCalledOnce()
  })

  it('does not clear the token on a non-401 response', async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse(200, {}))

    await apiFetch('/customers/')

    expect(clearToken).not.toHaveBeenCalled()
  })
})

describe('error handling', () => {
  it('throws an ApiError carrying the backend error message, kind, and status', async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse(404, { error: 'customer not found', kind: 'not_found' }))

    const failure = await apiFetch('/customers/missing').catch((err: unknown) => err)

    expect(failure).toBeInstanceOf(ApiError)
    expect(failure).toMatchObject({ message: 'customer not found', kind: 'not_found', status: 404 })
  })

  it('falls back to generic message/kind when the error response has no parseable JSON body', async () => {
    const response = {
      status: 500,
      ok: false,
      json: () => Promise.reject(new Error('not JSON')),
    } as Response
    vi.mocked(fetch).mockResolvedValue(response)

    const failure = await apiFetch('/customers/').catch((err: unknown) => err)

    expect(failure).toBeInstanceOf(ApiError)
    expect(failure).toMatchObject({ message: 'Request failed', kind: 'unknown', status: 500 })
  })
})
