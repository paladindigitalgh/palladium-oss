/**
 * The Event domain type (internal/event), matching
 * internal/event/httpapi/dto.go's eventResponse -- the real, immutable
 * operational history behind every Timeline section
 * (docs/02-DESIGN-PRINCIPLES.md principle 10). Named TimelineEvent, not
 * Event, only to avoid shadowing the DOM's global Event type.
 */
export interface TimelineEvent {
  id: string
  entityType: string
  entityId: string
  type: string
  message: string
  metadata: Record<string, unknown> | null
  actorUserId: string | null
  createdAt: string
}
