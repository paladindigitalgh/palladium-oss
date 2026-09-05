import { describe, it, expect, vi, beforeEach } from 'vitest'
import { resolveServiceLabels } from './serviceLabels'
import { listProducts } from '@/services/products/productRepository'
import { listProviders } from '@/services/providers/providerRepository'
import type { Service } from '@/types/service'
import type { Product } from '@/types/product'
import type { Provider } from '@/types/provider'

vi.mock('@/services/products/productRepository', () => ({ listProducts: vi.fn() }))
vi.mock('@/services/providers/providerRepository', () => ({ listProviders: vi.fn() }))

function service(overrides: Partial<Service> = {}): Service {
  return {
    id: 's1',
    locationId: 'l1',
    productId: 'p1',
    serviceProfileId: 'sp1',
    status: 'Active',
    description: '',
    activatedAt: null,
    suspendedAt: null,
    disconnectedAt: null,
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

function product(overrides: Partial<Product> = {}): Product {
  return {
    id: 'p1',
    catalogId: 'c1',
    providerId: 'pv1',
    name: 'Residential Internet 500 Mbps',
    category: 'Internet',
    status: 'Active',
    description: '',
    ...overrides,
  }
}

function provider(overrides: Partial<Provider> = {}): Provider {
  return { id: 'pv1', name: 'Default Provider', status: 'Active', description: '', ...overrides }
}

beforeEach(() => {
  vi.mocked(listProducts).mockReset()
  vi.mocked(listProviders).mockReset()
})

describe('resolveServiceLabels', () => {
  it('labels a service with just the Product name when only one Provider exists', async () => {
    vi.mocked(listProducts).mockResolvedValue([product()])
    vi.mocked(listProviders).mockResolvedValue([provider()])

    const labels = await resolveServiceLabels([service()])

    expect(labels.get('s1')).toBe('Residential Internet 500 Mbps')
  })

  it('labels a service with "Provider > Product" once a second Provider exists', async () => {
    vi.mocked(listProducts).mockResolvedValue([product({ providerId: 'pv2' })])
    vi.mocked(listProviders).mockResolvedValue([provider({ id: 'pv1' }), provider({ id: 'pv2', name: 'Acme Fiber' })])

    const labels = await resolveServiceLabels([service()])

    expect(labels.get('s1')).toBe('Acme Fiber > Residential Internet 500 Mbps')
  })

  it('falls back to the raw id when the Product cannot be resolved', async () => {
    vi.mocked(listProducts).mockResolvedValue([])
    vi.mocked(listProviders).mockResolvedValue([provider()])

    const labels = await resolveServiceLabels([service({ id: 'orphan', productId: 'missing' })])

    expect(labels.get('orphan')).toBe('orphan')
  })

  it('resolves every service in a single pair of requests', async () => {
    vi.mocked(listProducts).mockResolvedValue([product({ id: 'p1' }), product({ id: 'p2', name: 'Fiber 100' })])
    vi.mocked(listProviders).mockResolvedValue([provider()])

    const labels = await resolveServiceLabels([service({ id: 's1', productId: 'p1' }), service({ id: 's2', productId: 'p2' })])

    expect(labels.get('s1')).toBe('Residential Internet 500 Mbps')
    expect(labels.get('s2')).toBe('Fiber 100')
    expect(listProducts).toHaveBeenCalledTimes(1)
    expect(listProviders).toHaveBeenCalledTimes(1)
  })
})
