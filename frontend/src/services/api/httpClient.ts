/**
 * The one place this frontend talks to the real Palladium API. Every
 * `*Repository.ts` module (customers, services, events, auth, ...)
 * builds its requests on top of `apiFetch` rather than calling `fetch`
 * directly, so the base URL, auth header, and the backend's error shape
 * are handled in exactly one place.
 */
import { getToken, clearToken } from '@/composables/useAuth'

const BASE_URL = (import.meta.env.VITE_API_URL as string | undefined) ?? 'http://localhost:8080/api/v1'

/** The JSON error shape every Palladium API error response uses (see internal/httpx.WriteError). */
interface ApiErrorBody {
  error: string
  kind: string
}

export class ApiError extends Error {
  readonly kind: string
  readonly status: number

  constructor(message: string, kind: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.kind = kind
    this.status = status
  }
}

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
  body?: unknown
  /** Skip attaching Authorization -- only /auth/login needs this. */
  skipAuth?: boolean
}

/**
 * Performs a request against the Palladium API and decodes the JSON
 * response. Throws ApiError for any non-2xx response, and clears the
 * stored token on a 401 so a stale/expired session does not keep
 * silently failing every subsequent request -- the next protected route
 * navigation redirects to /login instead (see router/index.ts's guard).
 */
export async function apiFetch<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }

  if (!options.skipAuth) {
    const token = getToken()
    if (token) headers.Authorization = `Bearer ${token}`
  }

  const response = await fetch(`${BASE_URL}${path}`, {
    method: options.method ?? 'GET',
    headers,
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
  })

  if (response.status === 401) {
    clearToken()
  }

  if (response.status === 204) {
    return undefined as T
  }

  const payload = await response.json().catch(() => null)

  if (!response.ok) {
    const body = payload as ApiErrorBody | null
    throw new ApiError(body?.error ?? 'Request failed', body?.kind ?? 'unknown', response.status)
  }

  return payload as T
}
