import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listCatalogs } from './catalogRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', () => ({ apiFetch }))

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listCatalogs', () => {
  it('maps every catalog from the DTO', async () => {
    apiFetch.mockResolvedValue({
      catalogs: [
        { id: 'c1', name: 'Residential Internet', description: '', status: 'Active', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
        { id: 'c2', name: 'Business Internet', description: '', status: 'Active', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
      ],
    })

    const result = await listCatalogs()

    expect(apiFetch).toHaveBeenCalledWith('/catalogs/')
    expect(result).toEqual([
      { id: 'c1', name: 'Residential Internet' },
      { id: 'c2', name: 'Business Internet' },
    ])
  })
})
