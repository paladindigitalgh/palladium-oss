import type { ActivityEntry } from './activity'
import type { Note } from './note'

/**
 * The mock Customer/Service/Asset vocabulary the Device Workspace's mock
 * dataset (services/devices/deviceDataset.ts) and its supporting
 * customer/service datasets (services/customers/customerDataset.ts,
 * services/services/serviceDataset.ts) are still built from.
 *
 * This file used to be types/customer.ts itself, before the Customer and
 * Service Workspaces were wired to the real backend (see types/customer.ts
 * and types/service.ts for the real shapes). Device intentionally stays
 * on mock data for now (its "device" concept blends real inventory
 * fields with live telemetry the backend has no model for), so this
 * vocabulary is preserved here, under its original names, purely to keep
 * that mock dataset internally consistent -- `Customer` here is the mock
 * dataset's Customer, not the domain's real one.
 */
export type CustomerType = 'residential' | 'business'

export type CustomerStatus = 'active' | 'suspended' | 'pending' | 'cancelled'

export type ServiceTechnology = 'gpon' | 'xgs-pon'

export type ServiceStatus = 'active' | 'provisioning' | 'suspended' | 'cancelled'

/**
 * "ONT" here (not the domain model's "ONU" -- docs/03-DOMAIN-MODEL.md
 * section 6) matches the shared Device Type vocabulary
 * (docs/09-WORKSPACE-SPECIFICATIONS.md section 10, "Device Workspace";
 * the Device Collection Workspace's own Device Type filter). Same
 * physical asset, industry-interchangeable term -- kept as one spelling
 * across the app rather than two for the same concept.
 */
export type AssetRole = 'ONT' | 'Router'

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

export interface Customer {
  /** Mock-only identifier, e.g. "CUST-100482". */
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
  activity: ActivityEntry[]
  timeline: ActivityEntry[]
  notes: Note[]
}
