import { it, expect, vi, beforeEach } from 'vitest'
import { listServiceProfiles } from './serviceProfileRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', () => ({ apiFetch }))

beforeEach(() => {
  apiFetch.mockReset()
})

it('fetches from /service-profiles/ and maps the DTO fields', async () => {
  apiFetch.mockResolvedValue({ service_profiles: [{ id: 'sp1', name: 'Residential Standard', status: 'Active' }] })

  const result = await listServiceProfiles()

  expect(apiFetch).toHaveBeenCalledWith('/service-profiles/')
  expect(result).toEqual([{ id: 'sp1', name: 'Residential Standard', status: 'Active' }])
})
