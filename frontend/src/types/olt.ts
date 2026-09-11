/**
 * The OLT domain type (internal/olt), matching internal/olt/httpapi/dto.go's
 * oltResponse. OLT is the root of the Network hierarchy
 * (docs/03-DOMAIN-MODEL.md) -- connectionProfileId is nullable (see
 * @/types/connectionProfile): an OLT can exist with none set, and
 * OLTFormDialog.vue's Connection Profile picker is how one is assigned.
 *
 * oltModelId replaced this type's former vendor/model string fields:
 * both moved to the OLTModel catalog (see @/types/oltModel) an OLT now
 * references instead of naming its chassis type inline.
 */
export interface OLT {
  id: string
  name: string
  oltModelId: string
  managementIpAddress: string
  connectionProfileId: string | null
  description: string
  createdAt: string
  updatedAt: string
}
