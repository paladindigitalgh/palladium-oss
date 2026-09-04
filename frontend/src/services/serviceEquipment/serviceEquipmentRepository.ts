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

export interface CreateServiceEquipmentInput {
  serviceId: string
  deviceId: string
  role: ServiceEquipment['role']
  description: string
}

/**
 * Assigns a Device to a Service (docs/03-DOMAIN-MODEL.md section 7).
 * installedAt is always sent as "now" and removedAt as null -- this
 * dialog only creates a fresh, currently-active assignment; backdating
 * one is not a flow this form supports. The backend rejects a Device
 * that already has an active assignment elsewhere (see
 * internal/serviceequipment/service's active-assignment-uniqueness
 * rule) with a conflict error.
 */
export async function createServiceEquipment(input: CreateServiceEquipmentInput): Promise<ServiceEquipment> {
  const dto = await apiFetch<ServiceEquipmentDto>('/service-equipment/', {
    method: 'POST',
    body: {
      service_id: input.serviceId,
      device_id: input.deviceId,
      role: input.role,
      description: input.description,
      installed_at: new Date().toISOString(),
      removed_at: null,
    },
  })
  return fromDto(dto)
}

/**
 * Permanently deletes a ServiceEquipment assignment. docs/03-DOMAIN-MODEL.md
 * states historical Service Equipment records are never deleted -- that
 * describes real operational history, not disposable test/demo data an
 * operator is deliberately cycling through. The backend still enforces
 * the real constraint regardless: an active AccessAttachment referencing
 * this record (see accessAttachmentRepository.ts) blocks the delete with
 * a conflict error until that attachment is removed first.
 */
export async function deleteServiceEquipment(id: string): Promise<void> {
  await apiFetch<void>(`/service-equipment/${id}`, { method: 'DELETE' })
}
