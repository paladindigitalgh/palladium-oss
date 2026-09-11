/**
 * The Service domain type, matching internal/service's real API shape
 * exactly (internal/service/httpapi/dto.go's serviceResponse). Unlike
 * this file's previous mock version, there is no tier/technology/
 * category/network/provisioning-profile detail here -- the backend
 * Service record itself is this lean; that richer detail belongs to
 * Product and the Network domain, neither of which have a frontend read
 * model yet (Product's Service Type is the one exception -- see
 * types/product.ts's ServiceType -- reached by joining through
 * productId, never a field on Service itself). Location, Customer, and
 * Service Equipment are resolved separately (see services/locations,
 * services/customers, services/serviceEquipment), the same
 * on-demand-resolution pattern the Detail Workspace already used for its
 * mock relationships.
 */
export type ServiceStatus = 'Pending' | 'Active' | 'Suspended' | 'Disconnected'

export interface Service {
  id: string
  locationId: string
  productId: string
  status: ServiceStatus
  description: string
  activatedAt: string | null
  suspendedAt: string | null
  disconnectedAt: string | null
  createdAt: string
  updatedAt: string
}
