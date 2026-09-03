/**
 * The Device domain type, matching internal/inventory's real API shape
 * exactly (internal/inventory/httpapi/dto.go's deviceResponse). Unlike
 * this file's previous mock version, nothing here is fabricated to fill
 * out a richer UI: the old telemetry-shaped fields (optical power,
 * temperature, uptime, management IP/VLANs, firmware version) had no
 * backend source and never will -- CLAUDE.md is explicit that Palladium
 * is NOT a monitoring platform ("Monitoring belongs in Zabbix or other
 * monitoring systems"). A Device here is a physical inventory record:
 * what it is, where it sits in the Rack hierarchy, and its lifecycle
 * status (docs/ARCHITECTURE.md's Ordered -> ... -> Disposed lifecycle).
 * Which Service, if any, it currently fulfills is a separate concern --
 * see internal/serviceequipment and types/serviceEquipment.ts -- resolved
 * on demand by the Device Detail Workspace, never embedded here.
 */
export type DeviceStatus = 'Ordered' | 'Received' | 'InStock' | 'Installed' | 'Maintenance' | 'Retired' | 'Disposed'

export interface Device {
  id: string
  name: string
  description: string
  rackId: string | null
  manufacturer: string
  model: string
  serialNumber: string
  assetTag: string
  status: DeviceStatus
  createdAt: string
  updatedAt: string
}
