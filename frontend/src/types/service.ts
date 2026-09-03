/**
 * The Service domain type, matching internal/service's real API shape
 * exactly (internal/service/httpapi/dto.go's serviceResponse). Unlike
 * this file's previous mock version, there is no tier/technology/
 * category/network/provisioning-profile detail here -- the backend
 * Service record itself is this lean; that richer detail belongs to
 * Product, ServiceProfile, and the Network domain, none of which have a
 * frontend read model yet. Location, Customer, and Service Equipment are
 * resolved separately (see services/locations, services/customers,
 * services/serviceEquipment), the same on-demand-resolution pattern the
 * Detail Workspace already used for its mock relationships.
 */
export type ServiceStatus = 'Pending' | 'Active' | 'Suspended' | 'Disconnected'

export interface Service {
  id: string
  locationId: string
  productId: string
  serviceProfileId: string
  status: ServiceStatus
  description: string
  activatedAt: string | null
  suspendedAt: string | null
  disconnectedAt: string | null
  createdAt: string
  updatedAt: string
}
