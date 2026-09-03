/**
 * The AccessInterface domain type (internal/accessinterface), matching
 * internal/accessinterface/httpapi/dto.go's accessInterfaceResponse. An
 * AccessInterface belongs to exactly one PONPort (docs/03-DOMAIN-MODEL.md).
 */
export type AccessInterfaceTechnology = 'GPON' | 'XGSPON' | 'ActiveEthernet' | 'Other'

export type AccessInterfaceStatus = 'Active' | 'Disabled'

export interface AccessInterface {
  id: string
  ponPortId: string
  technology: AccessInterfaceTechnology
  name: string
  status: AccessInterfaceStatus
  description: string
  createdAt: string
  updatedAt: string
}
