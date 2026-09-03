import type { Location } from '@/types/location'
import { apiFetch } from '@/services/api/httpClient'

/**
 * The real Location data source. GET /locations has no server-side
 * filtering (see internal/location/httpapi), so "a customer's
 * Locations" and "a service's Location" are both resolved by fetching
 * every Location once and filtering/finding client-side -- fine at this
 * dataset's size, and the same pattern customerRepository.ts uses for
 * its own list filtering.
 */

interface LocationDto {
  id: string
  customer_id: string
  name: string
  type: Location['type']
  status: Location['status']
  address1: string
  address2: string
  city: string
  state: string
  postal_code: string
  country: string
  description: string
}

function fromDto(dto: LocationDto): Location {
  return {
    id: dto.id,
    customerId: dto.customer_id,
    name: dto.name,
    type: dto.type,
    status: dto.status,
    address1: dto.address1,
    address2: dto.address2,
    city: dto.city,
    state: dto.state,
    postalCode: dto.postal_code,
    country: dto.country,
    description: dto.description,
  }
}

export async function listLocations(): Promise<Location[]> {
  const { locations } = await apiFetch<{ locations: LocationDto[] }>('/locations/')
  return locations.map(fromDto)
}

export async function listLocationsByCustomerId(customerId: string): Promise<Location[]> {
  const locations = await listLocations()
  return locations.filter((location) => location.customerId === customerId)
}

export async function getLocationById(id: string): Promise<Location | null> {
  const locations = await listLocations()
  return locations.find((location) => location.id === id) ?? null
}

export interface CreateLocationInput {
  customerId: string
  name: string
  type: Location['type']
  status: Location['status']
  address1: string
  address2: string
  city: string
  state: string
  postalCode: string
  country: string
  description: string
}

export async function createLocation(input: CreateLocationInput): Promise<Location> {
  const dto = await apiFetch<LocationDto>('/locations/', {
    method: 'POST',
    body: {
      customer_id: input.customerId,
      name: input.name,
      type: input.type,
      status: input.status,
      address1: input.address1,
      address2: input.address2,
      city: input.city,
      state: input.state,
      postal_code: input.postalCode,
      country: input.country,
      description: input.description,
    },
  })
  return fromDto(dto)
}

export interface UpdateLocationInput {
  customerId: string
  name: string
  type: Location['type']
  status: Location['status']
  address1: string
  address2: string
  city: string
  state: string
  postalCode: string
  country: string
  description: string
}

export async function updateLocation(id: string, input: UpdateLocationInput): Promise<Location> {
  const dto = await apiFetch<LocationDto>(`/locations/${id}`, {
    method: 'PUT',
    body: {
      customer_id: input.customerId,
      name: input.name,
      type: input.type,
      status: input.status,
      address1: input.address1,
      address2: input.address2,
      city: input.city,
      state: input.state,
      postal_code: input.postalCode,
      country: input.country,
      description: input.description,
    },
  })
  return fromDto(dto)
}

/**
 * Deletes the Location identified by id. locations.id is referenced by
 * services.location_id ON DELETE RESTRICT, so this throws an ApiError
 * with kind "conflict" if the location still has any Service.
 */
export async function deleteLocation(id: string): Promise<void> {
  await apiFetch<void>(`/locations/${id}`, { method: 'DELETE' })
}
