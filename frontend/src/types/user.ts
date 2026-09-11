/**
 * The User domain type (internal/auth): a platform login identity, with
 * a Role (what they may do) and a Status (whether they may authenticate
 * at all). Deactivating (see the Users panel in AdministrationView.vue)
 * never deletes a User -- both events and workflow instances can
 * reference one -- it only flips Status to 'Inactive'.
 *
 * firstName/lastName are both optional -- an empty string means unset,
 * matching internal/auth.User's own "empty string means unset" shape.
 * Use formatDisplayName() (@/lib/users) rather than reading them directly
 * wherever the UI shows who a User is -- it applies the fallback to
 * email every such place on the site is expected to use consistently.
 */
export interface User {
  id: string
  email: string
  firstName: string
  lastName: string
  role: 'Administrator' | 'Operator' | 'Viewer'
  status: 'Active' | 'Inactive'
  createdAt: string
}
