/**
 * The Customer domain type and everything it's composed of. This is the
 * shape the mock repository (services/customers/customerRepository.ts)
 * returns today and a real backend would return tomorrow.
 *
 * Follows docs/03-DOMAIN-MODEL.md, section 4 ("Customers do not directly
 * own network equipment. Equipment is associated through services") and
 * CLAUDE.md's Core Philosophy ("Never couple inventory directly to
 * customers"): equipment lives on `CustomerService.equipment`, never
 * directly on `Customer`. The Devices section on the Customer Detail
 * Workspace is a correlated *view* across a customer's services
 * (docs/02-DESIGN-PRINCIPLES.md principle 11, "Correlation Over
 * Collection"), not a separate ownership relationship.
 */
export type CustomerType = 'residential' | 'business'

export type CustomerStatus = 'active' | 'suspended' | 'pending' | 'cancelled'

export type ServiceTechnology = 'gpon' | 'xgs-pon'

export type ServiceStatus = 'active' | 'suspended' | 'pending' | 'decommissioned'

export type AssetRole = 'ONU' | 'Router'

export type AssetStatus = 'online' | 'offline' | 'unknown'

/**
 * A generic, non-vendor-specific equipment label (CLAUDE.md, "Plugin
 * Philosophy": the core system must never contain vendor-specific
 * logic).
 */
export interface CustomerAsset {
  id: string
  role: AssetRole
  model: string
  serialNumber: string
  status: AssetStatus
}

export interface CustomerService {
  id: string
  tier: string
  technology: ServiceTechnology
  status: ServiceStatus
  provisionedDate: string
  serviceAddress: string
  /** docs/03-DOMAIN-MODEL.md section 7: a Service may have one or more Equipment Assignments. */
  equipment: CustomerAsset[]
}

export interface CustomerContact {
  name: string
  role?: string
  phone: string
  email: string
}

export type AlertSeverity = 'critical' | 'warning' | 'info'

export interface CustomerAlert {
  id: string
  severity: AlertSeverity
  title: string
  description: string
  timestamp: string
}

/** Matches TimelineEntries.vue's entry shape exactly so both Recent Activity and Timeline can render through it. */
export interface CustomerActivityEntry {
  id: string
  label: string
  timestamp: string
  description?: string
}

export interface CustomerNote {
  id: string
  author: string
  timestamp: string
  body: string
}

export interface Customer {
  /** Customer-facing identifier, e.g. "CUST-100482". Also the route param in /customers/:id. */
  id: string
  name: string
  type: CustomerType
  status: CustomerStatus
  address: string
  city: string
  state: string
  postalCode: string
  installDate: string
  services: CustomerService[]
  contacts: {
    primary: CustomerContact
    secondary?: CustomerContact
  }
  alerts: CustomerAlert[]
  /** The most recent slice of `timeline`, not an independently maintained list. */
  activity: CustomerActivityEntry[]
  timeline: CustomerActivityEntry[]
  notes: CustomerNote[]
}
