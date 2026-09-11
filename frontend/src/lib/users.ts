/**
 * Formats how a User should be displayed anywhere the site names who did
 * something (a Note's author, the signed-in caller in UserMenu.vue, a
 * row in AdministrationUsersView.vue's table): "First Last" if either is
 * set, or email as a fallback if neither is -- the same rule for every
 * one of those call sites, so it lives here once rather than being
 * re-implemented at each of them.
 *
 * Takes the three loose fields rather than a full User, since a Note's
 * author is a per-note snapshot (internal/note.Note's
 * AuthorFirstName/AuthorLastName/AuthorEmail), not a User record.
 */
export function formatDisplayName(input: { firstName: string; lastName: string; email: string }): string {
  const name = [input.firstName, input.lastName].filter((part) => part.trim() !== '').join(' ')
  return name !== '' ? name : input.email
}
