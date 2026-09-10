import type { Note } from '@/types/note'
import { apiFetch } from '@/services/api/httpClient'

interface NoteDto {
  id: string
  entity_type: string
  entity_id: string
  author_user_id: string
  author_email: string
  body: string
  created_at: string
}

function fromDto(dto: NoteDto): Note {
  return {
    id: dto.id,
    entityType: dto.entity_type,
    entityId: dto.entity_id,
    authorUserId: dto.author_user_id,
    authorEmail: dto.author_email,
    body: dto.body,
    createdAt: dto.created_at,
  }
}

/**
 * Fetches every Note recorded for one entity, newest first (see GET
 * /api/v1/notes) -- the same "fetch the full set once, no server-side
 * pagination" shape eventRepository.ts's own listEvents documents for
 * itself. A Notes section paginates client-side over this array (see
 * NotesSection.vue), the same pattern useDeviceCollection.ts's
 * client-side page slicing already establishes elsewhere.
 */
export async function listNotes(entityType: string, entityId: string): Promise<Note[]> {
  const { notes } = await apiFetch<{ notes: NoteDto[] }>(
    `/notes/?entity_type=${encodeURIComponent(entityType)}&entity_id=${encodeURIComponent(entityId)}`,
  )
  return notes.map(fromDto)
}

export interface CreateNoteInput {
  entityType: string
  entityId: string
  body: string
}

/**
 * Submits a new Note. authorUserId/authorEmail are never sent by the
 * client -- the backend stamps both from the caller's authenticated JWT
 * claims (see internal/note/httpapi's Create handler), the same reason
 * this input has no author field to omit in the first place.
 */
export async function createNote(input: CreateNoteInput): Promise<Note> {
  const dto = await apiFetch<NoteDto>('/notes/', {
    method: 'POST',
    body: {
      entity_type: input.entityType,
      entity_id: input.entityId,
      body: input.body,
    },
  })
  return fromDto(dto)
}
