/**
 * Explorer's three curated reports (internal/report;
 * docs/09-WORKSPACE-SPECIFICATIONS.md §15), matching each report's
 * response row shape exactly. A report row is its own flat, already-
 * joined shape -- not a reuse of e.g. @/types/customer.Customer -- the
 * same "no dependency on the domains it reads from" reasoning
 * internal/report's own Go doc comment gives.
 */

/** One row of the "Customers & Contacts" report: every Customer, one row per Contact (blank contact fields when a Customer has none on file). */
export interface CustomerContactRow {
  customerId: string
  customerName: string
  customerType: string
  customerStatus: string
  contactId: string | null
  contactName: string
  contactRole: string
  contactEmail: string
  contactPhone: string
  contactStatus: string
}

/** One row of the "Devices" report: every Device, resolved Manufacturer/Model, physical location path (blank when unracked), and whichever Customer currently has it, if any. */
export interface DeviceReportRow {
  deviceId: string
  deviceName: string
  serialNumber: string
  assetTag: string
  deviceStatus: string
  manufacturer: string
  model: string
  siteName: string
  buildingName: string
  roomName: string
  rackName: string
  assignedCustomerId: string | null
  assignedCustomerName: string
}

/** One row of the "Customers & Devices" report: every active Customer-Device relationship, one row per relationship (a Device delivered to a Customer both ways at once gets two rows, not one). */
export interface CustomerDeviceRow {
  customerId: string
  customerName: string
  customerType: string
  customerStatus: string
  deviceId: string
  deviceName: string
  serialNumber: string
  manufacturer: string
  model: string
  deviceStatus: string
  relationship: 'Placement' | 'Service Equipment'
  serviceStatus: string
  locationName: string
}
