/**
 * The Rack domain type (internal/inventory), matching
 * internal/inventory/httpapi/dto.go's rackResponse -- the bottom of the
 * Inventory hierarchy, one level below Room. Unlike Site/Building/Room,
 * roomId is nullable: a Rack can exist unassigned to any Room (see
 * internal/inventory/model.go's Rack.Validate doc comment).
 */
export interface Rack {
  id: string
  name: string
  description: string
  roomId: string | null
  createdAt: string
  updatedAt: string
}
