import type { ServiceTechnology } from './customer'

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
 *
 * Network infrastructure devices (OLT, Switch) belong to a Site, not a
 * Customer (docs/03-DOMAIN-MODEL.md section 8), so
 * `assignedCustomerId`/`assignedCustomerName`/`serviceId` are absent for
 * those -- not an empty string, genuinely undefined.
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
}
