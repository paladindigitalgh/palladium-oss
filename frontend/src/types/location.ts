/**
 * The Location domain type (internal/location), matching
 * internal/location/httpapi/dto.go's locationResponse. A Customer's
 * address lives here, not on Customer itself (docs/03-DOMAIN-MODEL.md
 * section 4) -- the Customer Detail Workspace resolves a customer's
 * Locations on demand rather than Customer embedding address fields.
 */
export type LocationType = 'Service' | 'Billing' | 'Office' | 'Warehouse' | 'POP' | 'DataCenter' | 'Other'

export type LocationStatus = 'Active' | 'Inactive'

export interface Location {
  id: string
  customerId: string
  name: string
  type: LocationType
  status: LocationStatus
  address1: string
  address2: string
  city: string
  state: string
  postalCode: string
  country: string
  description: string
}
