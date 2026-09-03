/**
 * The Site domain type (internal/inventory), matching
 * internal/inventory/httpapi/dto.go's siteResponse -- the root of the
 * Inventory hierarchy (Site -> Building -> Room -> Rack -> Device,
 * docs/03-DOMAIN-MODEL.md). Unlike AccessNetwork/Location/Contact, Site
 * has no status field -- it is just Name/Description plus timestamps.
 */
export interface Site {
  id: string
  name: string
  description: string
  createdAt: string
  updatedAt: string
}
