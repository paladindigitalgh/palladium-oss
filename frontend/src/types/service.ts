import type { CustomerType, ServiceStatus, ServiceTechnology } from './customer'
import type { ActivityEntry } from './activity'
import type { Note } from './note'

/**
 * The Service domain type -- the Service Collection/Detail Workspace's
 * own read model of what is nested as `CustomerService` inside
 * services/customers/customerDataset.ts's Customer -> Service -> Asset
 * tree. Mirrors exactly how types/device.ts diverges from
 * `CustomerAsset`: `CustomerService` stays the lean record Customer
 * Detail's own Services section needs (docs/03-DOMAIN-MODEL.md, "A
 * Customer owns Services"), while `Service` carries the fuller
 * provisioning/network/status detail the Service Detail Workspace needs,
 * generated once in services/services/serviceDataset.ts.
 *
 * `customerId`/`customerName`/`customerType` are joined in at generation
 * time (read off the owning Customer), never a second stored copy of
 * customer identity. Device detail is deliberately NOT carried on
 * Service -- the Devices section resolves the service's devices on
 * demand via deviceRepository.listDevicesByServiceId(), the same
 * on-demand-resolution pattern Device Detail already uses for its own
 * Assignment section.
 */
export type ServiceCategory = 'internet' | 'internet-static-ipv4' | 'internet-ipv6' | 'business-internet'

export interface Service {
  id: string
  tier: string
  technology: ServiceTechnology
  status: ServiceStatus
  category: ServiceCategory
  serviceAddress: string
  provisionedDate: string
  /** Undefined while still `provisioning` -- it hasn't gone live yet. */
  activationDate?: string

  customerId: string
  customerName: string
  customerType: CustomerType

  // Provisioning -- read-only.
  provisioningProfile: string
  bandwidthProfile: string
  authenticationProfile: string
  configurationProfile: string

  // Network -- read off the service's own ONT device where applicable,
  // so the OLT/PON/VLAN shown here always agrees with the same circuit's
  // Device Detail Workspace rather than being independently generated.
  oltId?: string
  ponPort?: string
  serviceVlan?: number
  managementVlan?: number
  ipv4Address?: string
  ipv6Address?: string
  gateway?: string

  /** "Last successful synchronization" (Status section). */
  lastSync: string

  activity: ActivityEntry[]
  timeline: ActivityEntry[]
  notes: Note[]
}
