import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listProducts, createProduct } from './productRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', () => ({ apiFetch }))

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listProducts', () => {
  it('fetches from /products/ and maps the DTO fields', async () => {
    apiFetch.mockResolvedValue({
      products: [{ id: 'p1', catalog_id: 'c1', name: 'Fiber 1G', category: 'Internet', status: 'Active', description: '' }],
    })

    const result = await listProducts()

    expect(apiFetch).toHaveBeenCalledWith('/products/')
    expect(result).toEqual([{ id: 'p1', catalogId: 'c1', name: 'Fiber 1G', category: 'Internet', status: 'Active', description: '' }])
  })
})

describe('createProduct', () => {
  it('sends the request body in the API wire shape, always as Active', async () => {
    apiFetch.mockResolvedValue({ id: 'new', catalog_id: 'c1', name: 'Fiber 500M', category: 'Internet', status: 'Active', description: '500 Mbps' })

    const result = await createProduct({ catalogId: 'c1', name: 'Fiber 500M', category: 'Internet', description: '500 Mbps' })

    expect(apiFetch).toHaveBeenCalledWith('/products/', {
      method: 'POST',
      body: {
        catalog_id: 'c1',
        name: 'Fiber 500M',
        category: 'Internet',
        status: 'Active',
        description: '500 Mbps',
      },
    })
    expect(result.id).toBe('new')
  })
})
