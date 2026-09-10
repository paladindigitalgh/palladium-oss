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
  is_default: boolean
  created_at: string
  updated_at: string
}

function fromDto(dto: DeviceManufacturerDto): DeviceManufacturer {
  return {
    id: dto.id,
    name: dto.name,
    description: dto.description,
    isDefault: dto.is_default,
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

/**
 * Sets or clears isDefault on the DeviceManufacturer identified by id
 * (PUT /device-manufacturers/{id}/default -- a dedicated endpoint, not
 * the general update one, since there is no edit flow to route through;
 * see internal/devicemanufacturer/httpapi's own doc comment). Setting
 * isDefault true also clears it on whichever other DeviceManufacturer
 * previously held it, server-side, in one atomic statement -- the
 * caller never needs a second call to "unset the old one" first.
 */
export async function setDeviceManufacturerDefault(id: string, isDefault: boolean): Promise<DeviceManufacturer> {
  const dto = await apiFetch<DeviceManufacturerDto>(`/device-manufacturers/${id}/default`, {
    method: 'PUT',
    body: { is_default: isDefault },
  })
  return fromDto(dto)
}
