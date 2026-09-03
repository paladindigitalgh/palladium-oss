/**
 * The AccessAttachment domain type (internal/accessattachment), matching
 * internal/accessattachment/httpapi/dto.go's accessAttachmentResponse --
 * the link between an AccessInterface and the ServiceEquipment plugged
 * into it (docs/03-DOMAIN-MODEL.md). removedAt === null means active,
 * the same convention ServiceEquipment uses (see types/serviceEquipment.ts)
 * -- an attachment is detached (removedAt/removalReason set), never
 * deleted, so it stays as history rather than being erased.
 */
export interface AccessAttachment {
  id: string
  accessInterfaceId: string
  serviceEquipmentId: string
  installedAt: string | null
  removedAt: string | null
  removalReason: string
  createdAt: string
  updatedAt: string
}
