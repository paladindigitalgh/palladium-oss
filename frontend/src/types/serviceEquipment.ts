/**
 * The Service Equipment domain type (internal/serviceequipment),
 * matching internal/serviceequipment/httpapi/dto.go's
 * serviceEquipmentResponse -- the link between a Service and the Device
 * delivering it (docs/03-DOMAIN-MODEL.md section 7). Deliberately lean:
 * no vendor, telemetry, or configuration detail here -- that belongs to
 * Device itself (see types/device.ts); this record only says that a link
 * exists and what role the Device plays, never how it is configured.
 */
export type EquipmentRole = 'ONU' | 'Gateway' | 'Router' | 'ONT' | 'WiFiAccessPoint' | 'UPS' | 'Other'

export interface ServiceEquipment {
  id: string
  serviceId: string
  deviceId: string
  role: EquipmentRole
  description: string
  installedAt: string | null
  removedAt: string | null
}
