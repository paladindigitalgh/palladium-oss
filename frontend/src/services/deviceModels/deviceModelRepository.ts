import type { DeviceModel } from '@/types/deviceModel'
import { apiFetch } from '@/services/api/httpClient'

/**
 * The real DeviceModel data source. GET /device-models has no
 * server-side filtering (see internal/devicemodel/httpapi), the same as
 * oltModelRepository.ts's listOLTModels -- a caller narrowing to one
 * Manufacturer's Models (DeviceModelFormDialog.vue's cascading picker)
 * filters client-side.
 */

interface DeviceModelDto {
  id: string
  manufacturer_id: string
  name: string
  description: string
  is_default: boolean
  created_at: string
  updated_at: string
}

function fromDto(dto: DeviceModelDto): DeviceModel {
  return {
    id: dto.id,
    manufacturerId: dto.manufacturer_id,
    name: dto.name,
    description: dto.description,
    isDefault: dto.is_default,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

export async function listDeviceModels(): Promise<DeviceModel[]> {
  const { device_models: models } = await apiFetch<{ device_models: DeviceModelDto[] }>('/device-models/')
  return models.map(fromDto)
}

export interface CreateDeviceModelInput {
  manufacturerId: string
  name: string
  description: string
}

export async function createDeviceModel(input: CreateDeviceModelInput): Promise<DeviceModel> {
  const dto = await apiFetch<DeviceModelDto>('/device-models/', {
    method: 'POST',
    body: {
      manufacturer_id: input.manufacturerId,
      name: input.name,
      description: input.description,
    },
  })
  return fromDto(dto)
}

/**
 * Deletes the DeviceModel identified by id. device_models.id is
 * referenced by devices.device_model_id ON DELETE RESTRICT, so this
 * throws an ApiError with kind "conflict" if any Device still
 * references it.
 */
export async function deleteDeviceModel(id: string): Promise<void> {
  await apiFetch<void>(`/device-models/${id}`, { method: 'DELETE' })
}

/**
 * Sets or clears isDefault on the DeviceModel identified by id (PUT
 * /device-models/{id}/default -- a dedicated endpoint, not the general
 * update one; see internal/devicemodel/httpapi's own doc comment).
 * Setting isDefault true also clears it on whichever other DeviceModel
 * under the same ManufacturerID previously held it, server-side, in one
 * atomic statement -- the caller never needs a second call to "unset the
 * old one" first. Scoped per manufacturer, not system-wide (see
 * DeviceModel.isDefault's own doc comment).
 */
export async function setDeviceModelDefault(id: string, isDefault: boolean): Promise<DeviceModel> {
  const dto = await apiFetch<DeviceModelDto>(`/device-models/${id}/default`, {
    method: 'PUT',
    body: { is_default: isDefault },
  })
  return fromDto(dto)
}
