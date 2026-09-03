/**
 * The Building domain type (internal/inventory), matching
 * internal/inventory/httpapi/dto.go's buildingResponse -- one level
 * below Site in the Inventory hierarchy. No status field, same as Site.
 */
export interface Building {
  id: string
  name: string
  description: string
  siteId: string
  createdAt: string
  updatedAt: string
}
