import type { Service } from '@/types/service'
import { apiFetch, ApiError } from '@/services/api/httpClient'

/**
 * The real Service data source, replacing the mock dataset this file
 * used to read from. Mirrors customerRepository.ts's shape exactly.
 */

interface ServiceDto {
  id: string
  location_id: string
  product_id: string
  service_profile_id: string
  status: Service['status']
  description: string
  activated_at: string | null
  suspended_at: string | null
  disconnected_at: string | null
  created_at: string
  updated_at: string
}

function fromDto(dto: ServiceDto): Service {
  return {
    id: dto.id,
    locationId: dto.location_id,
    productId: dto.product_id,
    serviceProfileId: dto.service_profile_id,
    status: dto.status,
    description: dto.description,
    activatedAt: dto.activated_at,
    suspendedAt: dto.suspended_at,
    disconnectedAt: dto.disconnected_at,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

export interface ServiceListQuery {
  search?: string
  status?: Service['status'] | 'all'
  sortKey?: 'id' | 'status'
  sortDirection?: 'asc' | 'desc'
  page?: number
  pageSize?: number
}

export interface ServiceListResult {
  items: Service[]
  total: number
}

export async function listAllServices(): Promise<Service[]> {
  const { services } = await apiFetch<{ services: ServiceDto[] }>('/services/')
  return services.map(fromDto)
}

/** Fetches every Service and applies search/filter/sort/pagination client-side. */
export async function listServices(query: ServiceListQuery = {}): Promise<ServiceListResult> {
  const { search = '', status = 'all', sortKey = 'id', sortDirection = 'asc', page = 1, pageSize = 15 } = query

  let results = await listAllServices()

  if (search.trim()) {
    const needle = search.trim().toLowerCase()
    results = results.filter((service) => service.id.toLowerCase().includes(needle) || service.description.toLowerCase().includes(needle))
  }
  if (status !== 'all') results = results.filter((service) => service.status === status)

  const direction = sortDirection === 'desc' ? -1 : 1
  results = results
    .slice()
    .sort((a, b) => (sortKey === 'status' ? a.status.localeCompare(b.status) : a.id.localeCompare(b.id)) * direction)

  const total = results.length
  const start = (page - 1) * pageSize
  return { items: results.slice(start, start + pageSize), total }
}

export async function listServicesByLocationIds(locationIds: string[]): Promise<Service[]> {
  const ids = new Set(locationIds)
  const services = await listAllServices()
  return services.filter((service) => ids.has(service.locationId))
}

/** Fetches a single Service, returning null (not throwing) when it does not exist. */
export async function getServiceById(id: string): Promise<Service | null> {
  try {
    const dto = await apiFetch<ServiceDto>(`/services/${id}`)
    return fromDto(dto)
  } catch (err) {
    if (err instanceof ApiError && err.kind === 'not_found') return null
    throw err
  }
}

export interface CreateServiceInput {
  locationId: string
  productId: string
  serviceProfileId: string
  status: Service['status']
  description: string
}

export async function createService(input: CreateServiceInput): Promise<Service> {
  const dto = await apiFetch<ServiceDto>('/services/', {
    method: 'POST',
    body: {
      location_id: input.locationId,
      product_id: input.productId,
      service_profile_id: input.serviceProfileId,
      status: input.status,
      description: input.description,
      activated_at: null,
      suspended_at: null,
      disconnected_at: null,
    },
  })
  return fromDto(dto)
}

/**
 * Deletes the Service identified by id. services.id is referenced by
 * service_equipment.service_id and workflow_instances.service_id, both
 * ON DELETE RESTRICT, so this throws an ApiError with kind "conflict" if
 * the service still has equipment assigned or any workflow history.
 */
export async function deleteService(id: string): Promise<void> {
  await apiFetch<void>(`/services/${id}`, { method: 'DELETE' })
}
