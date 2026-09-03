/**
 * Formats a timestamp for display in a Detail Workspace fact (e.g.
 * "Created", "Last Updated"). Every real domain type's timestamp fields
 * are full RFC3339 strings straight off Go's time.Time (e.g.
 * "2026-09-03T10:53:06.10142-06:00") -- `new Date(iso)` parses that
 * correctly and unambiguously, no manual splitting needed.
 */
export function formatDisplayDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}
