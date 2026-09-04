import type { AccessAttachment } from '@/types/accessAttachment'
import { apiFetch } from '@/services/api/httpClient'

/**
 * The real AccessAttachment data source -- the leaf of the access-network
 * hierarchy, with no Detail page of its own (see AccessInterfaceDetailView.vue's
 * Attachments section). GET /access-attachments has no server-side
 * filtering, so every list below fetches the full set once and filters
 * client-side, the same pattern serviceEquipmentRepository.ts uses.
 *
 * There is no getAccessAttachmentById: an attachment is never opened on
 * its own page. During normal operation an attachment is detached (a PUT
 * setting removedAt/removalReason), never deleted -- see
 * updateAccessAttachment and DetachAccessAttachmentDialog.vue, which
 * preserve the historical row on purpose. deleteAccessAttachment exists
 * alongside that for the one case where a row should not be kept at all
 * -- disposable test/demo data an operator wants to fully remove, not a
 * real operational history -- and AccessInterfaceDetailView.vue only
 * offers it once an attachment is already detached (removedAt set), so
 * the soft path stays the default for anything still active.
 */

interface AccessAttachmentDto {
  id: string
  access_interface_id: string
  service_equipment_id: string
  installed_at: string | null
  removed_at: string | null
  removal_reason: string
  created_at: string
  updated_at: string
}

function fromDto(dto: AccessAttachmentDto): AccessAttachment {
  return {
    id: dto.id,
    accessInterfaceId: dto.access_interface_id,
    serviceEquipmentId: dto.service_equipment_id,
    installedAt: dto.installed_at,
    removedAt: dto.removed_at,
    removalReason: dto.removal_reason,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

export async function listAccessAttachments(): Promise<AccessAttachment[]> {
  const { access_attachments: accessAttachments } = await apiFetch<{ access_attachments: AccessAttachmentDto[] }>('/access-attachments/')
  return accessAttachments.map(fromDto)
}

export async function listAccessAttachmentsByAccessInterfaceId(accessInterfaceId: string): Promise<AccessAttachment[]> {
  const accessAttachments = await listAccessAttachments()
  return accessAttachments.filter((attachment) => attachment.accessInterfaceId === accessInterfaceId)
}

export async function listAccessAttachmentsByServiceEquipmentId(serviceEquipmentId: string): Promise<AccessAttachment[]> {
  const accessAttachments = await listAccessAttachments()
  return accessAttachments.filter((attachment) => attachment.serviceEquipmentId === serviceEquipmentId)
}

/** The active (not yet detached) AccessAttachment for a given ServiceEquipment, if any -- see types/accessAttachment.ts on what "active" means. */
export async function getActiveAccessAttachmentByServiceEquipmentId(serviceEquipmentId: string): Promise<AccessAttachment | null> {
  const attachments = await listAccessAttachmentsByServiceEquipmentId(serviceEquipmentId)
  return attachments.find((attachment) => attachment.removedAt === null) ?? null
}

export interface CreateAccessAttachmentInput {
  accessInterfaceId: string
  serviceEquipmentId: string
}

/** Attaches a piece of ServiceEquipment to an AccessInterface. installedAt is stamped here, not a form field -- the moment of attaching *is* the install. */
export async function createAccessAttachment(input: CreateAccessAttachmentInput): Promise<AccessAttachment> {
  const dto = await apiFetch<AccessAttachmentDto>('/access-attachments/', {
    method: 'POST',
    body: {
      access_interface_id: input.accessInterfaceId,
      service_equipment_id: input.serviceEquipmentId,
      installed_at: new Date().toISOString(),
      removed_at: null,
      removal_reason: '',
    },
  })
  return fromDto(dto)
}

export interface UpdateAccessAttachmentInput {
  accessInterfaceId: string
  serviceEquipmentId: string
  installedAt: string | null
  removedAt: string | null
  removalReason: string
}

/**
 * PUT replaces the whole row, so callers pass through every field
 * unchanged except the ones they're actually mutating -- the same rule
 * updateDevice's rackId documents. DetachAccessAttachmentDialog.vue is
 * the only caller today, passing through accessInterfaceId/
 * serviceEquipmentId/installedAt from the attachment it received and
 * setting only removedAt/removalReason.
 */
export async function updateAccessAttachment(id: string, input: UpdateAccessAttachmentInput): Promise<AccessAttachment> {
  const dto = await apiFetch<AccessAttachmentDto>(`/access-attachments/${id}`, {
    method: 'PUT',
    body: {
      access_interface_id: input.accessInterfaceId,
      service_equipment_id: input.serviceEquipmentId,
      installed_at: input.installedAt,
      removed_at: input.removedAt,
      removal_reason: input.removalReason,
    },
  })
  return fromDto(dto)
}

/**
 * Permanently deletes an AccessAttachment row -- see this file's own
 * doc comment on why this exists alongside the normal detach flow.
 */
export async function deleteAccessAttachment(id: string): Promise<void> {
  await apiFetch<void>(`/access-attachments/${id}`, { method: 'DELETE' })
}
