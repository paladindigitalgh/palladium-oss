/**
 * The Service Equipment domain type (internal/serviceequipment),
 * matching internal/serviceequipment/httpapi/dto.go's
 * serviceEquipmentResponse -- the link between a Service and the Device
 * delivering it (docs/03-DOMAIN-MODEL.md section 7). Deliberately lean:
 * no vendor, telemetry, or configuration detail here -- that belongs to
 * Device itself (see types/device.ts); this record only says that a link
 * exists and what role the Device plays, never how it is configured.
 *
 * uniPort (added 2026-09-10) is the one narrow exception -- see the Go
 * type's own doc comment. It is 1 ("10GE") or 2 ("1GE") for ONU/ONT
 * equipment, meaningless for any other role.
 */
export type EquipmentRole = 'ONU' | 'Gateway' | 'Router' | 'ONT' | 'WiFiAccessPoint' | 'UPS' | 'Other'

export interface ServiceEquipment {
  id: string
  serviceId: string
  deviceId: string
  role: EquipmentRole
  uniPort: number
  installedAt: string | null
  removedAt: string | null
}

/**
 * The two LAN ports an operator can choose between for ONU/ONT
 * equipment, and the single source of truth every picker (ServiceFormDialog.vue,
 * AssignServiceEquipmentDialog.vue) draws its options from, so the
 * label-to-port mapping can never drift between them. `value` is a
 * string, not a number, because BaseSelect's v-model is always a string
 * (a native <select>'s value always is) -- convert with Number(...) at
 * the point a uniPort: number is actually sent to the API.
 */
export const UNI_PORT_OPTIONS: { value: string; label: string }[] = [
  { value: '1', label: '10GE' },
  { value: '2', label: '1GE' },
]
