import type { ServiceEquipment } from '@/types/serviceEquipment'
import { apiFetch } from '@/services/api/httpClient'

interface ServiceEquipmentDto {
  id: string
  service_id: string
  device_id: string
  role: ServiceEquipment['role']
  description: string
  installed_at: string | null
  removed_at: string | null
}

function fromDto(dto: ServiceEquipmentDto): ServiceEquipment {
  return {
    id: dto.id,
    serviceId: dto.service_id,
    deviceId: dto.device_id,
    role: dto.role,
    description: dto.description,
    installedAt: dto.installed_at,
    removedAt: dto.removed_at,
  }
}

/**
 * GET /service-equipment has no server-side filtering (see
 * internal/serviceequipment/httpapi), so every list below fetches the
 * full set once and filters client-side, the same pattern
 * locationRepository.ts uses. listServiceEquipment is the one place that
 * actually calls apiFetch -- added for AttachAccessAttachmentDialog.vue's
 * equipment picker and the Service Detail cross-link, which both need
 * every piece of equipment, not one service's or device's.
 */
export async function listServiceEquipment(): Promise<ServiceEquipment[]> {
  const { service_equipment: equipment } = await apiFetch<{ service_equipment: ServiceEquipmentDto[] }>('/service-equipment/')
  return equipment.map(fromDto)
}

export async function listServiceEquipmentByServiceId(serviceId: string): Promise<ServiceEquipment[]> {
  const equipment = await listServiceEquipment()
  return equipment.filter((item) => item.serviceId === serviceId)
}

/** Same as listServiceEquipmentByServiceId, filtered by deviceId instead -- which Service(s), if any, a given Device currently fulfills. */
export async function listServiceEquipmentByDeviceId(deviceId: string): Promise<ServiceEquipment[]> {
  const equipment = await listServiceEquipment()
  return equipment.filter((item) => item.deviceId === deviceId)
}
