/**
 * The Customer domain type. This is the shape the mock repository
 * (services/customers/customerRepository.ts) returns today and a real
 * backend would return tomorrow -- nothing in this file is specific to
 * how the data happens to be sourced right now.
 */
export type CustomerType = 'residential' | 'business'

export type CustomerStatus = 'active' | 'suspended' | 'pending' | 'cancelled'

export type ServiceTechnology = 'gpon' | 'xgs-pon'

export interface CustomerService {
  tier: string
  technology: ServiceTechnology
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
  primaryService: CustomerService
  /**
   * A generic, non-vendor-specific equipment label (CLAUDE.md, "Plugin
   * Philosophy": the core system must never contain vendor-specific
   * logic). Not surfaced in the Collection table -- present so the
   * dataset can support a future Devices section without inventing
   * real vendor/model names in core UI code.
   */
  equipment: string
  installDate: string
}
