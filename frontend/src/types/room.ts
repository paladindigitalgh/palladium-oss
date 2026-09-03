/**
 * The Room domain type (internal/inventory), matching
 * internal/inventory/httpapi/dto.go's roomResponse -- one level below
 * Building in the Inventory hierarchy. No status field, same as
 * Site/Building.
 */
export interface Room {
  id: string
  name: string
  description: string
  buildingId: string
  createdAt: string
  updatedAt: string
}
