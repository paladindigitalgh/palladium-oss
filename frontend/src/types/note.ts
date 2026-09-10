/**
 * The Note domain type (internal/note), matching
 * internal/note/httpapi/dto.go's noteResponse -- free-text, operator-
 * authored commentary attached to a Customer, Device, or Service
 * (docs/03-DOMAIN-MODEL.md). Immutable once written: there is no
 * update/delete anywhere in this domain (see the Go package's own doc
 * comment), so unlike most domain types here there is no
 * CreateNoteInput/UpdateNoteInput pair -- only the one shape a Note ever
 * has, from creation onward.
 *
 * entityType is a loose string, not a closed union, mirroring
 * TimelineEvent's own entityType -- it identifies what a Note is about
 * without this type needing to know every entity kind that might ever
 * attach one.
 */
export interface Note {
  id: string
  entityType: string
  entityId: string
  authorUserId: string
  authorEmail: string
  body: string
  createdAt: string
}
