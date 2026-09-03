/**
 * The PONPort domain type (internal/ponport), matching
 * internal/ponport/httpapi/dto.go's ponPortResponse. A PONPort belongs to
 * exactly one OLT (docs/03-DOMAIN-MODEL.md) -- there is no status field,
 * deliberately: this is a thin entity (see internal/ponport/model.go).
 */
export interface PONPort {
  id: string
  oltId: string
  portNumber: number
  description: string
  createdAt: string
  updatedAt: string
}
