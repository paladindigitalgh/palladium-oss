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
      products: [
        {
          id: 'p1',
          catalog_id: 'c1',
          provider_id: 'pv1',
          name: 'Fiber 1G',
          category: 'Internet',
          service_type: 'Residential',
          status: 'Active',
        },
      ],
    })

    const result = await listProducts()

    expect(apiFetch).toHaveBeenCalledWith('/products/')
    expect(result).toEqual([
      {
        id: 'p1',
        catalogId: 'c1',
        providerId: 'pv1',
        name: 'Fiber 1G',
        category: 'Internet',
        serviceType: 'Residential',
        status: 'Active',
      },
    ])
  })
})

describe('createProduct', () => {
  it('sends the request body in the API wire shape, always as Active', async () => {
    apiFetch.mockResolvedValue({
      id: 'new',
      catalog_id: 'c1',
      provider_id: 'pv1',
      name: 'Fiber 500M',
      category: 'Internet',
      service_type: 'Business',
      status: 'Active',
    })

    const result = await createProduct({
      catalogId: 'c1',
      providerId: 'pv1',
      name: 'Fiber 500M',
      category: 'Internet',
      serviceType: 'Business',
    })

    expect(apiFetch).toHaveBeenCalledWith('/products/', {
      method: 'POST',
      body: {
        catalog_id: 'c1',
        provider_id: 'pv1',
        name: 'Fiber 500M',
        category: 'Internet',
        service_type: 'Business',
        status: 'Active',
      },
    })
    expect(result.id).toBe('new')
  })
})
