import type { DeviceManufacturer } from '@/types/deviceManufacturer'
import { apiFetch } from '@/services/api/httpClient'

/**
 * The real DeviceManufacturer data source. GET /device-manufacturers has
 * no server-side filtering (see internal/devicemanufacturer/httpapi),
 * the same as oltModelRepository.ts's listOLTModels.
 */

interface DeviceManufacturerDto {
  id: string
  name: string
  description: string
  created_at: string
  updated_at: string
}

function fromDto(dto: DeviceManufacturerDto): DeviceManufacturer {
  return {
    id: dto.id,
    name: dto.name,
    description: dto.description,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

export async function listDeviceManufacturers(): Promise<DeviceManufacturer[]> {
  const { device_manufacturers: manufacturers } = await apiFetch<{ device_manufacturers: DeviceManufacturerDto[] }>(
    '/device-manufacturers/',
  )
  return manufacturers.map(fromDto)
}

export interface CreateDeviceManufacturerInput {
  name: string
  description: string
}

export async function createDeviceManufacturer(input: CreateDeviceManufacturerInput): Promise<DeviceManufacturer> {
  const dto = await apiFetch<DeviceManufacturerDto>('/device-manufacturers/', {
    method: 'POST',
    body: {
      name: input.name,
      description: input.description,
    },
  })
  return fromDto(dto)
}

/**
 * Deletes the DeviceManufacturer identified by id.
 * device_manufacturers.id is referenced by device_models.manufacturer_id
 * ON DELETE RESTRICT, so this throws an ApiError with kind "conflict" if
 * any DeviceModel still references it.
 */
export async function deleteDeviceManufacturer(id: string): Promise<void> {
  await apiFetch<void>(`/device-manufacturers/${id}`, { method: 'DELETE' })
}
