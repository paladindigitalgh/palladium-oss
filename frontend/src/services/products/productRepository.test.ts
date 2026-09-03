import { it, expect, vi, beforeEach } from 'vitest'
import { listProducts } from './productRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', () => ({ apiFetch }))

beforeEach(() => {
  apiFetch.mockReset()
})

it('fetches from /products/ and maps the DTO fields', async () => {
  apiFetch.mockResolvedValue({ products: [{ id: 'p1', name: 'Fiber 1G', status: 'Active' }] })

  const result = await listProducts()

  expect(apiFetch).toHaveBeenCalledWith('/products/')
  expect(result).toEqual([{ id: 'p1', name: 'Fiber 1G', status: 'Active' }])
})
