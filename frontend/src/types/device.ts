import type { ServiceTechnology } from './customer'
import type { ActivityEntry } from './activity'
import type { Note } from './note'

/**
 * The Device domain type -- the Device Collection/Detail Workspace's own
 * view of what docs/03-DOMAIN-MODEL.md calls an Asset. A `Device` is a
 * *read model*, not a second stored copy of ownership data
 * (CLAUDE.md, "Never couple inventory directly to customers"; this
 * milestone's own instruction: "Do not duplicate ownership
 * information"): `assignedCustomerId`/`assignedCustomerName` are joined
 * in by services/devices/deviceDataset.ts at generation time by reading
 * the existing Customer -> Service -> Asset tree
 * (services/customers/customerDataset.ts) -- nothing about a customer's
 * identity is stored a second time anywhere in that source tree itself.
 * Full assigned-service detail is deliberately NOT copied onto Device
 * (that would duplicate CustomerService itself) -- the Device Detail
 * Workspace resolves it on demand via `serviceId` + `assignedCustomerId`
 * (see views/DeviceDetailView.vue).
 *
 * Network infrastructure devices (OLT, Switch) belong to a Site, not a
 * Customer (docs/03-DOMAIN-MODEL.md section 8), so
 * `assignedCustomerId`/`assignedCustomerName`/`serviceId` are absent for
 * those -- not an empty string, genuinely undefined.
 *
 * Everything below `location` is device-intrinsic operational data (own
 * firmware, own telemetry, own history) generated once alongside the
 * device itself -- not read off the customer/service tree, and not
 * duplicated ownership either way.
 */
export type DeviceType = 'ONT' | 'Router' | 'Switch' | 'OLT'

export type DeviceStatus = 'online' | 'offline' | 'warning' | 'provisioning'

export interface Device {
  id: string
  model: string
  serialNumber: string
  type: DeviceType
  technology?: ServiceTechnology
  status: DeviceStatus
  location: string
  serviceId?: string
  assignedCustomerId?: string
  assignedCustomerName?: string

  // Summary
  vendor: string
  firmwareVersion: string
  installedDate: string

  // Network -- which values are present varies by device type.
  siteName: string
  oltId?: string
  ponPort?: string
  managementIp?: string
  uplinkPort?: string

  // Status -- the operational heart of the Detail Workspace.
  lastContact: string
  uptimeSeconds?: number
  opticalPowerDbm?: number
  temperatureC: number

  // Configuration -- read-only provisioning parameters.
  configProfile: string
  serviceVlan?: number
  managementVlan?: number
  configVersion: string

  activity: ActivityEntry[]
  timeline: ActivityEntry[]
  notes: Note[]
}
