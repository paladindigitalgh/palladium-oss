/**
 * The OLT domain type (internal/olt), matching internal/olt/httpapi/dto.go's
 * oltResponse. An OLT belongs to exactly one AccessNetwork
 * (docs/03-DOMAIN-MODEL.md) -- connectionProfileId is nullable and has no
 * picker in this workspace yet (see services/olts/oltRepository.ts),
 * always sent as null on create.
 *
 * oltModelId replaced this type's former vendor/model string fields:
 * both moved to the OLTModel catalog (see @/types/oltModel) an OLT now
 * references instead of naming its chassis type inline.
 */
export interface OLT {
  id: string
  accessNetworkId: string
  name: string
  oltModelId: string
  managementIpAddress: string
  connectionProfileId: string | null
  description: string
  createdAt: string
  updatedAt: string
}
