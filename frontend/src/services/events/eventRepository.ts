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
 * newest first (see GET /api/v1/events/recent) -- the Dashboard's
 * bounded system-wide activity preview, not a per-entity Timeline or the
 * full searchable history (see listAllEvents below). Deliberately a
 * separate endpoint from listEvents/listAllEvents: this one is always
 * capped, by design, regardless of how many Events actually exist.
 */
export async function listRecentEvents(limit = 20): Promise<TimelineEvent[]> {
  const { events } = await apiFetch<{ events: EventDto[] }>(`/events/recent?limit=${limit}`)
  return events.map(fromDto)
}

/**
 * Fetches every Event ever recorded, across every entity, newest first
 * (see GET /api/v1/events with neither entity_type nor entity_id set) --
 * backs the Explorer Activity page
 * (docs/09-WORKSPACE-SPECIFICATIONS.md §15). Unbounded, the same "no
 * server-side filtering, frontend paginates client-side" shape every
 * other domain repository's own list function uses in this codebase
 * (e.g. listOLTs in oltRepository.ts).
 */
export async function listAllEvents(): Promise<TimelineEvent[]> {
  const { events } = await apiFetch<{ events: EventDto[] }>('/events/')
  return events.map(fromDto)
}
