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
 * internal/serviceequipment/httpapi), so "a service's equipment" is
 * resolved by fetching the full list once and filtering client-side --
 * the same pattern locationRepository.ts uses.
 */
export async function listServiceEquipmentByServiceId(serviceId: string): Promise<ServiceEquipment[]> {
  const { service_equipment: equipment } = await apiFetch<{ service_equipment: ServiceEquipmentDto[] }>('/service-equipment/')
  return equipment.map(fromDto).filter((item) => item.serviceId === serviceId)
}
