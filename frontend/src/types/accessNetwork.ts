/**
 * The AccessNetwork domain type (internal/accessnetwork), matching
 * internal/accessnetwork/httpapi/dto.go's accessNetworkResponse -- the
 * root of the access-network hierarchy (AccessNetwork -> OLT -> PONPort
 * -> AccessInterface -> AccessAttachment, docs/03-DOMAIN-MODEL.md).
 */
export type AccessNetworkStatus = 'Active' | 'Inactive'

export interface AccessNetwork {
  id: string
  name: string
  status: AccessNetworkStatus
  description: string
  createdAt: string
  updatedAt: string
}
