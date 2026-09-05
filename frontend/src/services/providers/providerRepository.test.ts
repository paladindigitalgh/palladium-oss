import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listProviders, createProvider } from './providerRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', () => ({ apiFetch }))

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listProviders', () => {
  it('maps every provider from the DTO', async () => {
    apiFetch.mockResolvedValue({
      providers: [
        { id: 'pv1', name: 'Acme Fiber', status: 'Active', description: '' },
        { id: 'pv2', name: 'Beta Fiber', status: 'Inactive', description: 'Paused' },
      ],
    })

    const result = await listProviders()

    expect(apiFetch).toHaveBeenCalledWith('/providers/')
    expect(result).toEqual([
      { id: 'pv1', name: 'Acme Fiber', status: 'Active', description: '' },
      { id: 'pv2', name: 'Beta Fiber', status: 'Inactive', description: 'Paused' },
    ])
  })
})

describe('createProvider', () => {
  it('sends the request body in the API wire shape, always as Active', async () => {
    apiFetch.mockResolvedValue({ id: 'new', name: 'Acme Fiber', status: 'Active', description: 'Retail ISP' })

    const result = await createProvider({ name: 'Acme Fiber', description: 'Retail ISP' })

    expect(apiFetch).toHaveBeenCalledWith('/providers/', {
      method: 'POST',
      body: { name: 'Acme Fiber', status: 'Active', description: 'Retail ISP' },
    })
    expect(result.id).toBe('new')
  })
})
