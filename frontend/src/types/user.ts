/**
 * The User domain type (internal/auth): a platform login identity, with
 * a Role (what they may do) and a Status (whether they may authenticate
 * at all). Deactivating (see the Users panel in AdministrationView.vue)
 * never deletes a User -- both events and workflow instances can
 * reference one -- it only flips Status to 'Inactive'.
 */
export interface User {
  id: string
  email: string
  role: 'Administrator' | 'Operator' | 'Viewer'
  status: 'Active' | 'Inactive'
  createdAt: string
}
