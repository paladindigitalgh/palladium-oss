/**
 * The OLT domain type (internal/olt), matching internal/olt/httpapi/dto.go's
 * oltResponse. An OLT belongs to exactly one AccessNetwork
 * (docs/03-DOMAIN-MODEL.md) -- connectionProfileId is nullable and has no
 * picker in this workspace yet (see services/olts/oltRepository.ts),
 * always sent as null on create.
 */
export type OLTVendor = 'Kontron' | 'Nokia' | 'Calix' | 'Adtran' | 'Other'

export interface OLT {
  id: string
  accessNetworkId: string
  name: string
  vendor: OLTVendor
  model: string
  managementIpAddress: string
  connectionProfileId: string | null
  description: string
  createdAt: string
  updatedAt: string
}
