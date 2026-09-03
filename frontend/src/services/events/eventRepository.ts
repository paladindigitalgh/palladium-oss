import type { TimelineEvent } from '@/types/timelineEvent'
import { apiFetch } from '@/services/api/httpClient'

interface EventDto {
  id: string
  entity_type: string
  entity_id: string
  type: string
  message: string
  metadata: Record<string, unknown> | null
  actor_user_id: string | null
  created_at: string
}

function fromDto(dto: EventDto): TimelineEvent {
  return {
    id: dto.id,
    entityType: dto.entity_type,
    entityId: dto.entity_id,
    type: dto.type,
    message: dto.message,
    metadata: dto.metadata,
    actorUserId: dto.actor_user_id,
    createdAt: dto.created_at,
  }
}

/** Fetches every Event recorded for one entity (see GET /api/v1/events). */
export async function listEvents(entityType: string, entityId: string): Promise<TimelineEvent[]> {
  const { events } = await apiFetch<{ events: EventDto[] }>(
    `/events/?entity_type=${encodeURIComponent(entityType)}&entity_id=${encodeURIComponent(entityId)}`,
  )
  return events.map(fromDto)
}

/**
 * Fetches the `limit` most recently recorded Events across every entity,
 * newest first (see GET /api/v1/events/recent) -- the Dashboard's system-
 * wide activity feed, not a per-entity Timeline. Deliberately a separate
 * endpoint from listEvents, not the same one with entity_type/entity_id
 * omitted: /events has no unbounded mode (see internal/event/httpapi's
 * own doc comment for why), this one is bounded by design.
 */
export async function listRecentEvents(limit = 20): Promise<TimelineEvent[]> {
  const { events } = await apiFetch<{ events: EventDto[] }>(`/events/recent?limit=${limit}`)
  return events.map(fromDto)
}
