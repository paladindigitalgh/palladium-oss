/**
 * The Contact domain type (internal/contact), matching
 * internal/contact/httpapi/dto.go's contactResponse. Who to reach about a
 * Customer's account -- billing, technical, emergency -- lives here, not
 * on Customer itself (docs/03-DOMAIN-MODEL.md section 8), the same
 * separation Location already has: the Customer Detail Workspace
 * resolves a customer's Contacts on demand rather than Customer
 * embedding contact fields.
 */
export type ContactRole = 'Primary' | 'Billing' | 'Technical' | 'Emergency' | 'Other'

export type ContactStatus = 'Active' | 'Inactive'

export interface Contact {
  id: string
  customerId: string
  name: string
  role: ContactRole
  email: string
  phone: string
  status: ContactStatus
  description: string
}
