/**
 * The Service Profile domain type (internal/serviceprofile), trimmed to
 * what the Service creation form's dropdown needs -- see types/product.ts
 * for the same reasoning.
 */
export interface ServiceProfile {
  id: string
  name: string
  status: 'Active' | 'Inactive'
}
