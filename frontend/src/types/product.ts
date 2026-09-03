/**
 * The Product domain type (internal/product), trimmed to what the
 * Service creation form's dropdown needs -- an id and a display label.
 * Not full domain modeling: there is no Product Workspace yet.
 */
export interface Product {
  id: string
  name: string
  status: 'Active' | 'Retired'
}
