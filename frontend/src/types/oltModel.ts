/**
 * The OLT Model domain type (internal/oltmodel), matching
 * internal/oltmodel/httpapi/dto.go's oltModelResponse. An OLTModel is an
 * Administration-managed catalog entry naming a physical OLT chassis
 * type -- e.g. "Kontron C16" -- and how many PON ports it has.
 * internal/olt.OLT.OLTModelID references one of these, and creating an
 * OLT auto-creates that many PON ports (see internal/olt/service's own
 * doc comment).
 */
export type OLTVendor = 'Kontron' | 'Nokia' | 'Calix' | 'Adtran' | 'Other'

export interface OLTModel {
  id: string
  vendor: OLTVendor
  name: string
  ponPortCount: number
  description: string
  createdAt: string
  updatedAt: string
}
