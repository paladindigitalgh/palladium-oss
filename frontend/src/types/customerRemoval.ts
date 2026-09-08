/**
 * "Remove Customer" preview (internal/customer/removal.Preview via
 * GET /customers/:id/removal-preview): everything Execute would do for
 * one Customer, without changing anything. See that package's own doc
 * comment for why removal never deletes a row -- Locations go Inactive,
 * Services go Disconnected, equipment gets unassigned (RemovedAt), and
 * the underlying Device is never touched.
 */
export interface CustomerRemovalEquipmentPreview {
  serviceEquipmentId: string
  deviceId: string
  deviceName: string
  role: string
  willRunOltTeardown: boolean
}

export interface CustomerRemovalServicePreview {
  serviceId: string
  description: string
  status: string
  equipment: CustomerRemovalEquipmentPreview[]
}

export interface CustomerRemovalLocationPreview {
  locationId: string
  name: string
  status: string
  services: CustomerRemovalServicePreview[]
}

export interface CustomerRemovalPreview {
  customerId: string
  locations: CustomerRemovalLocationPreview[]
}
