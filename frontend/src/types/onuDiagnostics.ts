/**
 * Where one of a Customer's currently-attached pieces of equipment sits
 * on the access network (internal/accesstopology), matching
 * GET /diagnostics/customers/:customerId/equipment-locations's
 * "locations" entries. Not a persisted domain entity -- it is resolved
 * on demand from ServiceEquipment/AccessAttachment/AccessInterface/
 * PONPort, never stored as its own row.
 */
export interface CustomerEquipmentLocation {
  serviceEquipmentId: string
  oltId: string
  interface: string
}
